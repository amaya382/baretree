package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	addNewBranch  bool
	addBaseBranch string
	addDetach     bool
	addForce      bool
	addNoFetch    bool
	addBehind     string
)

var addCmd = &cobra.Command{
	Use:   "add <branch-name>",
	Short: "Create a worktree for a branch (creates branch with -b)",
	Long: `Create a new worktree for a branch.

When remotes are configured, fetches from all remotes before adding the worktree
(use --no-fetch to skip).

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

When the base branch is behind its upstream, an interactive prompt lets you choose:
  1. Continue without pulling (use as-is)
  2. Pull (fast-forward) and then continue
  3. Abort
Use --behind=continue|pull|abort to skip the prompt.

Examples:
  bt add -b feature/auth           # Creates new branch and worktree
  bt add -b feature/new --base abc123  # Creates new branch based on a commit
  bt add existing-local-branch     # Uses existing local branch
  bt add feature/remote            # Auto-detects and tracks origin/feature/remote
  bt add upstream/feature/test     # Tracks upstream/feature/test
  bt add --no-fetch feature/new    # Skip auto-fetch from remotes
  bt add -b feature/new --behind=pull  # Pull base branch if behind, then create`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().BoolVarP(&addNewBranch, "branch", "b", false, "Create new branch")
	addCmd.Flags().StringVar(&addBaseBranch, "base", "", "Base branch or commit hash for new branch (default: HEAD)")
	addCmd.Flags().BoolVar(&addDetach, "detach", false, "Create detached HEAD worktree")
	addCmd.Flags().BoolVar(&addForce, "force", false, "Force creation even if worktree exists")
	addCmd.Flags().BoolVar(&addNoFetch, "no-fetch", false, "Skip auto-fetch from remotes")
	addCmd.Flags().StringVar(&addBehind, "behind", "", "Action when base branch is behind upstream: continue, pull, abort")
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

	// Auto-fetch unless --no-fetch is specified or no remotes configured
	if !addNoFetch && wtMgr.Executor.HasRemotes() {
		fmt.Println("Fetching from remotes...")
		if err := wtMgr.Fetch(""); err != nil {
			return fmt.Errorf("failed to fetch: %w", err)
		}
	}

	// Resolve base branch if specified
	var resolvedBaseBranch string
	var baseDisplayInfo string
	var baseIsLocal bool
	if addBaseBranch != "" {
		baseInfo, err := wtMgr.ResolveBranch(addBaseBranch)
		if err != nil {
			return fmt.Errorf("failed to resolve base branch '%s': %w", addBaseBranch, err)
		}

		if baseInfo.IsLocal {
			resolvedBaseBranch = baseInfo.Name
			baseDisplayInfo = baseInfo.Name + " (local)"
			baseIsLocal = true
		} else if baseInfo.IsRemote {
			resolvedBaseBranch = baseInfo.RemoteRef
			baseDisplayInfo = baseInfo.RemoteRef + " (remote)"
		} else if wtMgr.Executor.IsCommitHash(addBaseBranch) {
			resolvedBaseBranch = addBaseBranch
			shortHash := addBaseBranch
			if len(shortHash) > 7 {
				shortHash = shortHash[:7]
			}
			baseDisplayInfo = shortHash + " (commit)"
		} else {
			return fmt.Errorf("base branch '%s' not found locally or on any remote", addBaseBranch)
		}
	}

	// Build add options
	opts := worktree.AddOptions{
		NewBranch:  addNewBranch,
		BaseBranch: resolvedBaseBranch,
	}

	var branchName string
	var resolvedBranchIsLocal bool

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
			resolvedBranchIsLocal = true
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

	// Validate --behind flag value
	if addBehind != "" && addBehind != "continue" && addBehind != "pull" && addBehind != "abort" {
		return fmt.Errorf("invalid value for --behind: %q (must be 'continue', 'pull', or 'abort')", addBehind)
	}

	// Upstream behind detection
	if !addForce {
		var branchToCheck string

		if addNewBranch {
			if addBaseBranch != "" && baseIsLocal {
				// --base was specified and resolved to a local branch
				branchToCheck = resolvedBaseBranch
			} else if addBaseBranch == "" {
				// No --base: check the default branch
				branchToCheck = mgr.Config.Repository.DefaultBranch
				if branchToCheck == "" {
					branchToCheck = "main"
				}
			}
		} else if resolvedBranchIsLocal {
			// Not -b mode: local branch resolved, check it
			branchToCheck = branchName
		}

		if branchToCheck != "" {
			behindCount, _ := wtMgr.Executor.GetUpstreamBehindCount(branchToCheck)
			if behindCount > 0 {
				fmt.Printf("Warning: '%s' is %d commit(s) behind its upstream.\n", branchToCheck, behindCount)

				action := addBehind
				if action == "" {
					if isTerminal() {
						// Interactive: show 3-choice prompt
						fmt.Println("Choose action:")
						fmt.Println("  [1] Continue without pulling (use as-is)")
						fmt.Println("  [2] Pull (fast-forward) and then continue")
						fmt.Println("  [3] Abort")
						fmt.Printf("Select [1/2/3]: ")
						reader := bufio.NewReader(os.Stdin)
						response, _ := reader.ReadString('\n')
						response = strings.TrimSpace(response)
						switch response {
						case "1":
							action = "continue"
						case "2":
							action = "pull"
						default:
							action = "abort"
						}
					} else {
						// Non-TTY: warn but proceed
						action = "continue"
					}
				}

				switch action {
				case "continue":
					// Proceed as-is
				case "pull":
					fmt.Printf("Pulling '%s'...\n", branchToCheck)
					if err := wtMgr.Executor.PullBranch(branchToCheck); err != nil {
						return fmt.Errorf("failed to pull '%s': %w", branchToCheck, err)
					}
					fmt.Printf("'%s' is now up to date.\n", branchToCheck)
				case "abort":
					return fmt.Errorf("aborted: '%s' is behind upstream", branchToCheck)
				}
			}
		}
	}

	// Display base information
	if addNewBranch {
		if baseDisplayInfo != "" {
			fmt.Printf("Based on '%s'\n", baseDisplayInfo)
		} else {
			headBranch := wtMgr.Executor.ResolveHEAD()
			if headBranch != "" {
				fmt.Printf("Based on HEAD (%s)\n", headBranch)
			} else {
				fmt.Println("Based on HEAD")
			}
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

// isTerminal checks if stdin is connected to a terminal
func isTerminal() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
