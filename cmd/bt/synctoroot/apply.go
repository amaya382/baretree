package synctoroot

import (
	"fmt"

	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	applyForce bool
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply all sync-to-root entries",
	Long: `Apply all sync-to-root configurations, creating symlinks in the repository root.

Use this command after importing configuration or to recreate missing symlinks.

Examples:
  bt sync-to-root apply
  bt sync-to-root apply --force`,
	RunE: runSyncToRootApply,
}

func init() {
	applyCmd.Flags().BoolVar(&applyForce, "force", false, "Overwrite existing incorrect symlinks")
}

func runSyncToRootApply(cmd *cobra.Command, args []string) error {
	// Find repository root
	cwd, err := cmd.Flags().GetString("cwd")
	if err != nil || cwd == "" {
		cwd = "."
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("not in a baretree repository: %w", err)
	}

	// Get bare directory
	bareDir, err := repository.GetBareRepoPath(repoRoot)
	if err != nil {
		return err
	}

	// Load config
	repoMgr, err := repository.NewManager(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create worktree manager
	mgr := worktree.NewManager(repoRoot, bareDir, repoMgr.Config)

	if len(repoMgr.Config.SyncToRoot) == 0 {
		fmt.Println("No sync-to-root entries configured.")
		return nil
	}

	// Get default branch for display
	defaultBranch := mgr.GetDefaultBranch()

	fmt.Println("Applying sync-to-root configuration...")
	fmt.Println()

	// Apply all sync-to-root configs
	results, err := mgr.ApplyAllSyncToRoot(applyForce)
	if err != nil {
		return err
	}

	// Show results
	appliedCount := 0
	skippedCount := 0
	errorCount := 0

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

	fmt.Println()

	if errorCount > 0 {
		fmt.Printf("Applied %d, skipped %d, errors %d.\n", appliedCount, skippedCount, errorCount)
		return fmt.Errorf("%d error(s) occurred", errorCount)
	}

	fmt.Printf("+ Applied %d sync-to-root entry(s), skipped %d.\n", appliedCount, skippedCount)

	return nil
}
