package repo

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/amaya382/baretree/internal/config"
	"github.com/amaya382/baretree/internal/git"
	"github.com/amaya382/baretree/internal/global"
	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/url"
	"github.com/spf13/cobra"
)

var (
	migrateDestination  string
	migrateInPlace      bool
	migrateToManaged    bool
	migrateRepoPath     string
	migrateRemoveSource bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate <existing-repo-path>",
	Short: "Convert existing Git repository to baretree [bt migrate]",
	Long: `Convert an existing Git repository (.git directory) to baretree structure.

This command:
  1. Validates the existing repository
  2. Creates a new baretree structure (bare repo + worktree)
  3. Preserves all working tree state (unstaged, staged, untracked files)
  4. Initializes baretree configuration in git-config

You must specify one of --in-place, --destination, or --to-managed:
  --in-place (-i): Replace the original repository in-place (recommended)
  --destination (-d): Create the baretree structure at a different location
  --to-managed (-m): Move to baretree managed directory with ghq-style path (host/user/repo)

The --to-managed option:
  - Automatically detects the destination path from git remote URL
  - Use --path to manually specify the path (e.g., github.com/user/repo)
  - Works with both regular Git repositories and existing baretree repositories
  - Existing baretree repositories are moved without re-conversion

With --destination and --to-managed, the original repository is preserved by default.
Use --remove-source to delete the original after successful migration.

Examples:
  # --to-managed: Move to baretree managed directory (e.g., ~/baretree/github.com/user/repo)
  bt repo migrate ~/projects/myapp -m
  bt repo migrate ~/projects/myapp -m --path github.com/user/myapp
  bt repo migrate ~/projects/myapp -m --remove-source

  # --in-place: Convert repository in current location
  bt repo migrate ~/projects/myapp -i
  bt repo migrate . --in-place

  # --destination: Copy to a specific directory
  bt repo migrate ~/projects/myapp -d ~/baretree/myapp
  bt repo migrate ~/projects/myapp -d ../my-project-baretree --remove-source`,
	Args: cobra.ExactArgs(1),
	RunE: runMigrate,
}

func init() {
	migrateCmd.Flags().BoolVarP(&migrateInPlace, "in-place", "i", false, "Replace the original repository in-place (recommended)")
	migrateCmd.Flags().StringVarP(&migrateDestination, "destination", "d", "", "Destination directory for the new baretree structure")
	migrateCmd.Flags().BoolVarP(&migrateToManaged, "to-managed", "m", false, "Move repository to baretree managed directory with ghq-style path")
	migrateCmd.Flags().StringVarP(&migrateRepoPath, "path", "p", "", "Repository path for --to-managed (default: auto-detect from remote URL)")
	migrateCmd.Flags().BoolVarP(&migrateRemoveSource, "remove-source", "r", false, "Remove the original repository after successful migration (only with -d or -m)")
}

func runMigrate(cmd *cobra.Command, args []string) error {
	sourcePath := args[0]

	// Count how many mode flags are set
	modeCount := 0
	if migrateInPlace {
		modeCount++
	}
	if migrateDestination != "" {
		modeCount++
	}
	if migrateToManaged {
		modeCount++
	}

	// Validate that exactly one mode is specified
	if modeCount == 0 {
		return fmt.Errorf("you must specify one of --in-place (-i), --destination (-d), or --to-managed (-m)")
	}
	if modeCount > 1 {
		return fmt.Errorf("cannot use multiple mode flags together")
	}

	// Validate --path is only used with --to-managed
	if migrateRepoPath != "" && !migrateToManaged {
		return fmt.Errorf("--path can only be used with --to-managed")
	}

	// Validate --remove-source is only used with --destination or --to-managed
	if migrateRemoveSource && migrateInPlace {
		return fmt.Errorf("--remove-source cannot be used with --in-place")
	}

	// Convert to absolute path
	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Handle --to-managed mode
	if migrateToManaged {
		return runMigrateToRoot(absSource)
	}

	// Check if source is a git repository
	gitDir := filepath.Join(absSource, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %s", absSource)
	}

	// Check if it's already a bare repository
	executor := git.NewExecutor(absSource)
	isBare, _ := executor.Execute("rev-parse", "--is-bare-repository")
	if isBare == "true" {
		return fmt.Errorf("repository is already bare")
	}

	// Check if it's already a baretree repository
	if repository.IsBaretreeRepo(absSource) {
		return fmt.Errorf("already a baretree repository: %s", absSource)
	}

	// Get current branch
	currentBranch, err := executor.Execute("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Determine destination
	var absDestination string
	if migrateInPlace {
		absDestination = absSource
	} else {
		absDestination, err = filepath.Abs(migrateDestination)
		if err != nil {
			return fmt.Errorf("failed to get absolute destination path: %w", err)
		}
		// Check if destination exists
		if _, err := os.Stat(absDestination); err == nil {
			return fmt.Errorf("destination already exists: %s", absDestination)
		}
	}

	// Get existing worktrees before migration
	srcExecutor := git.NewExecutor(absSource)
	output, err := srcExecutor.Execute("worktree", "list", "--porcelain")
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}
	existingWorktrees := git.ParseWorktreeList(output)

	// Filter external worktrees (worktrees outside the source directory)
	var externalWorktrees []git.Worktree
	for _, wt := range existingWorktrees {
		if wt.IsBare {
			continue
		}
		// Check if worktree is external (outside absSource)
		relPath, err := filepath.Rel(absSource, wt.Path)
		if err != nil || strings.HasPrefix(relPath, "..") {
			externalWorktrees = append(externalWorktrees, wt)
		}
	}

	fmt.Printf("Migrating repository: %s\n", absSource)
	fmt.Printf("Current branch: %s\n", currentBranch)
	if migrateInPlace {
		fmt.Printf("Mode: in-place\n")
	} else {
		fmt.Printf("Destination: %s\n", absDestination)
	}
	if len(externalWorktrees) > 0 {
		fmt.Printf("External worktrees to migrate: %d\n", len(externalWorktrees))
		for _, wt := range externalWorktrees {
			fmt.Printf("  - %s (%s)\n", wt.Branch, wt.Path)
		}
	}

	return performMigration(absSource, absDestination, currentBranch, migrateInPlace, externalWorktrees)
}

