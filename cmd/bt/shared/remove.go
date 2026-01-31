package shared

import (
	"fmt"
	"path/filepath"

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
	Short:   "Remove a shared file configuration",
	Long: `Remove a shared file or directory configuration.

By default, only symlinks are removed. Copied files are preserved.
Use --all to remove copied files as well.

Examples:
  bt shared remove .env
  bt shared rm .env
  bt shared remove config/local.json --all`,
	Args: cobra.ExactArgs(1),
	RunE: runSharedRemove,
}

func init() {
	removeCmd.Flags().BoolVar(&removeAll, "all", false, "Also remove copied files (not just symlinks)")
}

func runSharedRemove(cmd *cobra.Command, args []string) error {
	source := filepath.Clean(args[0])

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

	// Remove shared file
	result, err := mgr.RemoveShared(source, removeAll)
	if err != nil {
		return err
	}

	// Show results
	modeStr := ""
	if result.Managed {
		modeStr = ", managed"
	}
	fmt.Printf("Removing shared file: %s (%s%s)\n\n", source, result.Type, modeStr)

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
	fmt.Println("+ Shared configuration removed.")

	if len(result.SkippedCopies) > 0 {
		fmt.Println()
		fmt.Println("Note: Copied files were not removed. Use --all to remove them.")
	}

	return nil
}
