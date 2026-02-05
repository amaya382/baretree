package config

import (
	"fmt"
	"os"

	"github.com/amaya382/baretree/internal/config"
	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/worktree"
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

	// Get bare directory for worktree manager
	bareDir, err := repository.GetBareRepoPath(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to get bare repo path: %w", err)
	}

	// Unset mode: remove the default branch setting
	if defaultBranchUnset {
		if len(args) > 0 {
			return fmt.Errorf("cannot specify branch name with --unset flag")
		}

		// Load config before unset to check for sync-to-root entries
		cfg, err := config.LoadConfig(repoRoot)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		hasSyncToRoot := len(cfg.SyncToRoot) > 0

		if err := config.UnsetDefaultBranch(repoRoot); err != nil {
			return fmt.Errorf("failed to unset default branch: %w", err)
		}
		fmt.Println("Default branch setting removed (will use default 'main')")

		// Re-apply sync-to-root if configured
		if hasSyncToRoot {
			if err := reapplySyncToRoot(repoRoot, bareDir); err != nil {
				fmt.Printf("Warning: failed to update sync-to-root symlinks: %v\n", err)
			}
		}

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
	oldBranch := cfg.Repository.DefaultBranch
	if oldBranch == "" {
		oldBranch = "main"
	}

	// Check if branch is actually changing
	if newBranch == oldBranch {
		fmt.Printf("Default branch is already '%s'\n", newBranch)
		return nil
	}

	cfg.Repository.DefaultBranch = newBranch

	if err := config.SaveConfig(repoRoot, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Default branch set to '%s'\n", newBranch)

	// Re-apply sync-to-root if configured
	if len(cfg.SyncToRoot) > 0 {
		if err := reapplySyncToRoot(repoRoot, bareDir); err != nil {
			fmt.Printf("Warning: failed to update sync-to-root symlinks: %v\n", err)
		}
	}

	return nil
}

// reapplySyncToRoot re-applies sync-to-root symlinks after default branch change
func reapplySyncToRoot(repoRoot, bareDir string) error {
	// Reload config to get updated default branch
	cfg, err := config.LoadConfig(repoRoot)
	if err != nil {
		return err
	}

	if len(cfg.SyncToRoot) == 0 {
		return nil
	}

	// Create worktree manager with updated config
	mgr := worktree.NewManager(repoRoot, bareDir, cfg)

	fmt.Println()
	fmt.Println("Updating sync-to-root symlinks...")

	// Apply with force to update existing symlinks
	results, err := mgr.ApplyAllSyncToRoot(true)
	if err != nil {
		return err
	}

	// Show results
	appliedCount := 0
	skippedCount := 0
	errorCount := 0
	defaultBranch := mgr.GetDefaultBranch()

	for _, result := range results {
		target := result.Target
		if target == "" {
			target = result.Source
		}

		if result.Error != "" {
			fmt.Printf("  x %s: %s\n", result.Source, result.Error)
			errorCount++
		} else if result.Applied {
			fmt.Printf("  + %s -> %s/%s\n", target, defaultBranch, result.Source)
			appliedCount++
		} else if result.Skipped {
			fmt.Printf("  - %s (already correct)\n", target)
			skippedCount++
		}
	}

	if errorCount > 0 {
		return fmt.Errorf("%d sync-to-root error(s) occurred", errorCount)
	}

	fmt.Printf("Updated %d sync-to-root symlink(s), skipped %d.\n", appliedCount, skippedCount)
	return nil
}