func performMigration(absSource, absDestination, currentBranch string, inPlace bool, externalWorktrees []git.Worktree) error {
	if inPlace {
		return migrateInPlaceImpl(absSource, currentBranch, externalWorktrees)
	}
	return migrateToDestination(absSource, absDestination, currentBranch, externalWorktrees)
}

func migrateInPlaceImpl(absSource, currentBranch string, externalWorktrees []git.Worktree) error {
	// For in-place migration:
	// 1. Convert .git directory to a bare repo
	// 2. Initialize baretree config in git-config
	// 3. Move all files except .git to <branch>/ directory
	// 4. Set up worktree
	// 5. Move external worktrees into the baretree structure

	barePath := filepath.Join(absSource, config.BareDir)
	worktreePath := filepath.Join(absSource, currentBranch)

	// Step 1: Convert .git to bare repository
	fmt.Printf("Converting to bare repository at %s...\n", barePath)

	// Step 2: Convert to bare repository
	bareExecutor := git.NewExecutor(barePath)
	if _, err := bareExecutor.Execute("config", "--bool", "core.bare", "true"); err != nil {
		return fmt.Errorf("failed to set bare config: %w", err)
	}

	// Step 3: Read entries BEFORE creating worktree directory
	// This is important for nested branches like "feat/xxx" where MkdirAll creates "feat" directory
	entries, err := os.ReadDir(absSource)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	// Step 4: Create worktree directory
	if err := os.MkdirAll(worktreePath, 0755); err != nil {
		// Rollback: revert to non-bare
		if _, executeErr := bareExecutor.Execute("config", "--bool", "core.bare", "false"); executeErr != nil {
			return fmt.Errorf("failed to create worktree directory and also failed to roll back: %w /%w", err, executeErr)
		}
		return fmt.Errorf("failed to create worktree directory: %w", err)
	}

	// Step 5: Move all files (except bare dir) to worktree

	for _, entry := range entries {
		name := entry.Name()
		if name == config.BareDir {
			continue
		}
		srcPath := filepath.Join(absSource, name)
		dstPath := filepath.Join(worktreePath, name)

		// Check if srcPath is an ancestor of worktreePath (for nested branches like "feat/xxx")
		// In this case, srcPath would be "feat" and worktreePath would be "feat/xxx"
		relPath, relErr := filepath.Rel(srcPath, worktreePath)
		if relErr == nil && !strings.HasPrefix(relPath, "..") && relPath != "." {
			// srcPath is an ancestor of worktreePath
			// Move contents except the worktree path component
			if err := moveContentsExcludingWorktree(srcPath, dstPath, worktreePath); err != nil {
				return fmt.Errorf("failed to move contents of %s to worktree: %w", name, err)
			}
			// Remove the now-empty source directory (should only contain the worktree path now)
			// The worktree directory itself will remain
			continue
		}

		// Check if destination already exists (created by MkdirAll for nested branch like "feat/xxx")
		if dstInfo, err := os.Stat(dstPath); err == nil && dstInfo.IsDir() && entry.IsDir() {
			// Destination directory exists - move contents recursively
			if err := moveContentsRecursively(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to move contents of %s to worktree: %w", name, err)
			}
			// Remove the now-empty source directory
			if err := os.Remove(srcPath); err != nil {
				return fmt.Errorf("failed to remove empty source directory %s: %w", name, err)
			}
		} else {
			if err := os.Rename(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to move %s to worktree: %w", name, err)
			}
		}
	}

	// Step 6: Set up worktree link
	// Create .git file in worktree pointing to bare repo's worktrees directory
	// Use escaped name for the worktree git directory to avoid nested paths
	worktreeGitDirName := git.ToWorktreeGitDirName(currentBranch)
	worktreeGitDir := filepath.Join(barePath, "worktrees", worktreeGitDirName)
	if err := os.MkdirAll(worktreeGitDir, 0755); err != nil {
		return fmt.Errorf("failed to create worktree git dir: %w", err)
	}

	// Create gitdir file in worktree
	gitFileContent := fmt.Sprintf("gitdir: %s\n", worktreeGitDir)
	if err := os.WriteFile(filepath.Join(worktreePath, ".git"), []byte(gitFileContent), 0644); err != nil {
		return fmt.Errorf("failed to create .git file: %w", err)
	}

	// Create commondir file
	relBare, _ := filepath.Rel(worktreeGitDir, barePath)
	if err := os.WriteFile(filepath.Join(worktreeGitDir, "commondir"), []byte(relBare+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to create commondir file: %w", err)
	}

	// Create gitdir file in bare repo's worktrees
	absWorktreePath, _ := filepath.Abs(worktreePath)
	if err := os.WriteFile(filepath.Join(worktreeGitDir, "gitdir"), []byte(absWorktreePath+"/.git\n"), 0644); err != nil {
		return fmt.Errorf("failed to create gitdir file: %w", err)
	}

	// Create HEAD file
	if err := os.WriteFile(filepath.Join(worktreeGitDir, "HEAD"), []byte("ref: refs/heads/"+currentBranch+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to create HEAD file: %w", err)
	}

	// Copy index file to preserve staging state
	srcIndex := filepath.Join(barePath, "index")
	dstIndex := filepath.Join(worktreeGitDir, "index")
	if _, err := os.Stat(srcIndex); err == nil {
		if err := copyFile(srcIndex, dstIndex); err != nil {
			return fmt.Errorf("failed to copy index file: %w", err)
		}
		// Remove index from bare repo (it shouldn't have one)
		os.Remove(srcIndex)
	}

	// Step 6: Update submodule .git files for new directory structure
	fmt.Printf("Updating submodule paths...\n")
	if err := updateSubmoduleGitFiles(worktreePath, barePath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update some submodule paths: %v\n", err)
	}

	// Step 7: Detect default branch and initialize baretree config
	defaultBranch, err := git.GetDefaultBranch(barePath)
	if err != nil {
		// Fallback to current branch if detection fails
		defaultBranch = currentBranch
	}

	if defaultBranch != currentBranch {
		fmt.Printf("Default branch: %s (detected from remote)\n", defaultBranch)
	}

	if err := repository.InitializeBareRepo(absSource, defaultBranch); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	// Step 8: Create default branch worktree if different from current branch
	var defaultBranchWorktreePath string
	if defaultBranch != currentBranch {
		defaultBranchWorktreePath = filepath.Join(absSource, defaultBranch)
		if _, err := bareExecutor.Execute("worktree", "add", defaultBranchWorktreePath, defaultBranch); err != nil {
			fmt.Printf("Warning: failed to create default branch worktree: %v\n", err)
			defaultBranchWorktreePath = "" // Clear on failure
		}
	}

	// Step 9: Move external worktrees into the baretree structure
	movedWorktrees, err := migrateExternalWorktrees(absSource, barePath, bareExecutor, externalWorktrees)
	if err != nil {
		return fmt.Errorf("failed to migrate external worktrees: %w", err)
	}

	fmt.Printf("\n✓ Migration successful!\n")
	fmt.Printf("  Repository root: %s\n", absSource)
	fmt.Printf("  Bare repo: %s\n", barePath)
	fmt.Printf("  Worktree: %s\n", worktreePath)
	if defaultBranchWorktreePath != "" {
		fmt.Printf("  Worktree: %s (default branch)\n", defaultBranchWorktreePath)
	}
	for _, wt := range movedWorktrees {
		fmt.Printf("  Worktree: %s\n", wt)
	}
	fmt.Printf("\nAll working tree state (unstaged, staged, untracked) has been preserved.\n")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  cd %s\n", worktreePath)
	fmt.Printf("  bt status\n")

	return nil
}

func migrateToDestination(absSource, absDestination, currentBranch string, externalWorktrees []git.Worktree) error {
	// For destination migration:
	// 1. Create destination directory
	// 2. Copy .git as bare repository
	// 3. Create worktree directory and copy all working files
	// 4. Set up worktree links and preserve index

	barePath := filepath.Join(absDestination, config.BareDir)
	worktreePath := filepath.Join(absDestination, currentBranch)
	srcGitDir := filepath.Join(absSource, ".git")

	// Step 1: Create destination directory
	fmt.Printf("Creating destination at %s...\n", absDestination)
	if err := os.MkdirAll(absDestination, 0755); err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}

	// Step 2: Copy .git as bare repository
	fmt.Printf("Creating bare repository at %s...\n", barePath)
	if err := copyDir(srcGitDir, barePath); err != nil {
		os.RemoveAll(absDestination)
		return fmt.Errorf("failed to copy .git: %w", err)
	}

	// Convert to bare repository
	bareExecutor := git.NewExecutor(barePath)
	if _, err := bareExecutor.Execute("config", "--bool", "core.bare", "true"); err != nil {
		os.RemoveAll(absDestination)
		return fmt.Errorf("failed to set bare config: %w", err)
	}

	// Step 3: Create worktree directory
	if err := os.MkdirAll(worktreePath, 0755); err != nil {
		os.RemoveAll(absDestination)
		return fmt.Errorf("failed to create worktree directory: %w", err)
	}

	// Step 4: Copy all files from source to worktree (preserving working tree state)
	fmt.Printf("Copying working tree to %s...\n", worktreePath)
	entries, err := os.ReadDir(absSource)
	if err != nil {
		os.RemoveAll(absDestination)
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if name == ".git" {
			continue
		}
		srcPath := filepath.Join(absSource, name)
		dstPath := filepath.Join(worktreePath, name)

		// Use Lstat to detect symlinks
		info, err := os.Lstat(srcPath)
		if err != nil {
			os.RemoveAll(absDestination)
			return fmt.Errorf("failed to stat %s: %w", name, err)
		}

		// Handle symlinks first (before checking IsDir)
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(srcPath)
			if err != nil {
				os.RemoveAll(absDestination)
				return fmt.Errorf("failed to read symlink %s: %w", name, err)
			}
			if err := os.Symlink(link, dstPath); err != nil {
				os.RemoveAll(absDestination)
				return fmt.Errorf("failed to create symlink %s: %w", name, err)
			}
		} else if info.IsDir() {
			// copyDir handles existing directories via MkdirAll, so it's safe for nested branches
			if err := copyDir(srcPath, dstPath); err != nil {
				os.RemoveAll(absDestination)
				return fmt.Errorf("failed to copy directory %s: %w", name, err)
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				os.RemoveAll(absDestination)
				return fmt.Errorf("failed to copy file %s: %w", name, err)
			}
		}
	}

	// Step 5: Set up worktree link
	// Use escaped name for the worktree git directory to avoid nested paths
	worktreeGitDirName := git.ToWorktreeGitDirName(currentBranch)
	worktreeGitDir := filepath.Join(barePath, "worktrees", worktreeGitDirName)
	if err := os.MkdirAll(worktreeGitDir, 0755); err != nil {
		os.RemoveAll(absDestination)
		return fmt.Errorf("failed to create worktree git dir: %w", err)
	}

	// Create .git file in worktree
	gitFileContent := fmt.Sprintf("gitdir: %s\n", worktreeGitDir)
	if err := os.WriteFile(filepath.Join(worktreePath, ".git"), []byte(gitFileContent), 0644); err != nil {
		os.RemoveAll(absDestination)
		return fmt.Errorf("failed to create .git file: %w", err)
	}

	// Create commondir file
	relBare, _ := filepath.Rel(worktreeGitDir, barePath)
	if err := os.WriteFile(filepath.Join(worktreeGitDir, "commondir"), []byte(relBare+"\n"), 0644); err != nil {
		os.RemoveAll(absDestination)
		return fmt.Errorf("failed to create commondir file: %w", err)
	}

	// Create gitdir file
	absWorktreePath, _ := filepath.Abs(worktreePath)
	if err := os.WriteFile(filepath.Join(worktreeGitDir, "gitdir"), []byte(absWorktreePath+"/.git\n"), 0644); err != nil {
		os.RemoveAll(absDestination)
		return fmt.Errorf("failed to create gitdir file: %w", err)
	}

	// Create HEAD file
	if err := os.WriteFile(filepath.Join(worktreeGitDir, "HEAD"), []byte("ref: refs/heads/"+currentBranch+"\n"), 0644); err != nil {
		os.RemoveAll(absDestination)
		return fmt.Errorf("failed to create HEAD file: %w", err)
	}

	// Copy index file to worktree git dir (preserve staging state)
	srcIndex := filepath.Join(barePath, "index")
	dstIndex := filepath.Join(worktreeGitDir, "index")
	if _, err := os.Stat(srcIndex); err == nil {
		if err := copyFile(srcIndex, dstIndex); err != nil {
			os.RemoveAll(absDestination)
			return fmt.Errorf("failed to copy index file: %w", err)
		}
		// Remove index from bare repo
		os.Remove(srcIndex)
	}

	// Step 6: Update submodule .git files for new directory structure
	fmt.Printf("Updating submodule paths...\n")
	if err := updateSubmoduleGitFiles(worktreePath, barePath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update some submodule paths: %v\n", err)
	}

	// Step 7: Detect default branch and initialize baretree config
	defaultBranch, err := git.GetDefaultBranch(barePath)
	if err != nil {
		// Fallback to current branch if detection fails
		defaultBranch = currentBranch
	}

	if defaultBranch != currentBranch {
		fmt.Printf("Default branch: %s (detected from remote)\n", defaultBranch)
	}

	if err := repository.InitializeBareRepo(absDestination, defaultBranch); err != nil {
		os.RemoveAll(absDestination)
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	// Step 8: Create default branch worktree if different from current branch
	var defaultBranchWorktreePath string
	if defaultBranch != currentBranch {
		defaultBranchWorktreePath = filepath.Join(absDestination, defaultBranch)
		if _, err := bareExecutor.Execute("worktree", "add", defaultBranchWorktreePath, defaultBranch); err != nil {
			fmt.Printf("Warning: failed to create default branch worktree: %v\n", err)
			defaultBranchWorktreePath = "" // Clear on failure
		}
	}

	// Step 9: Copy and migrate external worktrees
	movedWorktrees, err := migrateExternalWorktreesWithCopy(absDestination, barePath, bareExecutor, externalWorktrees)
	if err != nil {
		os.RemoveAll(absDestination)
		return fmt.Errorf("failed to migrate external worktrees: %w", err)
	}

	fmt.Printf("\n✓ Migration successful!\n")
	fmt.Printf("  New repository: %s\n", absDestination)
	fmt.Printf("  Bare repo: %s\n", barePath)
	fmt.Printf("  Worktree: %s\n", worktreePath)
	if defaultBranchWorktreePath != "" {
		fmt.Printf("  Worktree: %s (default branch)\n", defaultBranchWorktreePath)
	}
	for _, wt := range movedWorktrees {
		fmt.Printf("  Worktree: %s\n", wt)
	}
	fmt.Printf("\nAll working tree state (unstaged, staged, untracked) has been preserved.\n")

	// Remove original if requested
	if migrateRemoveSource {
		fmt.Printf("Removing original repository...\n")
		if err := os.RemoveAll(absSource); err != nil {
			fmt.Printf("Warning: failed to remove original directory: %v\n", err)
		} else {
			fmt.Printf("  Original removed: %s\n", absSource)
		}
	} else {
		fmt.Printf("\nOriginal repository preserved at: %s\n", absSource)
		fmt.Printf("\nIf migration is successful, you can remove the original:\n")
		fmt.Printf("  rm -rf %s\n", absSource)
	}

	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  cd %s\n", worktreePath)
	fmt.Printf("  bt status\n")

	return nil
}

// copyFile copies a single file from src to dst
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

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// copyDir recursively copies a directory from src to dst
func copyDir(src, dst string) error {
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

		// Get file info using Lstat to detect symlinks
		info, err := os.Lstat(srcPath)
		if err != nil {
			return err
		}

		// Handle symlinks first (before checking IsDir)
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(srcPath)
			if err != nil {
				return err
			}
			if err := os.Symlink(link, dstPath); err != nil {
				return err
			}
		} else if info.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
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

// moveContentsRecursively moves all contents from src directory to dst directory recursively.
// It handles the case where dst already has some directories (e.g., from MkdirAll with nested paths).
func moveContentsRecursively(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		// Check if destination already exists
		if dstInfo, err := os.Stat(dstPath); err == nil && dstInfo.IsDir() && entry.IsDir() {
			// Both are directories - recurse
			if err := moveContentsRecursively(srcPath, dstPath); err != nil {
				return err
			}
			// Remove the now-empty source directory
			if err := os.Remove(srcPath); err != nil {
				return err
			}
		} else {
			// Move directly
			if err := os.Rename(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// moveContentsExcludingWorktree moves contents from src to dst, excluding directories that are
// ancestors of the worktreePath. This handles cases like moving "feat" to "feat/xxx/feat"
// where "feat" contains "xxx" (the worktree directory).
func moveContentsExcludingWorktree(src, dst, worktreePath string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		// Check if srcPath is worktreePath or an ancestor of worktreePath
		relPath, relErr := filepath.Rel(srcPath, worktreePath)
		if relErr == nil && !strings.HasPrefix(relPath, "..") {
			// srcPath is worktreePath itself or an ancestor of worktreePath
			if relPath == "." {
				// srcPath IS worktreePath - skip entirely
				continue
			}
			// srcPath is an ancestor of worktreePath - need to recurse
			if entry.IsDir() {
				if err := moveContentsExcludingWorktree(srcPath, dstPath, worktreePath); err != nil {
					return err
				}
				continue
			}
		}

		// Normal move
		if err := os.Rename(srcPath, dstPath); err != nil {
			return err
		}
	}

	return nil
}

// runMigrateToRoot handles migration to baretree root directory
func runMigrateToRoot(absSource string) error {
	// Load global config to get baretree root
	cfg, err := global.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if source is already a baretree repository
	isBaretree := repository.IsBaretreeRepo(absSource)

	// Determine the git directory for remote URL lookup
	var gitExecutorPath string
	if isBaretree {
		// For baretree repos, find the bare directory
		bareDir, err := findBareDir(absSource)
		if err != nil {
			return fmt.Errorf("failed to find bare directory: %w", err)
		}
		gitExecutorPath = bareDir
	} else {
		// For regular repos, check if it's a valid git repo
		gitDir := filepath.Join(absSource, ".git")
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			return fmt.Errorf("not a git repository: %s", absSource)
		}
		gitExecutorPath = absSource
	}

	// Determine repository path (host/user/repo)
	var repoPath *url.RepoPath
	if migrateRepoPath != "" {
		// Use explicitly provided path
		repoPath, err = url.Parse(migrateRepoPath, "github.com", cfg.User)
		if err != nil {
			return fmt.Errorf("failed to parse --path: %w", err)
		}
	} else {
		// Detect from remote URL
		repoPath, err = detectRepoPathFromRemote(gitExecutorPath)
		if err != nil {
			return fmt.Errorf("failed to detect repository path from remote: %w\nUse --path to specify manually (e.g., --path github.com/user/repo)", err)
		}
	}

	// Build destination path
	absDestination := filepath.Join(cfg.PrimaryRoot(), repoPath.String())

	// Check if destination already exists
	if _, err := os.Stat(absDestination); err == nil {
		return fmt.Errorf("destination already exists: %s", absDestination)
	}

	// Check if source is inside destination or vice versa
	if isSubPath(absSource, absDestination) || isSubPath(absDestination, absSource) {
		return fmt.Errorf("source and destination paths overlap")
	}

	fmt.Printf("Migrating repository to baretree root:\n")
	fmt.Printf("  Source: %s\n", absSource)
	fmt.Printf("  Destination: %s\n", absDestination)
	fmt.Printf("  Repository: %s\n", repoPath.String())

	if isBaretree {
		// Already a baretree repository - just move it
		return moveBaretreeRepo(absSource, absDestination, migrateRemoveSource)
	}

	// Regular git repository - migrate and move
	return migrateToManagedImpl(absSource, absDestination, migrateRemoveSource)
}

// findBareDir finds the bare repository directory in a baretree repo
func findBareDir(repoRoot string) (string, error) {
	barePath := filepath.Join(repoRoot, config.BareDir)
	if info, err := os.Stat(barePath); err == nil && info.IsDir() {
		// Verify it's a git directory
		if _, err := os.Stat(filepath.Join(barePath, "HEAD")); err == nil {
			return barePath, nil
		}
	}

	return "", fmt.Errorf("bare repository directory not found")
}

// detectRepoPathFromRemote detects the repository path from git remote URL
func detectRepoPathFromRemote(gitPath string) (*url.RepoPath, error) {
	executor := git.NewExecutor(gitPath)

	// Try origin remote first
	remoteURL, err := executor.Execute("config", "--get", "remote.origin.url")
	if err != nil || remoteURL == "" {
		// Try to get any remote
		remotes, err := executor.Execute("remote")
		if err != nil || remotes == "" {
			return nil, fmt.Errorf("no git remotes configured")
		}
		// Use first remote
		firstRemote := strings.Split(strings.TrimSpace(remotes), "\n")[0]
		remoteURL, err = executor.Execute("config", "--get", fmt.Sprintf("remote.%s.url", firstRemote))
		if err != nil || remoteURL == "" {
			return nil, fmt.Errorf("failed to get remote URL")
		}
	}

	return url.ParseRemoteURL(strings.TrimSpace(remoteURL))
}

// isSubPath checks if child is a subpath of parent
func isSubPath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return !filepath.IsAbs(rel) && !startsWith(rel, "..")
}

// startsWith checks if s starts with prefix
func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// moveBaretreeRepo moves an existing baretree repository to a new location
func moveBaretreeRepo(absSource, absDestination string, removeSource bool) error {
	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(absDestination), 0755); err != nil {
		return fmt.Errorf("failed to create destination parent directory: %w", err)
	}

	fmt.Printf("Copying baretree repository...\n")

	// Copy the repository to destination
	if err := copyDir(absSource, absDestination); err != nil {
		return fmt.Errorf("failed to copy repository: %w", err)
	}

	// Update worktree paths after copy
	if err := updateWorktreePaths(absSource, absDestination); err != nil {
		os.RemoveAll(absDestination)
		return fmt.Errorf("failed to update worktree paths: %w", err)
	}

	fmt.Printf("\n✓ Repository copied successfully!\n")
	fmt.Printf("  New location: %s\n", absDestination)

	// Remove original if requested
	if removeSource {
		fmt.Printf("Removing original repository...\n")
		if err := os.RemoveAll(absSource); err != nil {
			fmt.Printf("Warning: failed to remove original directory: %v\n", err)
		} else {
			fmt.Printf("  Original removed: %s\n", absSource)
		}
	} else {
		fmt.Printf("\nOriginal repository preserved at: %s\n", absSource)
		fmt.Printf("To remove the original, run:\n")
		fmt.Printf("  rm -rf %s\n", absSource)
	}

	return nil
}

