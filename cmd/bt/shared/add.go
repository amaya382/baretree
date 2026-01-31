package shared

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	addType      string
	addNoManaged bool
)

var addCmd = &cobra.Command{
	Use:   "add <source>",
	Short: "Add a shared file or directory",
	Long: `Add a file or directory to be shared across all worktrees.

The source file must exist in the default branch worktree (usually main).

Modes:
  - Managed (default): File is moved to .shared/ directory, independent of any worktree
  - Non-managed (--no-managed): File is sourced from the default branch worktree

Examples:
  bt shared add .env --type symlink
  bt shared add .env --type symlink --no-managed
  bt shared add config/local.json --type copy`,
	Args: cobra.ExactArgs(1),
	RunE: runSharedAdd,
}

func init() {
	addCmd.Flags().StringVarP(&addType, "type", "t", "symlink", "Type of sharing: symlink or copy")
	addCmd.Flags().BoolVar(&addNoManaged, "no-managed", false, "Source file from the default branch worktree instead of .shared/ directory")
}

func runSharedAdd(cmd *cobra.Command, args []string) error {
	source := filepath.Clean(args[0])

	// Validate type
	if addType != "symlink" && addType != "copy" {
		return fmt.Errorf("invalid type: %s (must be 'symlink' or 'copy')", addType)
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

	// Determine managed mode (default is managed, --no-managed disables it)
	managed := !addNoManaged

	// Show what will happen
	defaultBranch := mgr.GetDefaultBranch()
	if managed {
		fmt.Printf("Adding shared file: %s (type: %s, managed)\n\n", source, addType)
		fmt.Printf("  Source: %s/%s -> .shared/%s (move)\n", defaultBranch, source, source)
	} else {
		fmt.Printf("Adding shared file: %s (type: %s)\n\n", source, addType)
		fmt.Printf("  Source: %s/%s\n", defaultBranch, source)
	}

	// Add shared file
	result, err := mgr.AddShared(source, addType, managed)
	if err != nil {
		var conflictErr *worktree.SharedConflictError
		if errors.As(err, &conflictErr) {
			fmt.Println("Error: conflicts detected, no changes made")
			fmt.Println()
			fmt.Printf("%s conflicts:\n", source)
			for _, c := range conflictErr.Conflicts {
				fmt.Printf("  x %s (file exists)\n", c.WorktreePath)
			}
			fmt.Println("To proceed, remove or rename conflicting files first.")
			return fmt.Errorf("conflicts detected")
		}
		return err
	}

	// Show results
	fmt.Println("  Apply:")
	if managed {
		// For managed, also show symlink to main worktree
		fmt.Printf("    + %s/%s -> ../.shared/%s (%s)\n", defaultBranch, source, source, addType)
	}
	for _, wt := range result.Applied {
		if managed {
			fmt.Printf("    + %s/%s -> ../.shared/%s (%s)\n", wt, source, source, addType)
		} else {
			if addType == "symlink" {
				fmt.Printf("    + %s/%s -> ../%s/%s (%s)\n", wt, source, defaultBranch, source, addType)
			} else {
				fmt.Printf("    + %s/%s (%s)\n", wt, source, addType)
			}
		}
	}
	for _, wt := range result.Skipped {
		fmt.Printf("    - %s/%s (already exists, skipped)\n", wt, source)
	}

	fmt.Println("+ Shared configuration added and applied.")

	return nil
}
