package repository

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/amaya382/baretree/internal/config"
)

// createTestBareRepo creates a bare git repository for testing
func createTestBareRepo(t *testing.T, tempDir, bareDir string) string {
	t.Helper()
	barePath := filepath.Join(tempDir, bareDir)
	if err := os.MkdirAll(barePath, 0755); err != nil {
		t.Fatalf("failed to create bare dir: %v", err)
	}
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = barePath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init bare repo: %v", err)
	}
	return barePath
}

func TestIsBaretreeRepo(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "baretree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Without bare repo
	if IsBaretreeRepo(tempDir) {
		t.Error("expected false for directory without bare repo, got true")
	}

	// Create bare repository
	createTestBareRepo(t, tempDir, ".bare")

	// With bare repo - should be true (baretree is identified by bare repo structure)
	if !IsBaretreeRepo(tempDir) {
		t.Error("expected true for directory with bare repo, got false")
	}
}

func TestFindRoot(t *testing.T) {
	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "baretree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create bare repository
	createTestBareRepo(t, tempDir, ".bare")

	// Initialize config
	if err := config.InitializeBaretreeConfig(tempDir, ".bare", "main"); err != nil {
		t.Fatalf("failed to initialize config: %v", err)
	}

	// Create nested worktree directory
	worktreePath := filepath.Join(tempDir, "feature", "auth", "src")
	if err := os.MkdirAll(worktreePath, 0755); err != nil {
		t.Fatalf("failed to create worktree dir: %v", err)
	}

	// Find root from worktree
	root, err := FindRoot(worktreePath)
	if err != nil {
		t.Fatalf("failed to find root: %v", err)
	}

	if root != tempDir {
		t.Errorf("expected root %q, got %q", tempDir, root)
	}
}

func TestGetBareRepoPath(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "baretree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create bare repository
	barePath := createTestBareRepo(t, tempDir, ".bare")

	// Initialize config
	if err := config.InitializeBaretreeConfig(tempDir, ".bare", "main"); err != nil {
		t.Fatalf("failed to initialize config: %v", err)
	}

	// Get bare repo path
	path, err := GetBareRepoPath(tempDir)
	if err != nil {
		t.Fatalf("failed to get bare repo path: %v", err)
	}

	if path != barePath {
		t.Errorf("expected %q, got %q", barePath, path)
	}
}

func TestGetBareRepoPathNotFound(t *testing.T) {
	// Create temp directory without bare repo
	tempDir, err := os.MkdirTemp("", "baretree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Try to get bare repo path (should fail)
	_, err = GetBareRepoPath(tempDir)
	if err == nil {
		t.Error("expected error when bare repo not found, got nil")
	}
}