// updateWorktreePaths updates all worktree gitdir paths after moving a baretree repo
func updateWorktreePaths(oldRepoRoot, newRepoRoot string) error {
	newBareDir, err := findBareDir(newRepoRoot)
	if err != nil {
		return err
	}

	// Calculate the relative path of bare dir from repo root
	relBareDir, err := filepath.Rel(newRepoRoot, newBareDir)
	if err != nil {
		return fmt.Errorf("failed to calculate relative bare dir path: %w", err)
	}

	// Calculate old bare dir path
	oldBareDir := filepath.Join(oldRepoRoot, relBareDir)

	worktreesDir := filepath.Join(newBareDir, "worktrees")
	if _, err := os.Stat(worktreesDir); os.IsNotExist(err) {
		return nil // No worktrees to update
	}

	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		worktreeName := entry.Name()
		newWorktreeGitDir := filepath.Join(worktreesDir, worktreeName)
		oldWorktreeGitDir := filepath.Join(oldBareDir, "worktrees", worktreeName)

		// Read current gitdir file to get the old worktree path
		gitdirFile := filepath.Join(newWorktreeGitDir, "gitdir")
		oldGitdirContent, err := os.ReadFile(gitdirFile)
		if err != nil {
			return fmt.Errorf("failed to read gitdir for %s: %w", worktreeName, err)
		}

		// Parse the old worktree path from gitdir (remove trailing newline and "/.git" suffix)
		oldWorktreeGitPath := strings.TrimSpace(string(oldGitdirContent))
		oldWorktreePath := strings.TrimSuffix(oldWorktreeGitPath, "/.git")

		// Handle relative paths by converting to absolute path based on OLD worktreeGitDir
		if !filepath.IsAbs(oldWorktreePath) {
			oldWorktreePath = filepath.Join(oldWorktreeGitDir, oldWorktreePath)
			oldWorktreePath = filepath.Clean(oldWorktreePath)
		}

		// Calculate relative path from old repo root to the worktree
		relWorktreePath, err := filepath.Rel(oldRepoRoot, oldWorktreePath)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path for %s: %w", worktreeName, err)
		}

		// Calculate new worktree path
		newWorktreePath := filepath.Join(newRepoRoot, relWorktreePath)

		// Update gitdir file in bare repo's worktrees directory
		newGitdir := filepath.Join(newWorktreePath, ".git") + "\n"
		if err := os.WriteFile(gitdirFile, []byte(newGitdir), 0644); err != nil {
			return fmt.Errorf("failed to update gitdir for %s: %w", worktreeName, err)
		}

		// Update .git file in worktree directory
		worktreeGitFile := filepath.Join(newWorktreePath, ".git")
		if _, err := os.Stat(worktreeGitFile); err == nil {
			newContent := fmt.Sprintf("gitdir: %s\n", newWorktreeGitDir)
			if err := os.WriteFile(worktreeGitFile, []byte(newContent), 0644); err != nil {
				return fmt.Errorf("failed to update .git file for %s: %w", worktreeName, err)
			}
		}
	}

	return nil
}

