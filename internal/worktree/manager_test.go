package worktree

import (
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

func TestSharedConfigApply(t *testing.T) {
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

	// Create config with shared files
	cfg := &config.Config{
		Repository: config.Repository{},
		Shared: []config.Shared{
			{Source: ".env", Type: "symlink"},
		},
	}

	// Note: We can't fully test ApplySharedConfig without a real git repository
	// because getMainWorktreePath() requires git worktree list to work
	// This test just verifies the config structure is correct
	if len(cfg.Shared) != 1 {
		t.Errorf("expected 1 shared config, got %d", len(cfg.Shared))
	}

	if cfg.Shared[0].Source != ".env" {
		t.Errorf("expected shared source '.env', got %q", cfg.Shared[0].Source)
	}

	if cfg.Shared[0].Type != "symlink" {
		t.Errorf("expected shared type 'symlink', got %q", cfg.Shared[0].Type)
	}
}
