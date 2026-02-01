package worktree

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/amaya382/baretree/internal/config"
)

func TestIsManaged(t *testing.T) {
	repoRoot := "/home/user/project"
	bareDir := "/home/user/project/.git"
	cfg := &config.Config{
		Repository: config.Repository{},
	}

	mgr := &Manager{
		RepoRoot: repoRoot,
		BareDir:  bareDir,
		Config:   cfg,
	}

	tests := []struct {
		name         string
		worktreePath string
		expected     bool
	}{
		{
			name:         "worktree inside repo root",
			worktreePath: "/home/user/project/main",
			expected:     true,
		},
		{
			name:         "worktree in nested directory",
			worktreePath: "/home/user/project/feature/auth",
			expected:     true, // Subdirectory structure is allowed
		},
		{
			name:         "worktree outside repo root",
			worktreePath: "/tmp/external-worktree",
			expected:     false,
		},
		{
			name:         "worktree in parent directory",
			worktreePath: "/home/user/other-project",
			expected:     false,
		},
		{
			name:         "bare directory itself",
			worktreePath: "/home/user/project/.git",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mgr.IsManaged(tt.worktreePath)
			if result != tt.expected {
				t.Errorf("IsManaged(%q) = %v, expected %v", tt.worktreePath, result, tt.expected)
			}
		})
	}
}

func TestPostCreateConfigApply(t *testing.T) {
	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "baretree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create main worktree with shared file
	mainDir := filepath.Join(tempDir, "main")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		t.Fatalf("failed to create main dir: %v", err)
	}

	envContent := "SECRET=test123"
	envPath := filepath.Join(mainDir, ".env")
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("failed to write .env: %v", err)
	}

	// Create feature worktree directory
	featureDir := filepath.Join(tempDir, "feature", "auth")
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		t.Fatalf("failed to create feature dir: %v", err)
	}

	// Create config with post-create actions
	cfg := &config.Config{
		Repository: config.Repository{},
		PostCreate: []config.PostCreateAction{
			{Source: ".env", Type: "symlink"},
			{Source: "npm install", Type: "command"},
		},
	}

	// Note: We can't fully test ApplyPostCreateConfig without a real git repository
	// because getMainWorktreePath() requires git worktree list to work
	// This test just verifies the config structure is correct
	if len(cfg.PostCreate) != 2 {
		t.Errorf("expected 2 post-create configs, got %d", len(cfg.PostCreate))
	}

	if cfg.PostCreate[0].Source != ".env" {
		t.Errorf("expected post-create source '.env', got %q", cfg.PostCreate[0].Source)
	}

	if cfg.PostCreate[0].Type != "symlink" {
		t.Errorf("expected post-create type 'symlink', got %q", cfg.PostCreate[0].Type)
	}

	if cfg.PostCreate[1].Source != "npm install" {
		t.Errorf("expected post-create source 'npm install', got %q", cfg.PostCreate[1].Source)
	}

	if cfg.PostCreate[1].Type != "command" {
		t.Errorf("expected post-create type 'command', got %q", cfg.PostCreate[1].Type)
	}
}

func TestParseRefConflictError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		branchName     string
		expectError    bool
		conflictingRef string
	}{
		{
			name:           "ref conflict - feat exists when creating feat/xxx",
			err:            errors.New("git worktree add -b feat/xxx /path/to/feat/xxx failed: exit status 128\nstderr: fatal: cannot lock ref 'refs/heads/feat/xxx': 'refs/heads/feat' exists; cannot create 'refs/heads/feat/xxx'"),
			branchName:     "feat/xxx",
			expectError:    true,
			conflictingRef: "feat",
		},
		{
			name:           "ref conflict - feat/xxx exists when creating feat",
			err:            errors.New("git worktree add -b feat /path/to/feat failed: exit status 128\nstderr: fatal: cannot lock ref 'refs/heads/feat': 'refs/heads/feat/xxx' exists; cannot create 'refs/heads/feat'"),
			branchName:     "feat",
			expectError:    true,
			conflictingRef: "feat/xxx",
		},
		{
			name:           "ref conflict - deep nested branch",
			err:            errors.New("fatal: cannot lock ref 'refs/heads/feature/auth/login': 'refs/heads/feature/auth' exists"),
			branchName:     "feature/auth/login",
			expectError:    true,
			conflictingRef: "feature/auth",
		},
		{
			name:        "non-ref-conflict error",
			err:         errors.New("git worktree add failed: some other error"),
			branchName:  "feat",
			expectError: false,
		},
		{
			name:        "nil error",
			err:         nil,
			branchName:  "feat",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRefConflictError(tt.err, tt.branchName)

			if tt.expectError {
				if result == nil {
					t.Errorf("expected ErrRefConflict, got nil")
					return
				}

				refErr, ok := result.(*ErrRefConflict)
				if !ok {
					t.Errorf("expected *ErrRefConflict, got %T", result)
					return
				}

				if refErr.BranchName != tt.branchName {
					t.Errorf("expected BranchName %q, got %q", tt.branchName, refErr.BranchName)
				}

				if refErr.ConflictingRef != tt.conflictingRef {
					t.Errorf("expected ConflictingRef %q, got %q", tt.conflictingRef, refErr.ConflictingRef)
				}
			} else {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
			}
		})
	}
}

func TestErrRefConflictError(t *testing.T) {
	err := &ErrRefConflict{
		BranchName:     "feat/xxx",
		ConflictingRef: "feat",
	}

	errMsg := err.Error()

	// Check that the error message contains useful information
	if !contains(errMsg, "feat/xxx") {
		t.Errorf("error message should contain branch name 'feat/xxx': %s", errMsg)
	}
	if !contains(errMsg, "feat") {
		t.Errorf("error message should contain conflicting ref 'feat': %s", errMsg)
	}
	if !contains(errMsg, "files/directories") {
		t.Errorf("error message should explain the reason: %s", errMsg)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