// migrateToManagedImpl migrates a regular git repository to baretree root
func migrateToManagedImpl(absSource, absDestination string, removeSource bool) error {
	// Check if it's already a bare repository
	executor := git.NewExecutor(absSource)
	isBare, _ := executor.Execute("rev-parse", "--is-bare-repository")
	if isBare == "true" {
		return fmt.Errorf("repository is already bare")
	}

	// Get current branch
	currentBranch, err := executor.Execute("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Get existing worktrees before migration
	output, err := executor.Execute("worktree", "list", "--porcelain")
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}
	existingWorktrees := git.ParseWorktreeList(output)

	// Filter external worktrees (worktrees outside the source directory)
	var externalWorktrees []git.Worktree
	for _, wt := range existingWorktrees {
		if wt.IsBare {
			continue
		}
		// Check if worktree is external (outside absSource)
		relPath, err := filepath.Rel(absSource, wt.Path)
		if err != nil || strings.HasPrefix(relPath, "..") {
			externalWorktrees = append(externalWorktrees, wt)
		}
	}

	fmt.Printf("Current branch: %s\n", currentBranch)
	if len(externalWorktrees) > 0 {
		fmt.Printf("External worktrees to migrate: %d\n", len(externalWorktrees))
		for _, wt := range externalWorktrees {
			fmt.Printf("  - %s (%s)\n", wt.Branch, wt.Path)
		}
	}

	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(absDestination), 0755); err != nil {
		return fmt.Errorf("failed to create destination parent directory: %w", err)
	}

	// Copy source to destination
	fmt.Printf("Copying repository to destination...\n")
	if err := copyDir(absSource, absDestination); err != nil {
		return fmt.Errorf("failed to copy repository: %w", err)
	}

	// Now perform in-place migration at destination with external worktrees
	if err := migrateInPlaceImpl(absDestination, currentBranch, externalWorktrees); err != nil {
		// Clean up destination on failure
		os.RemoveAll(absDestination)
		return fmt.Errorf("failed to migrate repository in place: %w", err)
	}

	// Remove original if requested
	if removeSource {
		fmt.Printf("Removing original repository...\n")
		if err := os.RemoveAll(absSource); err != nil {
			fmt.Printf("Warning: failed to remove original directory: %v\n", err)
		} else {
			fmt.Printf("  Original removed: %s\n", absSource)
		}
	} else {
		fmt.Printf("\nOriginal repository preserved at: %s\n", absSource)
		fmt.Printf("To remove the original, run:\n")
		fmt.Printf("  rm -rf %s\n", absSource)
	}

	return nil
}

