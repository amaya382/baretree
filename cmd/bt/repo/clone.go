package repo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/amaya382/baretree/internal/config"
	"github.com/amaya382/baretree/internal/git"
	"github.com/amaya382/baretree/internal/repository"
	"github.com/spf13/cobra"
)

var (
	cloneBranch string
)

var cloneCmd = &cobra.Command{
	Use:   "clone <repository-url> [destination]",
	Short: "Clone a remote repository with baretree structure [bt clone]",
	Long: `Clone a Git repository as a bare repository (.git) and automatically create
a worktree for the default branch. Initializes baretree configuration.

Example:
  bt repo clone git@github.com:user/repo.git my-project
  bt repo clone https://github.com/user/repo.git
  bt repo clone git@github.com:user/repo.git --branch develop`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runClone,
}

func init() {
	cloneCmd.Flags().StringVarP(&cloneBranch, "branch", "b", "", "Checkout specific branch instead of default")
}

func runClone(cmd *cobra.Command, args []string) error {
	repoURL := args[0]

	// Determine destination
	var destination string
	if len(args) == 2 {
		destination = args[1]
	} else {
		destination = repository.ExtractRepoName(repoURL)
	}

	// Convert to absolute path
	absDestination, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if destination already exists
	if _, err := os.Stat(absDestination); err == nil {
		return fmt.Errorf("destination already exists: %s", absDestination)
	}

	fmt.Printf("Cloning %s into %s...\n", repoURL, absDestination)

	// Create repository root directory
	if err := os.MkdirAll(absDestination, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Clone as bare repository
	barePath := filepath.Join(absDestination, config.BareDir)
	fmt.Printf("Creating bare repository at %s...\n", barePath)

	if err := git.Clone("--bare", repoURL, barePath); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Determine default branch
	var defaultBranch string
	if cloneBranch != "" {
		defaultBranch = cloneBranch
	} else {
		defaultBranch, err = git.GetDefaultBranch(barePath)
		if err != nil {
			fmt.Printf("Warning: failed to detect default branch, using 'main': %v\n", err)
			defaultBranch = "main"
		}
	}

	fmt.Printf("Default branch: %s\n", defaultBranch)

	// Initialize baretree config
	if err := repository.InitializeBareRepo(absDestination, defaultBranch); err != nil {
		return fmt.Errorf("failed to initialize baretree config: %w", err)
	}

	// Create default worktree
	defaultWorktreePath := filepath.Join(absDestination, defaultBranch)
	fmt.Printf("Creating worktree for %s at %s...\n", defaultBranch, defaultWorktreePath)

	executor := git.NewExecutor(barePath)
	if _, err := executor.Execute("worktree", "add", defaultWorktreePath, defaultBranch); err != nil {
		return fmt.Errorf("failed to create default worktree: %w", err)
	}

	fmt.Printf("\n✓ Successfully cloned repository\n")
	fmt.Printf("  Repository root: %s\n", absDestination)
	fmt.Printf("  Bare repository: %s\n", barePath)
	fmt.Printf("  Default worktree: %s\n", defaultWorktreePath)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  cd %s\n", absDestination)
	fmt.Printf("  bt add -b <branch-name>  # Create a new worktree\n")
	fmt.Printf("  bt list                  # List all worktrees\n")

	return nil
}
