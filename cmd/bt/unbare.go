package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/amaya382/baretree/internal/git"
	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/worktree"
	"github.com/spf13/cobra"
)

var unbareCmd = &cobra.Command{
	Use:   "unbare <worktree> <destination>",
	Short: "Convert a worktree to a standalone Git repository",
	Long: `Convert a worktree to a standalone Git repository.

This creates a complete, independent Git repository from a worktree.
The new repository will have its own .git directory and can be
used without baretree.

The worktree can be specified as:
  - Branch name (e.g., feature/auth)
  - Directory name relative to repo root
  - @ for the default branch worktree

Examples:
  bt unbare feature/auth ~/repos/auth-feature
  bt unbare @ ~/repos/main-copy
  bt unbare main ../standalone-main`,
	Args: cobra.ExactArgs(2),
	RunE: runUnbare,
}

func runUnbare(cmd *cobra.Command, args []string) error {
	worktreeName := args[0]
	destination := args[1]

	// Convert destination to absolute path
	absDestination, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("failed to resolve destination path: %w", err)
	}

	// Check if destination already exists
	if _, err := os.Stat(absDestination); err == nil {
		return fmt.Errorf("destination already exists: %s", absDestination)
	}

	// Find repository root
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("not in a baretree repository: %w", err)
	}

	// Get bare repository path
	bareDir, err := repository.GetBareRepoPath(repoRoot)
	if err != nil {
		return err
	}

	// Load config and create manager
	mgr, err := repository.NewManager(repoRoot)
	if err != nil {
		return err
	}

	wtMgr := worktree.NewManager(repoRoot, bareDir, mgr.Config)

	// Resolve worktree
	worktreePath, err := wtMgr.Resolve(worktreeName)
	if err != nil {
		var ambiguousErr *worktree.AmbiguousMatchError
		if errors.As(err, &ambiguousErr) {
			fmt.Fprintf(os.Stderr, "Ambiguous worktree name '%s'. Did you mean one of these?\n\n", ambiguousErr.Name)
			for _, wt := range ambiguousErr.Matches {
				relPath, _ := filepath.Rel(ambiguousErr.RepoRoot, wt.Path)
				fmt.Fprintf(os.Stderr, "  bt unbare %s %s\n", relPath, destination)
			}
			fmt.Fprintln(os.Stderr)
			return fmt.Errorf("ambiguous worktree name")
		}
		return err
	}

	// Get branch name for the worktree
	branchName, err := wtMgr.GetBranchName(worktreePath)
	if err != nil {
		return fmt.Errorf("failed to get branch name: %w", err)
	}

	fmt.Printf("Converting worktree to standalone repository:\n")
	fmt.Printf("  Source: %s (branch: %s)\n", worktreePath, branchName)
	fmt.Printf("  Destination: %s\n\n", absDestination)

	// Clone from bare repository using --no-checkout to avoid any checkout conflicts
	fmt.Println("Cloning repository...")
	// Use git clone with local bare repository
	cloneExecutor := git.NewExecutor(filepath.Dir(absDestination))
	if _, err := cloneExecutor.Execute("clone", "--no-checkout", bareDir, absDestination); err != nil {
		os.RemoveAll(absDestination)
		return fmt.Errorf("failed to clone from bare repository: %w", err)
	}

	destExecutor := git.NewExecutor(absDestination)

	// Checkout the branch BEFORE changing remotes
	// (git checkout relies on origin/branch to create tracking branch)
	fmt.Println("Checking out branch...")
	if _, err := destExecutor.Execute("checkout", branchName); err != nil {
		os.RemoveAll(absDestination)
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	// Copy remote configuration from bare repository (replace origin which points to local bare)
	fmt.Println("Copying remote configuration...")
	bareExecutor := git.NewExecutor(bareDir)

	// First remove the local origin that was set by clone
	_, _ = destExecutor.Execute("remote", "remove", "origin")

	remotes, err := bareExecutor.Execute("remote")
	if err == nil && remotes != "" {
		for _, remote := range splitLines(remotes) {
			if remote == "" {
				continue
			}
			url, err := bareExecutor.Execute("remote", "get-url", remote)
			if err == nil && url != "" {
				if _, addErr := destExecutor.Execute("remote", "add", remote, url); addErr != nil {
					// Ignore errors for already existing remotes
					continue
				}
			}
		}
	}

	// Copy submodules if they exist
	fmt.Println("Copying submodules...")
	if err := copySubmodules(bareDir, worktreePath, absDestination); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to copy submodules: %v\n", err)
	}

	// Get the list of deleted files before copying (we need to preserve this state)
	srcExecutor := git.NewExecutor(worktreePath)
	deletedFiles := getDeletedFiles(srcExecutor)

	// Copy all working tree files (including .gitignore'd files and empty directories)
	fmt.Println("Copying working tree files...")
	if err := copyWorktreeFiles(worktreePath, absDestination); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to copy some files: %v\n", err)
	}

	// Update submodule .git files for standalone repository structure
	fmt.Println("Updating submodule paths...")
	if err := updateSubmoduleGitFilesForUnbare(absDestination); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update submodule paths: %v\n", err)
	}

	// Remove files that were deleted in the source worktree
	// (copyWorktreeFiles doesn't copy deleted files, but checkout restored them)
	for _, filename := range deletedFiles {
		dstPath := filepath.Join(absDestination, filename)
		os.Remove(dstPath)
	}

	// Preserve staging state by copying the index file
	fmt.Println("Preserving staging state...")
	if err := copyIndexFile(bareDir, worktreePath, absDestination, branchName); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to preserve staging state: %v\n", err)
	}

	// Preserve submodule staging state by re-copying index files
	// (git submodule update overwrites index files, so we need to copy them again)
	if err := copySubmoduleIndexFiles(bareDir, absDestination); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to preserve submodule staging state: %v\n", err)
	}

	fmt.Printf("\n✓ Converted successfully!\n")
	fmt.Printf("  Repository: %s\n", absDestination)
	fmt.Printf("  Branch: %s\n", branchName)

	return nil
}