// migrateExternalWorktrees moves external worktrees into the baretree structure (for in-place migration)
func migrateExternalWorktrees(repoRoot, barePath string, executor *git.Executor, worktrees []git.Worktree) ([]string, error) {
	if len(worktrees) == 0 {
		return nil, nil
	}

	var movedWorktrees []string

	for _, wt := range worktrees {
		fmt.Printf("Moving external worktree: %s (%s)...\n", wt.Branch, wt.Path)

		targetPath := filepath.Join(repoRoot, wt.Branch)

		// Check if target already exists
		if _, err := os.Stat(targetPath); err == nil {
			return movedWorktrees, fmt.Errorf("target path already exists: %s", targetPath)
		}

		// Create parent directories if needed (for hierarchical branch names like feature/foo)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return movedWorktrees, fmt.Errorf("failed to create parent directory for %s: %w", wt.Branch, err)
		}

		// Move worktree directory
		if err := os.Rename(wt.Path, targetPath); err != nil {
			return movedWorktrees, fmt.Errorf("failed to move worktree %s: %w", wt.Branch, err)
		}

		// Repair Git worktree metadata using git worktree repair
		if _, err := executor.Execute("worktree", "repair", targetPath); err != nil {
			// Try to rollback
			if rollbackErr := os.Rename(targetPath, wt.Path); rollbackErr != nil {
				return movedWorktrees, fmt.Errorf("failed to repair worktree and also failed to roll back: %w / %w", err, rollbackErr)
			}
			return movedWorktrees, fmt.Errorf("failed to repair worktree %s: %w", wt.Branch, err)
		}

		// Update submodule .git files in the moved worktree
		if err := updateSubmoduleGitFiles(targetPath, barePath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update some submodule paths for %s: %v\n", wt.Branch, err)
		}

		movedWorktrees = append(movedWorktrees, targetPath)
	}

	return movedWorktrees, nil
}

