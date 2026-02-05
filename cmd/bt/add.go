package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	addNewBranch  bool
	addBaseBranch string
	addDetach     bool
	addForce      bool
	addFetch      bool
)

var addCmd = &cobra.Command{
	Use:   "add <branch-name>",
	Short: "Create a worktree for a branch (creates branch with -b)",
	Long: `Create a new worktree for a branch.

Supports multiple modes:
  1. Create new branch:     bt add -b feature/new
  2. Existing local branch: bt add existing-branch
  3. Remote branch:         bt add feature/remote (auto-detects origin/feature/remote)
  4. Explicit remote:       bt add upstream/feature/foo

The worktree path is automatically determined from the branch name.
Branch names with slashes create hierarchical directories.

Branch resolution order:
  1. Local branch exists -> use it
  2. origin/<branch> exists -> create tracking branch
  3. <remote>/<branch> format -> use specified remote

Examples:
  bt add -b feature/auth           # Creates new branch and worktree
  bt add existing-local-branch     # Uses existing local branch
  bt add feature/remote            # Auto-detects and tracks origin/feature/remote
  bt add upstream/feature/test     # Tracks upstream/feature/test
  bt add --fetch feature/new       # Fetch before adding`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().BoolVarP(&addNewBranch, "branch", "b", false, "Create new branch")
	addCmd.Flags().StringVar(&addBaseBranch, "base", "", "Base branch for new branch (default: HEAD)")
	addCmd.Flags().BoolVar(&addDetach, "detach", false, "Create detached HEAD worktree")
	addCmd.Flags().BoolVar(&addForce, "force", false, "Force creation even if worktree exists")
	addCmd.Flags().BoolVar(&addFetch, "fetch", false, "Fetch from remote before adding worktree")
}

func runAdd(cmd *cobra.Command, args []string) error {
	branchSpec := args[0]

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

	// Load config
	mgr, err := repository.NewManager(repoRoot)
	if err != nil {
		return err
	}

	// Create worktree manager
	wtMgr := worktree.NewManager(repoRoot, bareDir, mgr.Config)

	// Fetch if requested
	if addFetch {
		fmt.Println("Fetching from remotes...")
		if err := wtMgr.Fetch(""); err != nil {
			return fmt.Errorf("failed to fetch: %w", err)
		}
	}

	// Build add options
	opts := worktree.AddOptions{
		NewBranch:  addNewBranch,
		BaseBranch: addBaseBranch,
	}

	var branchName string

	if addNewBranch {
		// Creating a new branch - use spec as-is
		branchName = branchSpec
	} else {
		// Resolve the branch specification
		branchInfo, err := wtMgr.ResolveBranch(branchSpec)
		if err != nil {
			return fmt.Errorf("failed to resolve branch: %w", err)
		}

		if branchInfo.IsLocal {
			// Local branch exists
			branchName = branchInfo.Name
		} else if branchInfo.IsRemote {
			// Remote branch found - create tracking branch
			branchName = branchInfo.Name
			opts.TrackRef = branchInfo.RemoteRef
			fmt.Printf("Tracking remote branch '%s'...\n", branchInfo.RemoteRef)
		} else {
			// Branch not found anywhere
			return fmt.Errorf("branch '%s' not found locally or on any remote\nUse 'bt add -b %s' to create a new branch", branchSpec, branchSpec)
		}
	}

	fmt.Printf("Creating worktree for branch '%s'...\n", branchName)

	// Add worktree (pass os.Stdout for real-time output including "Worktree created" message)
	_, postCreateResult, err := wtMgr.AddWithOptions(branchName, opts, os.Stdout)
	if err != nil {
		var existsErr *worktree.ErrWorktreeAlreadyExists
		if errors.As(err, &existsErr) {
			fmt.Printf("Worktree for branch '%s' already exists at:\n", existsErr.BranchName)
			fmt.Printf("  %s\n\n", existsErr.WorktreePath)
			fmt.Printf("To switch to this worktree, use:\n")
			fmt.Printf("  bt cd %s\n", existsErr.BranchName)
			return nil
		}
		var refConflictErr *worktree.ErrRefConflict
		if errors.As(err, &refConflictErr) {
			return refConflictErr
		}
		return fmt.Errorf("failed to add worktree: %w", err)
	}

	// "Worktree created" message and post-create output are already printed by AddWithOptions
	// Just check if any commands failed and show warning
	if postCreateResult != nil && len(postCreateResult.CommandResults) > 0 {
		hasErrors := false
		for _, result := range postCreateResult.CommandResults {
			if !result.Success {
				hasErrors = true
				break
			}
		}

		if hasErrors {
			fmt.Println("\nWarning: Some post-create commands failed")
		}
	}

	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  bt cd %s\n", branchName)
	fmt.Printf("  # Start working on %s\n", branchName)

	return nil
}