// copySubmodules copies git submodule configuration and data
func copySubmodules(bareDir, worktreePath, destination string) error {
	// Check if .gitmodules exists
	gitmodulesPath := filepath.Join(worktreePath, ".gitmodules")
	if _, err := os.Stat(gitmodulesPath); os.IsNotExist(err) {
		return nil // No submodules
	}

	// Copy .git/modules directory from bare repo to destination
	srcModulesDir := filepath.Join(bareDir, "modules")
	if _, err := os.Stat(srcModulesDir); err == nil {
		dstModulesDir := filepath.Join(destination, ".git", "modules")
		if err := copyDirRecursive(srcModulesDir, dstModulesDir); err != nil {
			return fmt.Errorf("failed to copy modules directory: %w", err)
		}

		// Update gitdir paths in submodule .git files
		if err := updateSubmoduleGitdirs(destination); err != nil {
			return fmt.Errorf("failed to update submodule gitdirs: %w", err)
		}
	}

	// Initialize and update submodules in destination
	destExecutor := git.NewExecutor(destination)
	if _, err := destExecutor.Execute("submodule", "update", "--init", "--recursive"); err != nil {
		// Non-fatal, submodules might need network access
		return fmt.Errorf("failed to initialize submodules (may require network): %w", err)
	}

	return nil
}

