package worktree

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/amaya382/baretree/internal/config"
	"github.com/amaya382/baretree/internal/git"
)

// MockManager for testing resolver logic without git
type mockResolver struct {
	repoRoot  string
	worktrees []git.Worktree
}

func (m *mockResolver) resolve(name string, cwd string) (string, error) {
	// Replicate the resolution logic from Manager.ResolveFromCwd

	// Special case: empty string means current worktree root
	if name == "" {
		if cwd == "" {
			// Fallback to default worktree if no cwd provided
			for _, wt := range m.worktrees {
				if wt.IsMain {
					return wt.Path, nil
				}
			}
			return "", nil
		}
		// Find which worktree contains cwd
		for _, wt := range m.worktrees {
			if strings.HasPrefix(cwd, wt.Path+string(filepath.Separator)) || cwd == wt.Path {
				return wt.Path, nil
			}
		}
		// Not in a worktree (e.g., at repo root) - fall back to default worktree
		for _, wt := range m.worktrees {
			if wt.IsMain {
				return wt.Path, nil
			}
		}
		return "", nil
	}

	// Special case: @ means default worktree
	if name == "@" {
		for _, wt := range m.worktrees {
			if wt.IsMain {
				return wt.Path, nil
			}
		}
		return "", nil
	}

	// Try exact branch name match
	for _, wt := range m.worktrees {
		if wt.Branch == name {
			return wt.Path, nil
		}
	}

	// Try path-based match
	candidatePath := m.repoRoot + "/" + name
	for _, wt := range m.worktrees {
		if wt.Path == candidatePath {
			return wt.Path, nil
		}
	}

	return "", nil
}

func TestResolverLogic(t *testing.T) {
	resolver := &mockResolver{
		repoRoot: "/home/user/project",
		worktrees: []git.Worktree{
			{Path: "/home/user/project/main", Branch: "main", IsMain: true},
			{Path: "/home/user/project/feature/auth", Branch: "feature/auth", IsMain: false},
			{Path: "/home/user/project/bugfix/cors", Branch: "bugfix/cors", IsMain: false},
		},
	}

	tests := []struct {
		name     string
		input    string
		cwd      string
		expected string
	}{
		{
			name:     "empty string with cwd in main returns main root",
			input:    "",
			cwd:      "/home/user/project/main/src",
			expected: "/home/user/project/main",
		},
		{
			name:     "empty string with cwd in feature returns feature root",
			input:    "",
			cwd:      "/home/user/project/feature/auth/src",
			expected: "/home/user/project/feature/auth",
		},
		{
			name:     "empty string with no cwd returns default",
			input:    "",
			cwd:      "",
			expected: "/home/user/project/main",
		},
		{
			name:     "empty string with cwd at repo root returns default worktree",
			input:    "",
			cwd:      "/home/user/project",
			expected: "/home/user/project/main",
		},
		{
			name:     "@ returns default worktree",
			input:    "@",
			cwd:      "/home/user/project/feature/auth",
			expected: "/home/user/project/main",
		},
		{
			name:     "exact branch match",
			input:    "feature/auth",
			cwd:      "",
			expected: "/home/user/project/feature/auth",
		},
		{
			name:     "main branch",
			input:    "main",
			cwd:      "",
			expected: "/home/user/project/main",
		},
		{
			name:     "bugfix branch",
			input:    "bugfix/cors",
			cwd:      "",
			expected: "/home/user/project/bugfix/cors",
		},
		{
			name:     "non-existent branch",
			input:    "non-existent",
			cwd:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := resolver.resolve(tt.input, tt.cwd)
			if result != tt.expected {
				t.Errorf("resolve(%q, %q) = %q, expected %q", tt.input, tt.cwd, result, tt.expected)
			}
		})
	}
}

func TestManagerConfig(t *testing.T) {
	cfg := &config.Config{
		Repository: config.Repository{},
		PostCreate: []config.PostCreateAction{
			{Source: ".env", Type: "symlink"},
			{Source: "node_modules", Type: "symlink"},
		},
	}

	mgr := NewManager("/home/user/project", "/home/user/project/.git", cfg)

	if mgr.RepoRoot != "/home/user/project" {
		t.Errorf("expected RepoRoot '/home/user/project', got %q", mgr.RepoRoot)
	}

	if mgr.BareDir != "/home/user/project/.git" {
		t.Errorf("expected BareDir '/home/user/project/.git', got %q", mgr.BareDir)
	}

	if len(mgr.Config.PostCreate) != 2 {
		t.Errorf("expected 2 post-create configs, got %d", len(mgr.Config.PostCreate))
	}
}
