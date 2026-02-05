package config

import (
	"fmt"
	"os"

	"github.com/amaya382/baretree/internal/config"
	"github.com/amaya382/baretree/internal/repository"
	"github.com/spf13/cobra"
)

var defaultBranchUnset bool

var defaultBranchCmd = &cobra.Command{
	Use:   "default-branch [branch]",
	Short: "Get or set the default branch",
	Long: `Get or set the default branch for the baretree repository.

The default branch is used to:
  - Identify the main worktree for post-create files
  - Determine the source for sync-to-root symlinks

Without arguments, displays the current default branch.
With a branch name argument, sets the default branch.
With --unset flag, removes the default branch setting (reverts to default 'main').

Examples:
  bt config default-branch              # Show current default branch
  bt config default-branch main         # Set default branch to 'main'
  bt config default-branch develop      # Set default branch to 'develop'
  bt config default-branch --unset      # Remove setting (reverts to 'main')`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDefaultBranch,
}

func init() {
	defaultBranchCmd.Flags().BoolVar(&defaultBranchUnset, "unset", false, "Remove the default branch setting (reverts to default 'main')")
	Cmd.AddCommand(defaultBranchCmd)
}

func runDefaultBranch(cmd *cobra.Command, args []string) error {
	// Find repository root
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("not in a baretree repository: %w", err)
	}

	// Unset mode: remove the default branch setting
	if defaultBranchUnset {
		if len(args) > 0 {
			return fmt.Errorf("cannot specify branch name with --unset flag")
		}
		if err := config.UnsetDefaultBranch(repoRoot); err != nil {
			return fmt.Errorf("failed to unset default branch: %w", err)
		}
		fmt.Println("Default branch setting removed (will use default 'main')")
		return nil
	}

	// Load config
	cfg, err := config.LoadConfig(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(args) == 0 {
		// Get mode: display current default branch
		fmt.Println(cfg.Repository.DefaultBranch)
		return nil
	}

	// Set mode: update default branch
	newBranch := args[0]
	cfg.Repository.DefaultBranch = newBranch

	if err := config.SaveConfig(repoRoot, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Default branch set to '%s'\n", newBranch)
	return nil
}