// migrateExternalWorktreesWithCopy copies external worktrees into the baretree structure (for destination migration)
func migrateExternalWorktreesWithCopy(repoRoot, barePath string, executor *git.Executor, worktrees []git.Worktree) ([]string, error) {
	if len(worktrees) == 0 {
		return nil, nil
	}

	var movedWorktrees []string

	for _, wt := range worktrees {
		fmt.Printf("Copying external worktree: %s (%s)...\n", wt.Branch, wt.Path)

		targetPath := filepath.Join(repoRoot, wt.Branch)

		// Check if target already exists
		if _, err := os.Stat(targetPath); err == nil {
			return movedWorktrees, fmt.Errorf("target path already exists: %s", targetPath)
		}

		// Create parent directories if needed (for hierarchical branch names like feature/foo)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return movedWorktrees, fmt.Errorf("failed to create parent directory for %s: %w", wt.Branch, err)
		}

		// Copy worktree directory (excluding .git file which will be recreated)
		if err := copyWorktreeDir(wt.Path, targetPath); err != nil {
			return movedWorktrees, fmt.Errorf("failed to copy worktree %s: %w", wt.Branch, err)
		}

		// Find the original worktree git directory name from the source .git file
		srcGitFile := filepath.Join(wt.Path, ".git")
		srcGitContent, err := os.ReadFile(srcGitFile)
		if err != nil {
			return movedWorktrees, fmt.Errorf("failed to read .git file for %s: %w", wt.Branch, err)
		}
		srcGitDir := strings.TrimSpace(strings.TrimPrefix(string(srcGitContent), "gitdir:"))
		srcWorktreeName := filepath.Base(srcGitDir) // e.g., "auth" or "custom-wt-name"

		// The source git directory was already copied to barePath/worktrees/{srcWorktreeName}
		// We need to use this existing directory and update its paths
		srcWorktreeGitDir := filepath.Join(barePath, "worktrees", srcWorktreeName)

		// New worktree git directory (using escaped branch name to avoid nested paths)
		newWorktreeGitDirName := git.ToWorktreeGitDirName(wt.Branch)
		newWorktreeGitDir := filepath.Join(barePath, "worktrees", newWorktreeGitDirName)

		// If source and destination git dirs are different, we need to rename/restructure
		if srcWorktreeGitDir != newWorktreeGitDir {

			// Move the existing git directory to the new location
			if _, err := os.Stat(srcWorktreeGitDir); err == nil {
				if err := os.Rename(srcWorktreeGitDir, newWorktreeGitDir); err != nil {
					// If rename fails, copy instead
					if err := copyDir(srcWorktreeGitDir, newWorktreeGitDir); err != nil {
						return movedWorktrees, fmt.Errorf("failed to move worktree git dir for %s: %w", wt.Branch, err)
					}
					os.RemoveAll(srcWorktreeGitDir)
				}
			} else {
				// Source git dir doesn't exist, create new one
				if err := os.MkdirAll(newWorktreeGitDir, 0755); err != nil {
					return movedWorktrees, fmt.Errorf("failed to create worktree git dir for %s: %w", wt.Branch, err)
				}
			}
		}

		// Create/update .git file in worktree
		gitFileContent := fmt.Sprintf("gitdir: %s\n", newWorktreeGitDir)
		if err := os.WriteFile(filepath.Join(targetPath, ".git"), []byte(gitFileContent), 0644); err != nil {
			return movedWorktrees, fmt.Errorf("failed to create .git file for %s: %w", wt.Branch, err)
		}

		// Update commondir file
		relBare, _ := filepath.Rel(newWorktreeGitDir, barePath)
		if err := os.WriteFile(filepath.Join(newWorktreeGitDir, "commondir"), []byte(relBare+"\n"), 0644); err != nil {
			return movedWorktrees, fmt.Errorf("failed to create commondir for %s: %w", wt.Branch, err)
		}

		// Update gitdir file
		absTargetPath, _ := filepath.Abs(targetPath)
		if err := os.WriteFile(filepath.Join(newWorktreeGitDir, "gitdir"), []byte(absTargetPath+"/.git\n"), 0644); err != nil {
			return movedWorktrees, fmt.Errorf("failed to create gitdir for %s: %w", wt.Branch, err)
		}

		// Update HEAD file (preserve detached state if applicable)
		if wt.Branch == "detached" {
			// For detached HEAD, keep the existing HEAD file (it contains the commit hash)
			// The HEAD was already copied from source
		} else {
			if err := os.WriteFile(filepath.Join(newWorktreeGitDir, "HEAD"), []byte("ref: refs/heads/"+wt.Branch+"\n"), 0644); err != nil {
				return movedWorktrees, fmt.Errorf("failed to create HEAD for %s: %w", wt.Branch, err)
			}
		}

		// Update submodule .git files in the copied worktree
		if err := updateSubmoduleGitFiles(targetPath, barePath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update some submodule paths for %s: %v\n", wt.Branch, err)
		}

		movedWorktrees = append(movedWorktrees, targetPath)
	}

	return movedWorktrees, nil
}

