package postcreate

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
	Short: "Apply post-create file actions to all worktrees",
	Long: `Apply post-create file configuration to all worktrees.

Use this command after adding post-create configuration with 'bt post-create add'
or after importing configuration with 'bt config import'.

Note: Commands are NOT executed by this operation. Commands are only
executed when a new worktree is created with 'bt add'.

If any conflicts are detected (files already exist in worktrees),
the entire operation will fail and no changes will be made.

Examples:
  bt post-create apply
  bt post-create apply --dry-run`,
	RunE: runPostCreateApply,
}

func init() {
	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Show what would be done without making changes")
}

func runPostCreateApply(cmd *cobra.Command, args []string) error {
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

	// Count file-based actions
	fileActionCount := 0
	for _, action := range cfg.PostCreate {
		if action.Type != "command" {
			fileActionCount++
		}
	}

	if fileActionCount == 0 {
		fmt.Println("No file-based post-create actions configured.")
		fmt.Println("Note: Command actions are only executed when creating new worktrees.")
		return nil
	}

	// Create worktree manager
	mgr := worktree.NewManager(repoRoot, bareDir, cfg)

	defaultBranch := mgr.GetDefaultBranch()

	if applyDryRun {
		fmt.Println("Dry run: showing what would be done...")
		fmt.Println()
	} else {
		fmt.Println("Applying post-create configuration...")
		fmt.Println()
	}

	// Check for conflicts first
	var allConflicts []worktree.PostCreateConflict
	for _, action := range cfg.PostCreate {
		if action.Type == "command" {
			continue
		}
		conflicts, err := mgr.CheckPostCreateConflicts(action.Source, action.Managed)
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
		for _, action := range cfg.PostCreate {
			if action.Type == "command" {
				continue
			}
			printPostCreateInfo(action, defaultBranch)
		}
		fmt.Println("No changes made (dry run).")
		return nil
	}

	// Apply all post-create configs
	results, err := mgr.ApplyAllPostCreate()
	if err != nil {
		var conflictErr *worktree.PostCreateConflictError
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
	appliedCount := 0
	for _, result := range results {
		if result.Type == "command" {
			continue
		}
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
		appliedCount++
	}

	fmt.Printf("+ Applied %d post-create action(s).\n", appliedCount)

	return nil
}

func printConflicts(conflicts []worktree.PostCreateConflict) {
	// Group by source
	bySource := make(map[string][]worktree.PostCreateConflict)
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

func printPostCreateInfo(action config.PostCreateAction, defaultBranch string) {
	modeStr := ""
	if action.Managed {
		modeStr = ", managed"
	}
	fmt.Printf("%s (%s%s):\n", action.Source, action.Type, modeStr)

	if action.Managed {
		fmt.Printf("  Source: %s/%s -> .shared/%s (move)\n", defaultBranch, action.Source, action.Source)
		fmt.Printf("  Would create symlinks in all worktrees\n")
	} else {
		fmt.Printf("  Source: %s/%s\n", defaultBranch, action.Source)
		if action.Type == "symlink" {
			fmt.Printf("  Would create symlinks in other worktrees\n")
		} else {
			fmt.Printf("  Would copy to other worktrees\n")
		}
	}
	fmt.Println()
}
