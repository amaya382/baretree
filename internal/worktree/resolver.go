package worktree

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/amaya382/baretree/internal/git"
)

// AmbiguousMatchError is returned when multiple worktrees match a given name
type AmbiguousMatchError struct {
	Name     string
	Matches  []git.Worktree
	RepoRoot string
}

func (e *AmbiguousMatchError) Error() string {
	return fmt.Sprintf("ambiguous worktree name '%s': %d matches found", e.Name, len(e.Matches))
}

// Resolve resolves a worktree name to its path
func (m *Manager) Resolve(name string) (string, error) {
	return m.ResolveFromCwd(name, "")
}

// ResolveFromCwd resolves a worktree name to its path, with cwd used for empty name resolution
func (m *Manager) ResolveFromCwd(name string, cwd string) (string, error) {
	// Get all worktrees
	worktrees, err := m.List()
	if err != nil {
		return "", err
	}

	// Special case: empty string means current worktree root
	if name == "" {
		if cwd == "" {
			// Fallback to default worktree if no cwd provided
			for _, wt := range worktrees {
				if wt.IsMain {
					return wt.Path, nil
				}
			}
			return "", fmt.Errorf("default worktree not found")
		}
		// Find which worktree contains cwd
		for _, wt := range worktrees {
			if strings.HasPrefix(cwd, wt.Path+string(filepath.Separator)) || cwd == wt.Path {
				return wt.Path, nil
			}
		}
		return "", fmt.Errorf("not in a worktree")
	}

	// Special case: @ means default worktree
	if name == "@" {
		for _, wt := range worktrees {
			if wt.IsMain {
				return wt.Path, nil
			}
		}
		return "", fmt.Errorf("default worktree not found")
	}

	// Try exact branch name match
	for _, wt := range worktrees {
		if wt.Branch == name {
			return wt.Path, nil
		}
	}

	// Try path-based match (relative to repo root)
	candidatePath := filepath.Join(m.RepoRoot, name)
	for _, wt := range worktrees {
		if wt.Path == candidatePath {
			return wt.Path, nil
		}
	}

	// Try directory name match - collect all matches
	var matches []git.Worktree
	for _, wt := range worktrees {
		if filepath.Base(wt.Path) == name {
			matches = append(matches, wt)
		}
	}

	if len(matches) == 1 {
		return matches[0].Path, nil
	}

	if len(matches) > 1 {
		return "", &AmbiguousMatchError{
			Name:     name,
			Matches:  matches,
			RepoRoot: m.RepoRoot,
		}
	}

	return "", fmt.Errorf("worktree not found: %s", name)
}

// GetBranchName returns the branch name from a worktree path
func (m *Manager) GetBranchName(worktreePath string) (string, error) {
	executor := git.NewExecutor(worktreePath)

	// Get current branch
	branch, err := executor.Execute("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get branch name: %w", err)
	}

	return strings.TrimSpace(branch), nil
}