// updateSubmoduleGitFiles updates all submodule .git files to use correct relative paths
// This is needed because when files are moved from repo root to a worktree subdirectory,
// the relative paths to .git/modules/ need to be updated
func updateSubmoduleGitFiles(worktreePath, barePath string) error {
	// Check if .gitmodules exists
	gitmodulesPath := filepath.Join(worktreePath, ".gitmodules")
	if _, err := os.Stat(gitmodulesPath); os.IsNotExist(err) {
		return nil // No submodules
	}

	// Walk through worktree to find submodule .git files
	return filepath.Walk(worktreePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip the worktree's own .git file
		if path == filepath.Join(worktreePath, ".git") {
			return nil
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

			// Check if it references .git/modules (standard git structure)
			if strings.Contains(gitdirPath, "/.git/modules/") || strings.Contains(gitdirPath, string(filepath.Separator)+".git"+string(filepath.Separator)+"modules"+string(filepath.Separator)) {
				// Get the submodule path relative to worktree
				submodulePath := filepath.Dir(path)
				relToWorktree, err := filepath.Rel(worktreePath, submodulePath)
				if err != nil {
					return nil
				}

				// Calculate the depth (number of directories) from submodule to worktree
				depth := 1 // Start with 1 for the worktree directory itself
				for _, c := range relToWorktree {
					if c == filepath.Separator {
						depth++
					}
				}
				if relToWorktree != "." {
					depth++ // Add 1 for the submodule directory
				}

				// Extract module path from the gitdir path
				modulePath := extractModulePath(gitdirPath)
				if modulePath == "" {
					return nil
				}

				// Build new gitdir path with correct depth
				var relPrefix string
				for i := 0; i < depth; i++ {
					relPrefix = filepath.Join(relPrefix, "..")
				}
				newGitdir := filepath.Join(relPrefix, ".git", "modules", modulePath)
				newGitdir = filepath.Clean(newGitdir)

				// Write updated content
				newContent := fmt.Sprintf("gitdir: %s\n", newGitdir)
				if err := os.WriteFile(path, []byte(newContent), info.Mode()); err != nil {
					return nil // Skip if can't write
				}

				// Update the module's config file with correct worktree path
				moduleGitDir := filepath.Join(barePath, "modules", modulePath)
				if err := updateModuleWorktreePath(moduleGitDir, submodulePath); err != nil {
					// Non-fatal, just log
					fmt.Fprintf(os.Stderr, "Warning: failed to update module config for %s: %v\n", modulePath, err)
				}
			}
		}

		return nil
	})
}

