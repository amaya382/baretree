package postcreate

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	addNoManaged bool
)

var addCmd = &cobra.Command{
	Use:   "add <type> <source>",
	Short: "Add a post-create action",
	Long: `Add an action to be performed after worktree creation.

Types:
  - symlink: Create a symlink to a shared file
  - copy: Copy a file to the new worktree
  - command: Execute a shell command in the new worktree

For symlink/copy types:
  - The source file must exist in the default branch worktree (usually main).
  - Managed (default): File is moved to .shared/ directory, independent of any worktree
  - Non-managed (--no-managed): File is sourced from the default branch worktree

For command type:
  - The source is the command string to execute
  - Commands are executed via 'sh -c' in the new worktree directory
  - Command failures are treated as warnings (worktree creation continues)

Examples:
  bt post-create add symlink .env
  bt post-create add symlink .env --no-managed
  bt post-create add copy config/local.json
  bt post-create add command "direnv allow"
  bt post-create add command "npm install"`,
	Args: cobra.ExactArgs(2),
	RunE: runPostCreateAdd,
}

func init() {
	addCmd.Flags().BoolVar(&addNoManaged, "no-managed", false, "Source file from the default branch worktree instead of .shared/ directory (symlink/copy only)")
}

func runPostCreateAdd(cmd *cobra.Command, args []string) error {
	actionType := args[0]
	source := args[1]

	// Validate type
	if actionType != "symlink" && actionType != "copy" && actionType != "command" {
		return fmt.Errorf("invalid type: %s (must be 'symlink', 'copy', or 'command')", actionType)
	}

	// Clean source for file types
	if actionType != "command" {
		source = filepath.Clean(source)
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

	// For command type, managed flag is ignored
	managed := !addNoManaged
	if actionType == "command" {
		managed = false
	}

	// Show what will happen
	defaultBranch := mgr.GetDefaultBranch()
	switch actionType {
	case "command":
		fmt.Printf("Adding post-create command: %s\n\n", source)
		fmt.Printf("  This command will be executed in new worktrees after creation.\n")
	case "symlink", "copy":
		if managed {
			fmt.Printf("Adding post-create action: %s (type: %s, managed)\n\n", source, actionType)
			fmt.Printf("  Source: %s/%s -> .shared/%s (move)\n", defaultBranch, source, source)
		} else {
			fmt.Printf("Adding post-create action: %s (type: %s)\n\n", source, actionType)
			fmt.Printf("  Source: %s/%s\n", defaultBranch, source)
		}
	}

	// Add post-create action
	result, err := mgr.AddPostCreate(source, actionType, managed)
	if err != nil {
		var conflictErr *worktree.PostCreateConflictError
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
	if actionType == "command" {
		fmt.Println("+ Post-create command added.")
		fmt.Println()
		fmt.Println("Note: Commands are only executed for newly created worktrees.")
	} else {
		fmt.Println("  Apply:")
		if managed {
			// For managed, also show symlink to main worktree
			fmt.Printf("    + %s/%s -> ../.shared/%s (%s)\n", defaultBranch, source, source, actionType)
		}
		for _, wt := range result.Applied {
			if managed {
				fmt.Printf("    + %s/%s -> ../.shared/%s (%s)\n", wt, source, source, actionType)
			} else {
				if actionType == "symlink" {
					fmt.Printf("    + %s/%s -> ../%s/%s (%s)\n", wt, source, defaultBranch, source, actionType)
				} else {
					fmt.Printf("    + %s/%s (%s)\n", wt, source, actionType)
				}
			}
		}
		for _, wt := range result.Skipped {
			fmt.Printf("    - %s/%s (already exists, skipped)\n", wt, source)
		}

		fmt.Println("+ Post-create action added and applied.")
	}

	return nil
}
