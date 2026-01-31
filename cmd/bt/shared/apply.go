package shared

import (
	"errors"
	"fmt"

	"github.com/amaya382/baretree/internal/config"
	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	applyDryRun bool
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply shared configuration to all worktrees",
	Long: `Apply shared file configuration to all worktrees.

Use this command after adding shared configuration with 'bt shared add'
or after importing configuration with 'bt shared import'.

If any conflicts are detected (files already exist in worktrees),
the entire operation will fail and no changes will be made.

Examples:
  bt shared apply
  bt shared apply --dry-run`,
	RunE: runSharedApply,
}

func init() {
	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Show what would be done without making changes")
}

func runSharedApply(cmd *cobra.Command, args []string) error {
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
	cfg := repoMgr.Config

	if len(cfg.Shared) == 0 {
		fmt.Println("No shared files configured.")
		return nil
	}

	// Create worktree manager
	mgr := worktree.NewManager(repoRoot, bareDir, cfg)

	defaultBranch := mgr.GetDefaultBranch()

	if applyDryRun {
		fmt.Println("Dry run: showing what would be done...")
		fmt.Println()
	} else {
		fmt.Println("Applying shared configuration...")
		fmt.Println()
	}

	// Check for conflicts first
	var allConflicts []worktree.SharedConflict
	for _, shared := range cfg.Shared {
		conflicts, err := mgr.CheckSharedConflicts(shared.Source, shared.Managed)
		if err != nil {
			return err
		}
		allConflicts = append(allConflicts, conflicts...)
	}

	if len(allConflicts) > 0 {
		fmt.Println("Error: conflicts detected, no changes made")
		fmt.Println()
		printConflicts(allConflicts)
		fmt.Println()
		fmt.Println("To proceed, remove or rename conflicting files first.")
		return fmt.Errorf("conflicts detected")
	}

	if applyDryRun {
		// Just show what would be done
		for _, shared := range cfg.Shared {
			printSharedInfo(shared, defaultBranch)
		}
		fmt.Println("No changes made (dry run).")
		return nil
	}

	// Apply all shared configs
	results, err := mgr.ApplyAllShared()
	if err != nil {
		var conflictErr *worktree.SharedConflictError
		if errors.As(err, &conflictErr) {
			fmt.Println("Error: conflicts detected, no changes made")
			fmt.Println()
			printConflicts(conflictErr.Conflicts)
			fmt.Println()
			fmt.Println("To proceed, remove or rename conflicting files first.")
			return fmt.Errorf("conflicts detected")
		}
		return err
	}

	// Show results
	for _, result := range results {
		modeStr := ""
		if result.Managed {
			modeStr = ", managed"
		}
		fmt.Printf("%s (%s%s):\n", result.Source, result.Type, modeStr)

		if result.Managed {
			fmt.Printf("  Source: .shared/%s\n", result.Source)
		} else {
			fmt.Printf("  Source: %s/%s\n", defaultBranch, result.Source)
		}

		if len(result.Applied) > 0 {
			for _, wt := range result.Applied {
				if result.Managed {
					if result.Type == "symlink" {
						fmt.Printf("  + %s/%s -> ../.shared/%s\n", wt, result.Source, result.Source)
					} else {
						fmt.Printf("  + %s/%s (copy)\n", wt, result.Source)
					}
				} else {
					if result.Type == "symlink" {
						fmt.Printf("  + %s/%s -> ../%s/%s\n", wt, result.Source, defaultBranch, result.Source)
					} else {
						fmt.Printf("  + %s/%s (copy)\n", wt, result.Source)
					}
				}
			}
		}

		if len(result.Skipped) > 0 {
			for _, wt := range result.Skipped {
				fmt.Printf("  - %s/%s (already exists, skipped)\n", wt, result.Source)
			}
		}

		fmt.Println()
	}

	fmt.Printf("+ Applied %d shared configuration(s).\n", len(results))

	return nil
}

func printConflicts(conflicts []worktree.SharedConflict) {
	// Group by source
	bySource := make(map[string][]worktree.SharedConflict)
	for _, c := range conflicts {
		bySource[c.Source] = append(bySource[c.Source], c)
	}

	for source, cs := range bySource {
		fmt.Printf("%s:\n", source)
		for _, c := range cs {
			fmt.Printf("  x %s (file exists)\n", c.WorktreePath)
		}
		fmt.Println()
	}
}

func printSharedInfo(shared config.Shared, defaultBranch string) {
	modeStr := ""
	if shared.Managed {
		modeStr = ", managed"
	}
	fmt.Printf("%s (%s%s):\n", shared.Source, shared.Type, modeStr)

	if shared.Managed {
		fmt.Printf("  Source: %s/%s -> .shared/%s (move)\n", defaultBranch, shared.Source, shared.Source)
		fmt.Printf("  Would create symlinks in all worktrees\n")
	} else {
		fmt.Printf("  Source: %s/%s\n", defaultBranch, shared.Source)
		if shared.Type == "symlink" {
			fmt.Printf("  Would create symlinks in other worktrees\n")
		} else {
			fmt.Printf("  Would copy to other worktrees\n")
		}
	}
	fmt.Println()
}
