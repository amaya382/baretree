package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/amaya382/baretree/internal/git"
	"github.com/amaya382/baretree/internal/repository"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename [old-name] <new-name>",
	Short: "Rename a worktree (renames both the directory and branch together)",
	Long: `Rename a worktree, including both its directory and branch.

If only one argument is provided, renames the current worktree.
If two arguments are provided, renames the specified worktree.

This command renames:
  - The Git branch
  - The worktree directory
  - Updates the worktree registration

Note: The worktree name, branch name, and directory name must be consistent.
If they are not, use 'bt repair' to fix the inconsistency first.

Examples:
  bt rename feature/new-name           # Rename current worktree
  bt rename feature/old feature/new    # Rename specified worktree`,
	Args:              cobra.RangeArgs(1, 2),
	RunE:              runRename,
	ValidArgsFunction: completeWorktreeForRename,
}

// Note: renameCmd is registered in main.go

func runRename(cmd *cobra.Command, args []string) error {
	// Find repository root
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("not in a baretree repository: %w", err)
	}

	// Get bare repository path
	bareDir, err := repository.GetBareRepoPath(repoRoot)
	if err != nil {
		return err
	}

	executor := git.NewExecutor(bareDir)

	// Determine old and new names
	var oldName, newName string
	if len(args) == 1 {
		// Rename current worktree
		newName = args[0]

		// Detect current worktree
		oldName, err = detectCurrentWorktree(cwd, repoRoot)
		if err != nil {
			return fmt.Errorf("failed to detect current worktree: %w", err)
		}
	} else {
		oldName = args[0]
		newName = args[1]
	}

	// Validate names
	if oldName == newName {
		return fmt.Errorf("old and new names are the same")
	}

	// Get worktree info
	output, err := executor.Execute("worktree", "list", "--porcelain")
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	worktrees := git.ParseWorktreeList(output)

	// Find the worktree to rename
	var targetWorktree *git.Worktree
	oldWorktreePath := filepath.Join(repoRoot, oldName)

	for i := range worktrees {
		if worktrees[i].Path == oldWorktreePath {
			targetWorktree = &worktrees[i]
			break
		}
	}

	if targetWorktree == nil {
		return fmt.Errorf("worktree not found: %s", oldName)
	}

	if targetWorktree.IsBare {
		return fmt.Errorf("cannot rename bare repository")
	}

	// Check consistency: worktree path should match branch name
	expectedBranch := oldName
	if targetWorktree.Branch != expectedBranch && targetWorktree.Branch != "" {
		return fmt.Errorf("worktree name (%s) and branch name (%s) are inconsistent\nUse 'bt repair' to fix this inconsistency first", oldName, targetWorktree.Branch)
	}

	// Check if new worktree path already exists
	newWorktreePath := filepath.Join(repoRoot, newName)
	if _, err := os.Stat(newWorktreePath); err == nil {
		return fmt.Errorf("destination already exists: %s", newWorktreePath)
	}

	// Check if new branch name already exists
	if _, err := executor.Execute("show-ref", "--verify", "--quiet", "refs/heads/"+newName); err == nil {
		return fmt.Errorf("branch already exists: %s", newName)
	}

	fmt.Printf("Renaming worktree '%s' to '%s'...\n", oldName, newName)

	// Step 1: Rename the branch
	fmt.Printf("  Renaming branch...\n")
	if _, err := executor.Execute("branch", "-m", oldName, newName); err != nil {
		return fmt.Errorf("failed to rename branch: %w", err)
	}

	// Step 2: Move the worktree directory
	fmt.Printf("  Moving directory...\n")

	// Create parent directory if needed
	newParentDir := filepath.Dir(newWorktreePath)
	if err := os.MkdirAll(newParentDir, 0755); err != nil {
		// Rollback branch rename
		if _, rollbackErr := executor.Execute("branch", "-m", newName, oldName); rollbackErr != nil {
			return fmt.Errorf("failed to create parent directory and failed to roll back: %w / %w", err, rollbackErr)
		}
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	if err := os.Rename(oldWorktreePath, newWorktreePath); err != nil {
		// Rollback branch rename
		if _, rollbackErr := executor.Execute("branch", "-m", newName, oldName); rollbackErr != nil {
			return fmt.Errorf("failed to move worktree directory and also failed to roll back: %w / %w", err, rollbackErr)
		}
		return fmt.Errorf("failed to move worktree directory: %w", err)
	}

	// Step 3: Update worktree registration using git worktree move
	fmt.Printf("  Updating worktree registration...\n")
	if _, err := executor.Execute("worktree", "repair", newWorktreePath); err != nil {
		// Try to rollback
		if renameErr := os.Rename(newWorktreePath, oldWorktreePath); renameErr != nil {
			return fmt.Errorf("failed to update worktree registration and failed to roll back: %w / %w", err, renameErr)
		}
		if _, executeErr := executor.Execute("branch", "-m", newName, oldName); executeErr != nil {
			return fmt.Errorf("failed to update worktree registration and also failed to roll back: %w / %w", err, executeErr)
		}
		return fmt.Errorf("failed to update worktree registration: %w", err)
	}

	// Clean up empty parent directories of old path
	err = cleanupEmptyDirs(filepath.Dir(oldWorktreePath), repoRoot)

	fmt.Printf("\n✓ Successfully renamed worktree\n")
	fmt.Printf("  Old: %s\n", oldName)
	fmt.Printf("  New: %s\n", newName)
	fmt.Printf("  Path: %s\n", newWorktreePath)

	return err
}

// Note: detectCurrentWorktree and cleanupEmptyDirs are defined in repair.go