// updateSubmoduleGitdirs updates the gitdir paths in submodule working directories
func updateSubmoduleGitdirs(repoRoot string) error {
	modulesDir := filepath.Join(repoRoot, ".git", "modules")
	if _, err := os.Stat(modulesDir); os.IsNotExist(err) {
		return nil
	}

	return filepath.Walk(modulesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "gitdir" && !info.IsDir() {
			// Read current gitdir content
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// Get the relative path from modules dir
			relPath, _ := filepath.Rel(modulesDir, filepath.Dir(path))

			// Calculate new gitdir path
			submodulePath := filepath.Join(repoRoot, relPath)
			newGitdir := submodulePath + "/.git\n"

			// Write updated gitdir
			if err := os.WriteFile(path, []byte(newGitdir), 0644); err != nil {
				return err
			}

			// Also update the .git file in the submodule working directory if it exists
			submoduleGitFile := filepath.Join(submodulePath, ".git")
			if _, err := os.Stat(submoduleGitFile); err == nil {
				moduleGitDir := filepath.Dir(path)
				newContent := fmt.Sprintf("gitdir: %s\n", moduleGitDir)
				if string(content) != newContent {
					if err := os.WriteFile(submoduleGitFile, []byte(newContent), 0644); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}

// updateSubmoduleGitFilesForUnbare updates submodule .git files for standalone repository structure
// When copying from a baretree worktree (which has deeper nesting), the relative paths need adjustment
func updateSubmoduleGitFilesForUnbare(repoRoot string) error {
	// Check if .gitmodules exists
	gitmodulesPath := filepath.Join(repoRoot, ".gitmodules")
	if _, err := os.Stat(gitmodulesPath); os.IsNotExist(err) {
		return nil // No submodules
	}

	// Walk through repository to find submodule .git files
	return filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip the repo's own .git directory
		if path == filepath.Join(repoRoot, ".git") {
			return filepath.SkipDir
		}

		// Look for .git files (not directories) in subdirectories
		if info.Name() == ".git" && !info.IsDir() {
			// Read current content
			content, err := os.ReadFile(path)
			if err != nil {
				return nil // Skip if can't read
			}

			contentStr := string(content)
			if !strings.HasPrefix(contentStr, "gitdir:") {
				return nil // Not a gitdir reference
			}

			// Extract the gitdir path
			gitdirPath := strings.TrimSpace(strings.TrimPrefix(contentStr, "gitdir:"))

			// Check if it references .git/modules
			if strings.Contains(gitdirPath, "/.git/modules/") || strings.Contains(gitdirPath, string(filepath.Separator)+".git"+string(filepath.Separator)+"modules"+string(filepath.Separator)) {
				// Calculate the correct relative path for a standard git repo structure
				submodulePath := filepath.Dir(path)
				relToRoot, err := filepath.Rel(submodulePath, repoRoot)
				if err != nil {
					return nil
				}

				// Extract module path
				modulePath := extractModulePathForUnbare(gitdirPath)
				if modulePath == "" {
					return nil
				}

				// Build new gitdir path (from submodule to repo root, then to .git/modules)
				newGitdir := filepath.Join(relToRoot, ".git", "modules", modulePath)
				newGitdir = filepath.Clean(newGitdir)

				// Write updated content
				newContent := fmt.Sprintf("gitdir: %s\n", newGitdir)
				if err := os.WriteFile(path, []byte(newContent), info.Mode()); err != nil {
					return nil // Skip if can't write
				}

				// Also update the module's config file
				moduleGitDir := filepath.Join(repoRoot, ".git", "modules", modulePath)
				_ = updateModuleWorktreePathForUnbare(moduleGitDir, submodulePath)
			}
		}

		return nil
	})
}

// extractModulePathForUnbare extracts the module path from a gitdir path
func extractModulePathForUnbare(gitdirPath string) string {
	marker := ".git/modules/"
	idx := strings.Index(gitdirPath, marker)
	if idx == -1 {
		marker = ".git\\modules\\"
		idx = strings.Index(gitdirPath, marker)
	}
	if idx == -1 {
		return ""
	}
	return gitdirPath[idx+len(marker):]
}

// updateModuleWorktreePathForUnbare updates the core.worktree setting in a submodule's config
func updateModuleWorktreePathForUnbare(moduleGitDir, submodulePath string) error {
	absSubmodulePath, err := filepath.Abs(submodulePath)
	if err != nil {
		return err
	}

	relWorktree, err := filepath.Rel(moduleGitDir, absSubmodulePath)
	if err != nil {
		return err
	}

	configFile := filepath.Join(moduleGitDir, "config")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return nil
	}

	executor := git.NewExecutor(filepath.Dir(moduleGitDir))
	_, _ = executor.Execute("config", "-f", configFile, "core.worktree", relWorktree)
	return nil
}

// copyWorktreeFiles copies all files from source worktree to destination,
// including .gitignore'd files and preserving empty directories
func copyWorktreeFiles(source, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Skip .git file/directory (worktree has .git as a file pointing to bare repo)
		if relPath == ".git" || strings.HasPrefix(relPath, ".git"+string(filepath.Separator)) {
			return nil
		}

		dstPath := filepath.Join(destination, relPath)

		// Handle directories (including empty ones)
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Handle symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			// Remove existing file if present
			os.Remove(dstPath)
			return os.Symlink(linkTarget, dstPath)
		}

		// Copy regular files
		return copyFile(path, dstPath)
	})
}

