package synctoroot

import (
	"fmt"

	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/worktree"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove <source>",
	Aliases: []string{"rm"},
	Short:   "Remove a sync-to-root entry",
	Long: `Remove a sync-to-root entry and delete the symlink from the repository root.

Examples:
  bt sync-to-root remove CLAUDE.md
  bt sync-to-root rm .claude`,
	Args:              cobra.ExactArgs(1),
	RunE:              runSyncToRootRemove,
	ValidArgsFunction: completeConfiguredSources,
}

func runSyncToRootRemove(cmd *cobra.Command, args []string) error {
	source := args[0]

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

	fmt.Printf("Removing sync-to-root: %s\n\n", source)

	// Remove sync-to-root action
	if err := mgr.RemoveSyncToRoot(source); err != nil {
		return err
	}

	fmt.Println("+ Sync-to-root removed.")

	return nil
}
