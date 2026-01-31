package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	removeForce      bool
	removeWithBranch bool
)

var removeCmd = &cobra.Command{
	Use:     "remove <worktree-name>",
	Aliases: []string{"rm"},
	Short:   "Remove a worktree directory (optionally delete branch with -b)",
	Long: `Remove a worktree directory and optionally delete its branch.

The worktree name can be:
  - Branch name (e.g., feature/auth)
  - Directory name (e.g., feature/auth)
  - Path to worktree

Examples:
  bt remove feature/auth
  bt rm feature/auth --with-branch
  bt rm feature/auth --force`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Force removal even with uncommitted changes")
	removeCmd.Flags().BoolVarP(&removeWithBranch, "with-branch", "b", false, "Also delete the branch")
}

func runRemove(cmd *cobra.Command, args []string) error {
	worktreeName := args[0]

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

	// Load config and create manager
	mgr, err := repository.NewManager(repoRoot)
	if err != nil {
		return err
	}

	wtMgr := worktree.NewManager(repoRoot, bareDir, mgr.Config)

	// Resolve worktree name to path
	worktreePath, err := wtMgr.Resolve(worktreeName)
	if err != nil {
		return fmt.Errorf("failed to resolve worktree: %w", err)
	}

	// Get branch name before removal
	branchName, _ := wtMgr.GetBranchName(worktreePath)

	// Check if we're currently in the worktree
	if strings.HasPrefix(cwd, worktreePath) {
		return fmt.Errorf("cannot remove worktree while inside it")
	}

	fmt.Printf("Removing worktree at %s...\n", worktreePath)

	// Remove worktree
	if err := wtMgr.Remove(worktreePath, removeForce); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	fmt.Printf("✓ Worktree removed\n")

	// Ask about branch deletion if not forced
	if branchName != "" && branchName != "detached" {
		deleteBranch := removeWithBranch

		if !removeWithBranch && !removeForce {
			fmt.Printf("\nDelete branch '%s'? [y/N]: ", branchName)
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			deleteBranch = response == "y" || response == "yes"
		}

		if deleteBranch {
			forceFlag := ""
			if removeForce {
				forceFlag = "-D"
			} else {
				forceFlag = "-d"
			}

			if _, err := mgr.Executor.Execute("branch", forceFlag, branchName); err != nil {
				fmt.Printf("Warning: failed to delete branch '%s': %v\n", branchName, err)
			} else {
				fmt.Printf("✓ Branch '%s' deleted\n", branchName)
			}
		}
	}

	return nil
}
