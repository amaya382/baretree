package git

import (
	"testing"
)

func TestParseWorktreeList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Worktree
	}{
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
		{
			name: "single main worktree",
			input: `worktree /home/user/project/main
HEAD abc1234567890abcdef1234567890abcdef123456
branch refs/heads/main

`,
			expected: []Worktree{
				{
					Path:   "/home/user/project/main",
					Head:   "abc1234567890abcdef1234567890abcdef123456",
					Branch: "main",
					IsMain: true,
				},
			},
		},
		{
			name: "multiple worktrees",
			input: `worktree /home/user/project/main
HEAD abc1234567890abcdef1234567890abcdef123456
branch refs/heads/main

worktree /home/user/project/feature/auth
HEAD def5678901234567890abcdef1234567890abcdef
branch refs/heads/feature/auth

worktree /home/user/project/bugfix/cors
HEAD 9abcdef01234567890abcdef1234567890abcdef0
branch refs/heads/bugfix/cors

`,
			expected: []Worktree{
				{
					Path:   "/home/user/project/main",
					Head:   "abc1234567890abcdef1234567890abcdef123456",
					Branch: "main",
					IsMain: true,
				},
				{
					Path:   "/home/user/project/feature/auth",
					Head:   "def5678901234567890abcdef1234567890abcdef",
					Branch: "feature/auth",
					IsMain: false,
				},
				{
					Path:   "/home/user/project/bugfix/cors",
					Head:   "9abcdef01234567890abcdef1234567890abcdef0",
					Branch: "bugfix/cors",
					IsMain: false,
				},
			},
		},
		{
			name: "detached HEAD worktree",
			input: `worktree /home/user/project/main
HEAD abc1234567890abcdef1234567890abcdef123456
branch refs/heads/main

worktree /home/user/project/detached
HEAD def5678901234567890abcdef1234567890abcdef
detached

`,
			expected: []Worktree{
				{
					Path:   "/home/user/project/main",
					Head:   "abc1234567890abcdef1234567890abcdef123456",
					Branch: "main",
					IsMain: true,
				},
				{
					Path:   "/home/user/project/detached",
					Head:   "def5678901234567890abcdef1234567890abcdef",
					Branch: "detached",
					IsMain: false,
				},
			},
		},
		{
			name: "worktree without trailing newline",
			input: `worktree /home/user/project/main
HEAD abc1234567890abcdef1234567890abcdef123456
branch refs/heads/main`,
			expected: []Worktree{
				{
					Path:   "/home/user/project/main",
					Head:   "abc1234567890abcdef1234567890abcdef123456",
					Branch: "main",
					IsMain: true,
				},
			},
		},
		{
			name: "bare repository with worktrees",
			input: `worktree /home/user/project/.bare
HEAD abc1234567890abcdef1234567890abcdef123456
bare

worktree /home/user/project/main
HEAD abc1234567890abcdef1234567890abcdef123456
branch refs/heads/main

worktree /home/user/project/feature/auth
HEAD def5678901234567890abcdef1234567890abcdef
branch refs/heads/feature/auth

`,
			expected: []Worktree{
				{
					Path:   "/home/user/project/.bare",
					Head:   "abc1234567890abcdef1234567890abcdef123456",
					Branch: "",
					IsMain: true,
					IsBare: true,
				},
				{
					Path:   "/home/user/project/main",
					Head:   "abc1234567890abcdef1234567890abcdef123456",
					Branch: "main",
					IsMain: false,
					IsBare: false,
				},
				{
					Path:   "/home/user/project/feature/auth",
					Head:   "def5678901234567890abcdef1234567890abcdef",
					Branch: "feature/auth",
					IsMain: false,
					IsBare: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseWorktreeList(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d worktrees, got %d", len(tt.expected), len(result))
				return
			}

			for i, wt := range result {
				if wt.Path != tt.expected[i].Path {
					t.Errorf("worktree[%d].Path = %q, expected %q", i, wt.Path, tt.expected[i].Path)
				}
				if wt.Head != tt.expected[i].Head {
					t.Errorf("worktree[%d].Head = %q, expected %q", i, wt.Head, tt.expected[i].Head)
				}
				if wt.Branch != tt.expected[i].Branch {
					t.Errorf("worktree[%d].Branch = %q, expected %q", i, wt.Branch, tt.expected[i].Branch)
				}
				if wt.IsMain != tt.expected[i].IsMain {
					t.Errorf("worktree[%d].IsMain = %v, expected %v", i, wt.IsMain, tt.expected[i].IsMain)
				}
				if wt.IsBare != tt.expected[i].IsBare {
					t.Errorf("worktree[%d].IsBare = %v, expected %v", i, wt.IsBare, tt.expected[i].IsBare)
				}
			}
		})
	}
}
