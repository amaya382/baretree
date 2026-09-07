package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// BranchInfo contains information about a branch
type BranchInfo struct {
	Name      string // Local branch name (e.g., "feature/auth")
	Remote    string // Remote name if tracking (e.g., "origin")
	RemoteRef string // Full remote ref (e.g., "origin/feature/auth")
	IsLocal   bool   // Whether it exists locally
	IsRemote  bool   // Whether it exists on remote
}

// ResolveBranch resolves a branch specification to BranchInfo
// It checks local branches first, then remote branches
func (e *Executor) ResolveBranch(branchSpec string) (*BranchInfo, error) {
	info := &BranchInfo{}

	// Check if it's explicitly a remote branch (e.g., "origin/feature/x" or "upstream/main")
	if strings.Contains(branchSpec, "/") {
		parts := strings.SplitN(branchSpec, "/", 2)
		potentialRemote := parts[0]

		// Check if the first part is a known remote
		if e.isRemote(potentialRemote) {
			info.Remote = potentialRemote
			info.Name = parts[1]
			info.RemoteRef = branchSpec
			info.IsRemote = true

			// Also check if local branch exists
			info.IsLocal = e.localBranchExists(info.Name)
			return info, nil
		}
	}

	// Not explicitly remote, treat as branch name
	info.Name = branchSpec

	// Check if local branch exists
	if e.localBranchExists(branchSpec) {
		info.IsLocal = true
		return info, nil
	}

	// Check origin/<branch>
	if e.remoteBranchExists("origin", branchSpec) {
		info.Remote = "origin"
		info.RemoteRef = "origin/" + branchSpec
		info.IsRemote = true
		return info, nil
	}

	// Check other remotes
	remotes, err := e.ListRemotes()
	if err == nil {
		for _, remote := range remotes {
			if remote == "origin" {
				continue // Already checked
			}
			if e.remoteBranchExists(remote, branchSpec) {
				info.Remote = remote
				info.RemoteRef = remote + "/" + branchSpec
				info.IsRemote = true
				return info, nil
			}
		}
	}

	// Branch not found anywhere
	return info, nil
}

// ListRemotes returns a list of configured remotes
func (e *Executor) ListRemotes() ([]string, error) {
	output, err := e.Execute("remote")
	if err != nil {
		return nil, err
	}

	if output == "" {
		return []string{}, nil
	}

	return strings.Split(output, "\n"), nil
}

// Fetch fetches from the specified remote (or all if empty)
func (e *Executor) Fetch(remote string) error {
	args := []string{"fetch"}
	if remote != "" {
		args = append(args, remote)
	} else {
		args = append(args, "--all")
	}

	_, err := e.Execute(args...)
	return err
}

// isRemote checks if the given name is a configured remote
func (e *Executor) isRemote(name string) bool {
	remotes, err := e.ListRemotes()
	if err != nil {
		return false
	}

	for _, remote := range remotes {
		if remote == name {
			return true
		}
	}
	return false
}