// copyIndexFile copies the index file to preserve staging state
func copyIndexFile(bareDir, worktreePath, destination, branchName string) error {
	// The index file for worktrees is in .git/worktrees/<escaped-branch>/index
	worktreeGitDirName := git.ToWorktreeGitDirName(branchName)
	worktreeGitDir := filepath.Join(bareDir, "worktrees", worktreeGitDirName)
	srcIndex := filepath.Join(worktreeGitDir, "index")

	// If worktree-specific index doesn't exist, try the main bare repo index
	if _, err := os.Stat(srcIndex); os.IsNotExist(err) {
		srcIndex = filepath.Join(bareDir, "index")
	}

	if _, err := os.Stat(srcIndex); os.IsNotExist(err) {
		return nil // No index file
	}

	dstIndex := filepath.Join(destination, ".git", "index")
	return copyFile(srcIndex, dstIndex)
}

// copySubmoduleIndexFiles copies index files for all submodules to preserve staging state.
// This is called after git submodule update --init which overwrites index files.
func copySubmoduleIndexFiles(bareDir, destination string) error {
	srcModulesDir := filepath.Join(bareDir, "modules")
	dstModulesDir := filepath.Join(destination, ".git", "modules")

	if _, err := os.Stat(srcModulesDir); os.IsNotExist(err) {
		return nil // No modules directory
	}

	return filepath.Walk(srcModulesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Look for index files in module directories
		if info.Name() == "index" && !info.IsDir() {
			relPath, err := filepath.Rel(srcModulesDir, path)
			if err != nil {
				return nil
			}
			dstPath := filepath.Join(dstModulesDir, relPath)
			// Copy index file, ignoring errors (non-fatal)
			_ = copyFile(path, dstPath)
		}
		return nil
	})
}

// getDeletedFiles returns a list of files that are deleted in the worktree
// (both staged deletions and unstaged deletions)
func getDeletedFiles(executor *git.Executor) []string {
	var deletedFiles []string

	statusOutput, err := executor.Execute("status", "--porcelain")
	if err != nil || statusOutput == "" {
		return deletedFiles
	}

	for _, line := range splitLines(statusOutput) {
		if len(line) < 3 {
			continue
		}

		// Status format: XY filename
		// X = index status, Y = worktree status
		status := line[:2]
		filename := line[3:]

		// Handle renamed files: "R  old -> new" - the old file is effectively deleted
		if status[0] == 'R' || status[1] == 'R' {
			if idx := findArrow(filename); idx != -1 {
				// Add old filename as deleted
				oldName := filename[:idx]
				deletedFiles = append(deletedFiles, oldName)
			}
			continue
		}

		// Check if file was deleted (D in either index or worktree status)
		if status[0] == 'D' || status[1] == 'D' {
			deletedFiles = append(deletedFiles, filename)
		}
	}

	return deletedFiles
}

// findArrow finds the index of " -> " in a string
func findArrow(s string) int {
	for i := 0; i < len(s)-3; i++ {
		if s[i:i+4] == " -> " {
			return i
		}
	}
	return -1
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// copyDirRecursive recursively copies a directory from src to dst
func copyDirRecursive(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		info, err := entry.Info()
		if err != nil {
			return err
		}

		if entry.IsDir() {
			if err := copyDirRecursive(srcPath, dstPath); err != nil {
				return err
			}
		} else if info.Mode()&os.ModeSymlink != 0 {
			// Handle symlinks
			link, err := os.Readlink(srcPath)
			if err != nil {
				return err
			}
			if err := os.Symlink(link, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// splitLines splits a string into lines, trimming whitespace
func splitLines(s string) []string {
	var result []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			if line != "" {
				result = append(result, line)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		line := s[start:]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}
