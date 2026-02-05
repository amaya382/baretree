package synctoroot

import (
	"fmt"

	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/worktree"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List sync-to-root entries",
	Long: `List all sync-to-root entries and their status.

Examples:
  bt sync-to-root list
  bt sync-to-root ls`,
	RunE: runSyncToRootList,
}

func runSyncToRootList(cmd *cobra.Command, args []string) error {
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

	// Get status
	statuses, err := mgr.GetSyncToRootStatus()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	// Get default branch for display
	defaultBranch := mgr.GetDefaultBranch()

	fmt.Println("Sync-to-root entries:")

	// Calculate column widths
	maxSourceLen := 6 // "Source"
	maxTargetLen := 6 // "Target"
	for _, status := range statuses {
		if len(status.Source) > maxSourceLen {
			maxSourceLen = len(status.Source)
		}
		if len(status.Target) > maxTargetLen {
			maxTargetLen = len(status.Target)
		}
	}

	for _, status := range statuses {
		var stateStr string
		if !status.SourceExists {
			stateStr = "[MISSING SOURCE]"
		} else if !status.TargetExists {
			stateStr = "[NOT APPLIED]"
		} else if !status.IsCorrect {
			stateStr = "[WRONG TARGET]"
		} else {
			stateStr = "[OK]"
		}

		// Format the output
		if status.Source == status.Target {
			fmt.Printf("  %-*s  %-16s -> %s/%s\n",
				maxSourceLen, status.Source,
				stateStr,
				defaultBranch, status.Source,
			)
		} else {
			fmt.Printf("  %-*s  %-16s %s -> %s/%s\n",
				maxSourceLen, status.Source,
				stateStr,
				status.Target,
				defaultBranch, status.Source,
			)
		}
	}

	return nil
}