// localBranchExists checks if a local branch exists
func (e *Executor) localBranchExists(branch string) bool {
	_, err := e.Execute("show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}

// remoteBranchExists checks if a remote tracking branch exists
func (e *Executor) remoteBranchExists(remote, branch string) bool {
	_, err := e.Execute("show-ref", "--verify", "--quiet", "refs/remotes/"+remote+"/"+branch)
	return err == nil
}

// ResolveHEAD returns the branch name that HEAD points to (e.g., "main").
// Returns empty string if HEAD is detached or on error.
func (e *Executor) ResolveHEAD() string {
	output, err := e.Execute("symbolic-ref", "--short", "HEAD")
	if err != nil {
		return ""
	}
	return output
}

// HasRemotes returns true if any remotes are configured
func (e *Executor) HasRemotes() bool {
	remotes, err := e.ListRemotes()
	if err != nil {
		return false
	}
	return len(remotes) > 0
}

// GetUpstreamBehindCount returns how many commits a local branch is behind its upstream.
// Returns 0 if no upstream is configured or on any error.
func (e *Executor) GetUpstreamBehindCount(localBranch string) (int, error) {
	// Check if the branch has an upstream configured
	_, err := e.Execute("config", "--get", "branch."+localBranch+".remote")
	if err != nil {
		// No upstream configured
		return 0, nil
	}

	output, err := e.Execute("rev-list", "--count", localBranch+".."+localBranch+"@{u}")
	if err != nil {
		return 0, nil
	}

	count, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		return 0, nil
	}
	return count, nil
}

// IsCommitHash checks if the given string resolves to a valid commit object
func (e *Executor) IsCommitHash(ref string) bool {
	_, err := e.Execute("rev-parse", "--verify", "--quiet", ref+"^{commit}")
	return err == nil
}

// ListLocalBranches returns a list of all local branch names
func (e *Executor) ListLocalBranches() ([]string, error) {
	output, err := e.Execute("for-each-ref", "--format=%(refname:short)", "refs/heads/")
	if err != nil {
		return nil, fmt.Errorf("failed to list local branches: %w", err)
	}

	if output == "" {
		return []string{}, nil
	}

	return strings.Split(output, "\n"), nil
}

// IsBranchMerged reports whether branch is fully merged into target.
// A branch is considered merged when it is an ancestor of target, i.e. deleting
// it with `git branch -d` would not lose any commits reachable only from it.
func (e *Executor) IsBranchMerged(branch, target string) (bool, error) {
	if branch == "" {
		return false, fmt.Errorf("branch name is empty")
	}
	if target == "" {
		return false, fmt.Errorf("target branch is empty")
	}

	// merge-base --is-ancestor exits 0 when branch is ancestor of target, 1 when not.
	// Anything else is a real failure.
	_, _, err := e.ExecuteWithStderr("merge-base", "--is-ancestor", branch, target)
	if err == nil {
		return true, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("failed to check merge status of '%s' against '%s': %w", branch, target, err)
}

// DeleteBranch deletes a local branch. When force is true, uses `-D` to bypass
// the not-fully-merged safety check.
func (e *Executor) DeleteBranch(branch string, force bool) error {
	if branch == "" {
		return fmt.Errorf("branch name is empty")
	}

	flag := "-d"
	if force {
		flag = "-D"
	}

	if _, err := e.Execute("branch", flag, branch); err != nil {
		return fmt.Errorf("failed to delete branch '%s': %w", branch, err)
	}
	return nil
}

// PullBranch fast-forwards the specified local branch to its upstream.
// If the branch is checked out in a worktree, runs `git merge --ff-only` there
// so the working tree and index advance with the ref. Otherwise updates the ref
// directly via update-ref (bare-safe path).
func (e *Executor) PullBranch(localBranch string) error {
	upstreamHash, err := e.Execute("rev-parse", localBranch+"@{u}")
	if err != nil {
		return fmt.Errorf("failed to resolve upstream for '%s': %w", localBranch, err)
	}

	localHash, err := e.Execute("rev-parse", localBranch)
	if err != nil {
		return fmt.Errorf("failed to resolve '%s': %w", localBranch, err)
	}

	if localHash == upstreamHash {
		return nil
	}

	if _, err := e.Execute("merge-base", "--is-ancestor", localBranch, localBranch+"@{u}"); err != nil {
		return fmt.Errorf("cannot fast-forward '%s': upstream has diverged", localBranch)
	}

	worktreePath, err := e.checkedOutWorktreePath(localBranch)
	if err != nil {
		return fmt.Errorf("failed to locate worktree for '%s': %w", localBranch, err)
	}

	if worktreePath != "" {
		cmd := exec.Command("git", "merge", "--ff-only", localBranch+"@{u}")
		cmd.Dir = worktreePath
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to fast-forward '%s' in worktree %s: %w\n%s",
				localBranch, worktreePath, err, strings.TrimSpace(string(out)))
		}
		return nil
	}

	_, err = e.Execute("update-ref", "refs/heads/"+localBranch, upstreamHash)
	return err
}

// checkedOutWorktreePath returns the worktree path where localBranch is checked out,
// or "" if no worktree currently has it checked out.
func (e *Executor) checkedOutWorktreePath(localBranch string) (string, error) {
	output, err := e.Execute("worktree", "list", "--porcelain")
	if err != nil {
		return "", err
	}
	for _, wt := range ParseWorktreeList(output) {
		if wt.Branch == localBranch {
			return wt.Path, nil
		}
	}
	return "", nil
}
