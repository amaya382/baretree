package synctoroot

import (
	"fmt"
	"path/filepath"

	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	addForce bool
)

var addCmd = &cobra.Command{
	Use:   "add <source> [target]",
	Short: "Add a sync-to-root entry",
	Long: `Add a file or directory to be symlinked from the default branch worktree to the repository root.

The source path is relative to the default branch worktree.
The target path is relative to the repository root (defaults to source if not specified).

Examples:
  bt sync-to-root add CLAUDE.md
  bt sync-to-root add .claude
  bt sync-to-root add docs/guide.md guide.md`,
	Args:              cobra.RangeArgs(1, 2),
	RunE:              runSyncToRootAdd,
	ValidArgsFunction: completeSourceFiles,
}

func init() {
	addCmd.Flags().BoolVar(&addForce, "force", false, "Overwrite existing incorrect symlinks")
}

func runSyncToRootAdd(cmd *cobra.Command, args []string) error {
	source := filepath.Clean(args[0])
	target := ""
	if len(args) >= 2 {
		target = filepath.Clean(args[1])
	}

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

	// Get default branch for display
	defaultBranch := mgr.GetDefaultBranch()

	// Determine display target
	displayTarget := target
	if displayTarget == "" {
		displayTarget = source
	}

	// Show what will happen
	if target != "" && target != source {
		fmt.Printf("Adding sync-to-root: %s -> %s\n", source, displayTarget)
	} else {
		fmt.Printf("Adding sync-to-root: %s\n", source)
	}
	fmt.Printf("  Source: %s/%s\n", defaultBranch, source)
	fmt.Printf("  Target: %s -> %s/%s\n", displayTarget, defaultBranch, source)
	fmt.Println()

	// Add sync-to-root action
	result, err := mgr.AddSyncToRoot(source, target, addForce)
	if err != nil {
		return err
	}

	// Show results
	if result.Applied {
		fmt.Println("+ Sync-to-root added and applied.")
	} else if result.Skipped {
		fmt.Println("+ Sync-to-root added (symlink already exists).")
	}

	return nil
}
