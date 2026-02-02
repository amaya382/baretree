package repo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/amaya382/baretree/internal/config"
	"github.com/amaya382/baretree/internal/git"
	"github.com/amaya382/baretree/internal/global"
	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/url"
	"github.com/spf13/cobra"
)

var (
	getBranch  string
	getShallow bool
	getUpdate  bool
)

var getCmd = &cobra.Command{
	Use:   "get <repository>",
	Short: "Clone a repository into root directory [bt get]",
	Long: `Clone a repository into the baretree root directory with ghq-style path structure.

The repository is cloned into {root}/{host}/{user}/{repo}/ with baretree structure.

Supports various input formats:
  - SSH URL: git@github.com:user/repo.git
  - HTTPS URL: https://github.com/user/repo.git
  - Short path: github.com/user/repo
  - User/repo: user/repo (uses github.com as default host)
  - Repo only: repo (uses configured user and github.com)

Examples:
  bt repo get github.com/amaya382/baretree
  bt repo get git@github.com:amaya382/baretree.git
  bt repo get amaya382/dotfiles
  bt repo get --branch develop github.com/user/repo`,
	Args: cobra.ExactArgs(1),
	RunE: runGet,
}

func init() {
	getCmd.Flags().StringVarP(&getBranch, "branch", "b", "", "Checkout specific branch")
	getCmd.Flags().BoolVar(&getShallow, "shallow", false, "Perform a shallow clone")
	getCmd.Flags().BoolVarP(&getUpdate, "update", "u", false, "Update existing repository")
}

func runGet(cmd *cobra.Command, args []string) error {
	cfg, err := global.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Parse repository path
	repoPath, err := url.Parse(args[0], "github.com", cfg.User)
	if err != nil {
		return fmt.Errorf("failed to parse repository: %w", err)
	}

	// Build destination path: {root}/{host}/{user}/{repo}
	destination := filepath.Join(cfg.PrimaryRoot(), repoPath.String())

	// Check if repository already exists
	if _, err := os.Stat(destination); err == nil {
		if !getUpdate {
			return fmt.Errorf("repository already exists: %s (use -u to update)", destination)
		}
		return updateRepository(destination)
	}

	// Build clone URL
	cloneURL := buildCloneURL(repoPath)

	fmt.Printf("Cloning %s into %s...\n", cloneURL, destination)

	// Create destination directory
	if err := os.MkdirAll(destination, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Clone as bare repository
	barePath := filepath.Join(destination, config.BareDir)
	fmt.Printf("Creating bare repository at %s...\n", barePath)

	cloneArgs := []string{"--bare"}
	if getShallow {
		cloneArgs = append(cloneArgs, "--depth", "1")
	}
	cloneArgs = append(cloneArgs, cloneURL, barePath)

	if err := git.Clone(cloneArgs...); err != nil {
		// Cleanup on failure
		os.RemoveAll(destination)
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Determine default branch
	var defaultBranch string
	if getBranch != "" {
		defaultBranch = getBranch
	} else {
		defaultBranch, err = git.GetDefaultBranch(barePath)
		if err != nil {
			fmt.Printf("Warning: failed to detect default branch, using 'main': %v\n", err)
			defaultBranch = "main"
		}
	}

	fmt.Printf("Default branch: %s\n", defaultBranch)

	// Initialize baretree config
	if err := repository.InitializeBareRepo(destination, defaultBranch); err != nil {
		return fmt.Errorf("failed to initialize baretree config: %w", err)
	}

	// Create default worktree
	defaultWorktreePath := filepath.Join(destination, defaultBranch)
	fmt.Printf("Creating worktree for %s at %s...\n", defaultBranch, defaultWorktreePath)

	executor := git.NewExecutor(barePath)
	if _, err := executor.Execute("worktree", "add", defaultWorktreePath, defaultBranch); err != nil {
		return fmt.Errorf("failed to create default worktree: %w", err)
	}

	fmt.Printf("\n✓ Successfully cloned repository\n")
	fmt.Printf("  Repository: %s\n", destination)
	fmt.Printf("  Worktree: %s\n", defaultWorktreePath)
	fmt.Printf("\nTo navigate to this repository:\n")
	fmt.Printf("  bt go %s\n", repoPath.String())

	return nil
}

func buildCloneURL(repoPath *url.RepoPath) string {
	// Use SSH URL format for github.com, gitlab.com, etc.
	return fmt.Sprintf("git@%s:%s/%s.git", repoPath.Host, repoPath.User, repoPath.Repo)
}

func updateRepository(destination string) error {
	barePath := filepath.Join(destination, config.BareDir)

	// Check if bare repository exists
	if _, err := os.Stat(barePath); os.IsNotExist(err) {
		return fmt.Errorf("bare repository not found at %s", barePath)
	}

	fmt.Printf("Updating repository at %s...\n", destination)

	executor := git.NewExecutor(barePath)
	if _, err := executor.Execute("fetch", "--all", "--prune"); err != nil {
		return fmt.Errorf("failed to fetch updates: %w", err)
	}

	fmt.Printf("✓ Repository updated\n")
	return nil
}