// extractModulePath extracts the module path from a gitdir path
// e.g., "../../.git/modules/libs/mylib" -> "libs/mylib"
func extractModulePath(gitdirPath string) string {
	marker := ".git/modules/"
	idx := strings.Index(gitdirPath, marker)
	if idx == -1 {
		// Try with backslash for Windows compatibility
		marker = ".git\\modules\\"
		idx = strings.Index(gitdirPath, marker)
	}
	if idx == -1 {
		return ""
	}
	return gitdirPath[idx+len(marker):]
}

// updateModuleWorktreePath updates the core.worktree setting in a submodule's config
func updateModuleWorktreePath(moduleGitDir, submodulePath string) error {
	absSubmodulePath, err := filepath.Abs(submodulePath)
	if err != nil {
		return err
	}

	relWorktree, err := filepath.Rel(moduleGitDir, absSubmodulePath)
	if err != nil {
		return err
	}

	configFile := filepath.Join(moduleGitDir, "config")
	executor := git.NewExecutor(filepath.Dir(moduleGitDir))
	if _, err := executor.Execute("config", "-f", configFile, "core.worktree", relWorktree); err != nil {
		return err
	}

	return nil
}

// copyWorktreeDir copies a worktree directory, excluding the .git file
func copyWorktreeDir(src, dst string) error {
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

		// Skip .git file (will be recreated)
		if entry.Name() == ".git" {
			continue
		}

		// Get file info using Lstat to detect symlinks
		info, err := os.Lstat(srcPath)
		if err != nil {
			return err
		}

		// Handle symlinks first (before checking IsDir)
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(srcPath)
			if err != nil {
				return err
			}
			if err := os.Symlink(link, dstPath); err != nil {
				return err
			}
		} else if info.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
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
