package postcreate

import (
	"fmt"

	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	removeAll bool
)

var removeCmd = &cobra.Command{
	Use:     "remove <source>",
	Aliases: []string{"rm"},
	Short:   "Remove a post-create action",
	Long: `Remove a post-create action configuration.

For file-based actions (symlink/copy):
  By default, only symlinks are removed. Copied files are preserved.
  Use --all to remove copied files as well.

For command actions:
  The command is simply removed from the configuration.

Examples:
  bt post-create remove .env
  bt post-create rm .env
  bt post-create remove config/local.json --all
  bt post-create remove "direnv allow"`,
	Args: cobra.ExactArgs(1),
	RunE: runPostCreateRemove,
}

func init() {
	removeCmd.Flags().BoolVar(&removeAll, "all", false, "Also remove copied files (not just symlinks)")
}

func runPostCreateRemove(cmd *cobra.Command, args []string) error {
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

	// Remove post-create action
	result, err := mgr.RemovePostCreate(source, removeAll)
	if err != nil {
		return err
	}

	// Show results
	if result.Type == "command" {
		fmt.Printf("Removing post-create command: %s\n\n", source)
		fmt.Println("+ Post-create command removed.")
	} else {
		modeStr := ""
		if result.Managed {
			modeStr = ", managed"
		}
		fmt.Printf("Removing post-create action: %s (%s%s)\n\n", source, result.Type, modeStr)

		for _, wt := range result.RemovedSymlinks {
			fmt.Printf("  Removing symlink: %s/%s\n", wt, source)
		}

		for _, wt := range result.RemovedCopies {
			fmt.Printf("  Removing copy: %s/%s\n", wt, source)
		}

		for _, wt := range result.SkippedCopies {
			fmt.Printf("  Skipping: %s/%s (copy, not removed)\n", wt, source)
		}

		fmt.Println()
		fmt.Println("+ Post-create action removed.")

		if len(result.SkippedCopies) > 0 {
			fmt.Println()
			fmt.Println("Note: Copied files were not removed. Use --all to remove them.")
		}
	}

	return nil
}
