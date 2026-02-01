package git

import (
	"strings"
)

// Worktree represents a git worktree
type Worktree struct {
	Path   string
	Head   string
	Branch string
	IsMain bool
	IsBare bool
}

// ParseWorktreeList parses the output of "git worktree list --porcelain"
func ParseWorktreeList(output string) []Worktree {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var worktrees []Worktree
	var current Worktree
	isFirst := true

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			// Empty line marks end of worktree entry
			if current.Path != "" {
				if isFirst {
					current.IsMain = true
					isFirst = false
				}
				worktrees = append(worktrees, current)
				current = Worktree{}
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			current.Path = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "HEAD ") {
			current.Head = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") {
			branch := strings.TrimPrefix(line, "branch ")
			// Remove refs/heads/ prefix
			current.Branch = strings.TrimPrefix(branch, "refs/heads/")
		} else if line == "detached" {
			current.Branch = "detached"
		} else if line == "bare" {
			current.IsBare = true
		}
	}

	// Add last worktree if exists
	if current.Path != "" {
		if isFirst {
			current.IsMain = true
		}
		worktrees = append(worktrees, current)
	}

	return worktrees
}

// GetDefaultBranch detects the default branch from a bare repository
func GetDefaultBranch(bareRepoPath string) (string, error) {
	executor := NewExecutor(bareRepoPath)

	// Get the symbolic ref for origin/HEAD
	output, err := executor.Execute("symbolic-ref", "refs/remotes/origin/HEAD")
	if err != nil {
		// Fallback to common branch names
		for _, branch := range []string{"main", "master"} {
			_, err := executor.Execute("show-ref", "--verify", "--quiet", "refs/heads/"+branch)
			if err == nil {
				return branch, nil
			}
		}
		return "", err
	}

	// Parse refs/remotes/origin/main -> main
	branch := strings.TrimPrefix(output, "refs/remotes/origin/")
	return branch, nil
}

// ToWorktreeGitDirName converts a branch name to a safe directory name for .git/worktrees/.
// Git expects flat directory names under .git/worktrees/, so slashes in branch names
// must be escaped. We use URL-style encoding (%2F) to avoid conflicts with branch names
// that contain dashes.
func ToWorktreeGitDirName(branchName string) string {
	// Escape existing % first to avoid double-encoding issues
	escaped := strings.ReplaceAll(branchName, "%", "%25")
	// Then escape slashes
	escaped = strings.ReplaceAll(escaped, "/", "%2F")
	return escaped
}
