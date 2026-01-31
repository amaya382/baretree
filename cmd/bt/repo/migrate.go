package repo

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/amaya382/baretree/internal/git"
	"github.com/amaya382/baretree/internal/global"
	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/url"
	"github.com/spf13/cobra"
)

var (
	migrateDestination string
	migrateBareDir     string
	migrateInPlace     bool
	migrateToRoot      bool
	migrateRepoPath    string
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

You must specify one of --in-place, --destination, or --to-root:
  --in-place (-i): Replace the original repository in-place (recommended)
  --destination (-d): Create the baretree structure at a different location
  --to-root (-r): Move to baretree root with ghq-style path (host/user/repo)

The --to-root option:
  - Automatically detects the destination path from git remote URL
  - Use --path to manually specify the path (e.g., github.com/user/repo)
  - Works with both regular Git repositories and existing baretree repositories
  - Existing baretree repositories are moved without re-conversion

Examples:
  bt repo migrate /path/to/existing-repo --in-place
  bt repo migrate . -i
  bt repo migrate ~/projects/myapp --destination ../my-project-baretree
  bt repo migrate ~/projects/myapp -d ~/baretree/myapp
  bt repo migrate ~/projects/myapp --to-root
  bt repo migrate ~/projects/myapp -r --path github.com/user/myapp`,
	Args: cobra.ExactArgs(1),
	RunE: runMigrate,
}

func init() {
	migrateCmd.Flags().BoolVarP(&migrateInPlace, "in-place", "i", false, "Replace the original repository in-place (recommended)")
	migrateCmd.Flags().StringVarP(&migrateDestination, "destination", "d", "", "Destination directory for the new baretree structure")
	migrateCmd.Flags().StringVar(&migrateBareDir, "bare-dir", ".bare", "Bare repository directory name")
	migrateCmd.Flags().BoolVarP(&migrateToRoot, "to-root", "r", false, "Move repository to baretree root with ghq-style path")
	migrateCmd.Flags().StringVar(&migrateRepoPath, "path", "", "Repository path for --to-root (e.g., github.com/user/repo)")
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
	if migrateToRoot {
		modeCount++
	}

	// Validate that exactly one mode is specified
	if modeCount == 0 {
		return fmt.Errorf("you must specify one of --in-place (-i), --destination (-d), or --to-root (-r)")
	}
	if modeCount > 1 {
		return fmt.Errorf("cannot use multiple mode flags together")
	}

	// Validate --path is only used with --to-root
	if migrateRepoPath != "" && !migrateToRoot {
		return fmt.Errorf("--path can only be used with --to-root")
	}

	// Convert to absolute path
	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Handle --to-root mode
	if migrateToRoot {
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

	return performMigration(absSource, absDestination, currentBranch, migrateBareDir, migrateInPlace, externalWorktrees)
}

func performMigration(absSource, absDestination, currentBranch, bareDir string, inPlace bool, externalWorktrees []git.Worktree) error {
	if inPlace {
		return migrateInPlaceImpl(absSource, currentBranch, bareDir, externalWorktrees)
	}
	return migrateToDestination(absSource, absDestination, currentBranch, bareDir, externalWorktrees)
}

func migrateInPlaceImpl(absSource, currentBranch, bareDir string, externalWorktrees []git.Worktree) error {
	// For in-place migration:
	// 1. Move .git to .bare (or specified bareDir)
	// 2. Initialize baretree config in git-config
	// 3. Move all files except .bare to <branch>/ directory
	// 4. Convert .bare to a proper bare repo and set up worktree
	// 5. Move external worktrees into the baretree structure

	gitDir := filepath.Join(absSource, ".git")
	barePath := filepath.Join(absSource, bareDir)
	worktreePath := filepath.Join(absSource, currentBranch)

	// Step 1: Rename .git to bare directory
	fmt.Printf("Converting .git to bare repository at %s...\n", barePath)
	if err := os.Rename(gitDir, barePath); err != nil {
		return fmt.Errorf("failed to move .git to %s: %w", bareDir, err)
	}

	// Step 2: Convert to bare repository
	bareExecutor := git.NewExecutor(barePath)
	if _, err := bareExecutor.Execute("config", "--bool", "core.bare", "true"); err != nil {
		// Rollback
		if rollbackErr := os.Rename(barePath, gitDir); rollbackErr != nil {
			return fmt.Errorf("failed to set bare config and also failed to roll back: %w / %w", err, rollbackErr)
		}
		return fmt.Errorf("failed to set bare config: %w", err)
	}

	// Step 3: Create worktree directory
	if err := os.MkdirAll(worktreePath, 0755); err != nil {
		// Rollback
		if _, executeErr := bareExecutor.Execute("config", "--bool", "core.bare", "false"); executeErr != nil {
			return fmt.Errorf("failed to create worktree directory and also failed to roll back: %w /%w", err, executeErr)
		}
		if renameErr := os.Rename(barePath, gitDir); renameErr != nil {
			return fmt.Errorf("failed to create worktree directory and also failed to roll back: %w /%w", err, renameErr)
		}
		return fmt.Errorf("failed to create worktree directory: %w", err)
	}

	// Step 4: Move all files (except bare dir and worktree dir) to worktree
	entries, err := os.ReadDir(absSource)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if name == bareDir || name == currentBranch {
			continue
		}
		srcPath := filepath.Join(absSource, name)
		dstPath := filepath.Join(worktreePath, name)
		if err := os.Rename(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to move %s to worktree: %w", name, err)
		}
	}

	// Step 5: Set up worktree link
	// Create .git file in worktree pointing to bare repo's worktrees directory
	worktreeGitDir := filepath.Join(barePath, "worktrees", currentBranch)
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

	// Step 6: Initialize baretree config
	if err := repository.InitializeBareRepo(absSource, bareDir, currentBranch); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	// Step 7: Move external worktrees into the baretree structure
	movedWorktrees, err := migrateExternalWorktrees(absSource, barePath, bareExecutor, externalWorktrees)
	if err != nil {
		return fmt.Errorf("failed to migrate external worktrees: %w", err)
	}

	fmt.Printf("\n✓ Migration successful!\n")
	fmt.Printf("  Repository root: %s\n", absSource)
	fmt.Printf("  Bare repo: %s\n", barePath)
	fmt.Printf("  Worktree: %s\n", worktreePath)
	for _, wt := range movedWorktrees {
		fmt.Printf("  Worktree: %s\n", wt)
	}
	fmt.Printf("\nAll working tree state (unstaged, staged, untracked) has been preserved.\n")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  cd %s\n", worktreePath)
	fmt.Printf("  bt status\n")

	return nil
}

func migrateToDestination(absSource, absDestination, currentBranch, bareDir string, externalWorktrees []git.Worktree) error {
	// For destination migration:
	// 1. Create destination directory
	// 2. Copy .git as bare repository
	// 3. Create worktree directory and copy all working files
	// 4. Set up worktree links and preserve index

	barePath := filepath.Join(absDestination, bareDir)
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
	worktreeGitDir := filepath.Join(barePath, "worktrees", currentBranch)
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

	// Step 6: Initialize baretree config
	if err := repository.InitializeBareRepo(absDestination, bareDir, currentBranch); err != nil {
		os.RemoveAll(absDestination)
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	// Step 7: Copy and migrate external worktrees
	movedWorktrees, err := migrateExternalWorktreesWithCopy(absDestination, barePath, bareExecutor, externalWorktrees)
	if err != nil {
		os.RemoveAll(absDestination)
		return fmt.Errorf("failed to migrate external worktrees: %w", err)
	}

	fmt.Printf("\n✓ Migration successful!\n")
	fmt.Printf("  New repository: %s\n", absDestination)
	fmt.Printf("  Bare repo: %s\n", barePath)
	fmt.Printf("  Worktree: %s\n", worktreePath)
	for _, wt := range movedWorktrees {
		fmt.Printf("  Worktree: %s\n", wt)
	}
	fmt.Printf("\nAll working tree state (unstaged, staged, untracked) has been preserved.\n")
	fmt.Printf("\nOriginal repository preserved at: %s\n", absSource)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  cd %s\n", worktreePath)
	fmt.Printf("  bt status\n")
	fmt.Printf("\nIf migration is successful, you can remove the original:\n")
	fmt.Printf("  rm -rf %s\n", absSource)

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
		return moveBaretreeRepo(absSource, absDestination)
	}

	// Regular git repository - migrate and move
	return migrateToRootImpl(absSource, absDestination)
}

// findBareDir finds the bare repository directory in a baretree repo
func findBareDir(repoRoot string) (string, error) {
	// Try common bare directory names
	commonNames := []string{".bare", "bare"}
	for _, name := range commonNames {
		barePath := filepath.Join(repoRoot, name)
		if info, err := os.Stat(barePath); err == nil && info.IsDir() {
			// Verify it's a git directory
			if _, err := os.Stat(filepath.Join(barePath, "HEAD")); err == nil {
				return barePath, nil
			}
		}
	}

	// Search for any directory that looks like a bare repo
	entries, err := os.ReadDir(repoRoot)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		barePath := filepath.Join(repoRoot, entry.Name())
		if _, err := os.Stat(filepath.Join(barePath, "HEAD")); err == nil {
			executor := git.NewExecutor(barePath)
			isBare, _ := executor.Execute("rev-parse", "--is-bare-repository")
			if isBare == "true" {
				return barePath, nil
			}
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
func moveBaretreeRepo(absSource, absDestination string) error {
	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(absDestination), 0755); err != nil {
		return fmt.Errorf("failed to create destination parent directory: %w", err)
	}

	fmt.Printf("Moving baretree repository...\n")

	// Move the entire directory
	if err := os.Rename(absSource, absDestination); err != nil {
		// If rename fails (cross-device), fall back to copy+delete
		fmt.Printf("Direct move failed, copying instead...\n")
		if err := copyDir(absSource, absDestination); err != nil {
			return fmt.Errorf("failed to copy repository: %w", err)
		}

		// Update worktree paths after copy
		if err := updateWorktreePaths(absDestination); err != nil {
			os.RemoveAll(absDestination)
			return fmt.Errorf("failed to update worktree paths: %w", err)
		}

		// Remove original
		if err := os.RemoveAll(absSource); err != nil {
			fmt.Printf("Warning: failed to remove original directory: %v\n", err)
		}
	} else {
		// Update worktree paths after move
		if err := updateWorktreePaths(absDestination); err != nil {
			// Try to move back on failure
			if rollbackErr := os.Rename(absDestination, absSource); rollbackErr != nil {
				return fmt.Errorf("failed to update worktree paths and also failed to roll back: %w / %w", err, rollbackErr)
			}
			return fmt.Errorf("failed to update worktree paths: %w", err)
		}
	}

	fmt.Printf("\n✓ Repository moved successfully!\n")
	fmt.Printf("  New location: %s\n", absDestination)

	return nil
}

// updateWorktreePaths updates all worktree gitdir paths after moving a baretree repo
func updateWorktreePaths(repoRoot string) error {
	bareDir, err := findBareDir(repoRoot)
	if err != nil {
		return err
	}

	worktreesDir := filepath.Join(bareDir, "worktrees")
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
		worktreeGitDir := filepath.Join(worktreesDir, worktreeName)
		worktreePath := filepath.Join(repoRoot, worktreeName)

		// Update gitdir file in bare repo's worktrees directory
		gitdirFile := filepath.Join(worktreeGitDir, "gitdir")
		newGitdir := filepath.Join(worktreePath, ".git") + "\n"
		if err := os.WriteFile(gitdirFile, []byte(newGitdir), 0644); err != nil {
			return fmt.Errorf("failed to update gitdir for %s: %w", worktreeName, err)
		}

		// Update .git file in worktree directory
		worktreeGitFile := filepath.Join(worktreePath, ".git")
		if _, err := os.Stat(worktreeGitFile); err == nil {
			newContent := fmt.Sprintf("gitdir: %s\n", worktreeGitDir)
			if err := os.WriteFile(worktreeGitFile, []byte(newContent), 0644); err != nil {
				return fmt.Errorf("failed to update .git file for %s: %w", worktreeName, err)
			}
		}
	}

	return nil
}

// migrateToRootImpl migrates a regular git repository to baretree root
func migrateToRootImpl(absSource, absDestination string) error {
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

	fmt.Printf("Current branch: %s\n", currentBranch)

	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(absDestination), 0755); err != nil {
		return fmt.Errorf("failed to create destination parent directory: %w", err)
	}

	// Move source to destination first
	if err := os.Rename(absSource, absDestination); err != nil {
		// If rename fails (cross-device), fall back to copy+delete
		fmt.Printf("Direct move failed, copying instead...\n")
		if err := copyDir(absSource, absDestination); err != nil {
			return fmt.Errorf("failed to copy repository: %w", err)
		}
		// Will remove original after successful migration
	}

	// Now perform in-place migration at destination (no external worktrees for --to-root)
	if err := migrateInPlaceImpl(absDestination, currentBranch, migrateBareDir, nil); err != nil {
		// Try to restore on failure
		if rollbackErr := os.Rename(absDestination, absSource); rollbackErr != nil {
			return fmt.Errorf("failed to migrate repository in place and also failed to roll back: %w / %w", err, rollbackErr)
		}
		return fmt.Errorf("failed to migrate repository in place: %w", err)
	}

	// If we copied instead of moved, remove the original
	if _, err := os.Stat(absSource); err == nil {
		if err := os.RemoveAll(absSource); err != nil {
			fmt.Printf("Warning: failed to remove original directory: %v\n", err)
		}
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

		// Set up worktree link in the new location
		worktreeGitDir := filepath.Join(barePath, "worktrees", wt.Branch)

		// The worktree git dir should already exist from the bare repo copy
		// but we need to update the paths
		if err := os.MkdirAll(worktreeGitDir, 0755); err != nil {
			return movedWorktrees, fmt.Errorf("failed to create worktree git dir for %s: %w", wt.Branch, err)
		}

		// Create/update .git file in worktree
		gitFileContent := fmt.Sprintf("gitdir: %s\n", worktreeGitDir)
		if err := os.WriteFile(filepath.Join(targetPath, ".git"), []byte(gitFileContent), 0644); err != nil {
			return movedWorktrees, fmt.Errorf("failed to create .git file for %s: %w", wt.Branch, err)
		}

		// Update commondir file
		relBare, _ := filepath.Rel(worktreeGitDir, barePath)
		if err := os.WriteFile(filepath.Join(worktreeGitDir, "commondir"), []byte(relBare+"\n"), 0644); err != nil {
			return movedWorktrees, fmt.Errorf("failed to create commondir for %s: %w", wt.Branch, err)
		}

		// Update gitdir file
		absTargetPath, _ := filepath.Abs(targetPath)
		if err := os.WriteFile(filepath.Join(worktreeGitDir, "gitdir"), []byte(absTargetPath+"/.git\n"), 0644); err != nil {
			return movedWorktrees, fmt.Errorf("failed to create gitdir for %s: %w", wt.Branch, err)
		}

		// Create/update HEAD file
		if err := os.WriteFile(filepath.Join(worktreeGitDir, "HEAD"), []byte("ref: refs/heads/"+wt.Branch+"\n"), 0644); err != nil {
			return movedWorktrees, fmt.Errorf("failed to create HEAD for %s: %w", wt.Branch, err)
		}

		movedWorktrees = append(movedWorktrees, targetPath)
	}

	return movedWorktrees, nil
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
