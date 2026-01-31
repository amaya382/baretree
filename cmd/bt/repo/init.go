package repo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/amaya382/baretree/internal/git"
	"github.com/amaya382/baretree/internal/repository"
	"github.com/spf13/cobra"
)

var (
	initBareDir       string
	initDefaultBranch string
)

var initCmd = &cobra.Command{
	Use:   "init [directory]",
	Short: "Create a new baretree repository [bt init]",
	Long: `Create a new Git repository with baretree structure.

This command creates a new bare repository (.bare) and sets up the baretree
directory structure with a default branch worktree.

If no directory is specified, the current directory is used.

Examples:
  bt repo init                      # Initialize in current directory
  bt repo init my-project           # Initialize in my-project directory
  bt repo init . --branch develop   # Use 'develop' as default branch`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&initBareDir, "bare-dir", ".bare", "Bare repository directory name")
	initCmd.Flags().StringVarP(&initDefaultBranch, "branch", "b", "main", "Default branch name")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check git user configuration (required for creating commits)
	if err := git.CheckUserConfig(); err != nil {
		return err
	}

	// Determine target directory
	var targetDir string
	if len(args) == 1 {
		targetDir = args[0]
	} else {
		var err error
		targetDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Convert to absolute path
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if directory exists, create if not
	if _, err := os.Stat(absTarget); os.IsNotExist(err) {
		if err := os.MkdirAll(absTarget, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		fmt.Printf("Created directory %s\n", absTarget)
	}

	// Check if already a baretree repository
	if repository.IsBaretreeRepo(absTarget) {
		return fmt.Errorf("directory is already a baretree repository: %s", absTarget)
	}

	// Check if already a git repository
	gitDir := filepath.Join(absTarget, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return fmt.Errorf("directory is already a git repository (use 'bt repo migrate' instead): %s", absTarget)
	}

	barePath := filepath.Join(absTarget, initBareDir)
	defaultWorktreePath := filepath.Join(absTarget, initDefaultBranch)

	// Check if bare directory already exists
	if _, err := os.Stat(barePath); err == nil {
		return fmt.Errorf("bare directory already exists: %s", barePath)
	}

	// Check if default worktree directory already exists
	if _, err := os.Stat(defaultWorktreePath); err == nil {
		return fmt.Errorf("default worktree directory already exists: %s", defaultWorktreePath)
	}

	// Collect existing files in target directory (before creating baretree structure)
	existingFiles, err := collectExistingFiles(absTarget)
	if err != nil {
		return fmt.Errorf("failed to scan existing files: %w", err)
	}

	fmt.Printf("Initializing baretree repository in %s...\n", absTarget)

	// Initialize bare repository
	fmt.Printf("Creating bare repository at %s...\n", barePath)
	if err := os.MkdirAll(barePath, 0755); err != nil {
		return fmt.Errorf("failed to create bare directory: %w", err)
	}

	executor := git.NewExecutor(barePath)
	if _, err := executor.Execute("init", "--bare"); err != nil {
		return fmt.Errorf("failed to initialize bare repository: %w", err)
	}

	// Initialize baretree config
	if err := repository.InitializeBareRepo(absTarget, initBareDir, initDefaultBranch); err != nil {
		return fmt.Errorf("failed to initialize baretree config: %w", err)
	}

	// Create default worktree with initial commit
	fmt.Printf("Creating worktree for %s at %s...\n", initDefaultBranch, defaultWorktreePath)

	// Use a temporary directory for initial git operations
	tempInitDir := filepath.Join(absTarget, ".bt-init-temp")
	if err := os.MkdirAll(tempInitDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempInitDir) // Clean up temp dir

	// Initialize git in temp directory
	wtExecutor := git.NewExecutor(tempInitDir)
	if _, err := wtExecutor.Execute("init"); err != nil {
		return fmt.Errorf("failed to initialize git: %w", err)
	}

	// Create initial commit (empty)
	if _, err := wtExecutor.Execute("commit", "--allow-empty", "-m", "Initial commit"); err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}

	// Rename branch to default branch name if needed
	if _, err := wtExecutor.Execute("branch", "-M", initDefaultBranch); err != nil {
		return fmt.Errorf("failed to rename branch: %w", err)
	}

	// Push to bare repository
	if _, err := wtExecutor.Execute("remote", "add", "origin", barePath); err != nil {
		return fmt.Errorf("failed to add remote: %w", err)
	}

	if _, err := wtExecutor.Execute("push", "-u", "origin", initDefaultBranch); err != nil {
		return fmt.Errorf("failed to push to bare repository: %w", err)
	}

	// Update bare repository HEAD to point to default branch
	if _, err := executor.Execute("symbolic-ref", "HEAD", "refs/heads/"+initDefaultBranch); err != nil {
		return fmt.Errorf("failed to set HEAD: %w", err)
	}

	// Add worktree from bare repository
	if _, err := executor.Execute("worktree", "add", defaultWorktreePath, initDefaultBranch); err != nil {
		return fmt.Errorf("failed to add worktree: %w", err)
	}

	// Move existing files to worktree directory
	if len(existingFiles) > 0 {
		fmt.Printf("Moving %d existing file(s) to worktree...\n", len(existingFiles))
		for _, file := range existingFiles {
			src := filepath.Join(absTarget, file)
			dst := filepath.Join(defaultWorktreePath, file)

			// Create parent directory if needed
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", file, err)
			}

			if err := os.Rename(src, dst); err != nil {
				return fmt.Errorf("failed to move %s: %w", file, err)
			}
		}
	}

	fmt.Printf("\n✓ Successfully initialized baretree repository\n")
	fmt.Printf("  Repository root: %s\n", absTarget)
	fmt.Printf("  Bare repository: %s\n", barePath)
	fmt.Printf("  Default worktree: %s\n", defaultWorktreePath)
	if len(existingFiles) > 0 {
		fmt.Printf("  Moved files: %d\n", len(existingFiles))
	}
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  cd %s\n", defaultWorktreePath)
	if len(existingFiles) > 0 {
		fmt.Printf("  git add . && git commit -m \"Add existing files\"\n")
	} else {
		fmt.Printf("  # Add your files and commit\n")
	}
	fmt.Printf("  bt add -b <branch-name>  # Create a new worktree\n")

	return nil
}

// collectExistingFiles returns a list of files/directories in the target directory
func collectExistingFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		name := entry.Name()
		files = append(files, name)
	}

	return files, nil
}
