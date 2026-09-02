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

Safety:
  Without --force, bt aborts before touching anything when it detects
  uncommitted changes in the worktree, or (with --with-branch) a branch
  that is not fully merged into the default branch. Re-run with --force
  to delete anyway; --force enables both 'git worktree remove --force'
  and 'git branch -D'.

Examples:
  bt remove feature/auth
  bt rm feature/auth --with-branch
  bt rm feature/auth --with-branch --force`,
	Args:              cobra.ExactArgs(1),
	RunE:              runRemove,
	ValidArgsFunction: completeWorktreeNames(false),
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Force removal of uncommitted changes and unmerged branches")
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

	// Pre-flight safety checks: never touch worktree or branch until we know
	// none of the following will surface only after the worktree is already gone.
	//   - uncommitted changes in the worktree
	//   - branch not fully merged into the default branch (only when we would
	//     actually try to delete it, i.e. --with-branch)
	// --force bypasses both.
	if !removeForce {
		var blockers []string

		dirty, err := wtMgr.HasUncommittedChanges(worktreePath)
		if err != nil {
			return err
		}
		if dirty {
			blockers = append(blockers, "worktree has uncommitted changes")
		}

		if removeWithBranch && branchName != "" && branchName != "detached" {
			targetBranch := mgr.Config.Repository.DefaultBranch
			if targetBranch == "" {
				targetBranch = "main"
			}
			merged, err := mgr.Executor.IsBranchMerged(branchName, targetBranch)
			if err != nil {
				return err
			}
			if !merged {
				blockers = append(blockers, fmt.Sprintf("branch '%s' is not fully merged into '%s'", branchName, targetBranch))
			}
		}

		if len(blockers) > 0 {
			return fmt.Errorf("refusing to remove %s:\n  - %s\nRe-run with --force to delete anyway.",
				worktreePath, strings.Join(blockers, "\n  - "))
		}
	}

	fmt.Printf("Removing worktree at %s...\n", worktreePath)

	// Remove worktree
	if err := wtMgr.Remove(worktreePath, removeForce); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	fmt.Printf("✓ Worktree removed\n")

	// Branch deletion.
	// - With --with-branch (implied when --force is passed together with
	//   --with-branch, or when --with-branch alone was set): delete.
	// - Otherwise, prompt interactively. Skip the prompt when the branch is
	//   unmerged so we do not offer an option we would then reject.
	if branchName != "" && branchName != "detached" {
		deleteBranch := removeWithBranch

		if !removeWithBranch && !removeForce {
			targetBranch := mgr.Config.Repository.DefaultBranch
			if targetBranch == "" {
				targetBranch = "main"
			}
			merged, err := mgr.Executor.IsBranchMerged(branchName, targetBranch)
			if err != nil {
				fmt.Printf("Warning: failed to check merge status of '%s': %v\n", branchName, err)
			} else if !merged {
				fmt.Printf("\nBranch '%s' is not fully merged into '%s'. Skipping.\n", branchName, targetBranch)
				fmt.Printf("Re-run with --with-branch --force to delete anyway.\n")
			} else {
				fmt.Printf("\nDelete branch '%s'? [y/N]: ", branchName)
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))
				deleteBranch = response == "y" || response == "yes"
			}
		}

		if deleteBranch {
			if err := mgr.Executor.DeleteBranch(branchName, removeForce); err != nil {
				fmt.Printf("Warning: %v\n", err)
			} else {
				fmt.Printf("✓ Branch '%s' deleted\n", branchName)
			}
		}
	}

	return nil
}
