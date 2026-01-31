package git

import (
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
