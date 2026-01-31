package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

// TestJourney4_SharedFiles tests shared file configuration
func TestJourney4_SharedFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "journey4")

	// Clone
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "shared-test")
	projectDir := filepath.Join(tempDir, "shared-test")

	// Find the default branch worktree (main or master)
	var mainWorktree string
	if isDirectory(filepath.Join(projectDir, "main")) {
		mainWorktree = filepath.Join(projectDir, "main")
	} else if isDirectory(filepath.Join(projectDir, "master")) {
		mainWorktree = filepath.Join(projectDir, "master")
	} else {
		t.Fatal("could not find main or master worktree")
	}

	// Create shared file in main worktree
	t.Run("setup shared file", func(t *testing.T) {
		envPath := filepath.Join(mainWorktree, ".env")
		err := os.WriteFile(envPath, []byte("SECRET=test123\nAPI_KEY=abc"), 0644)
		if err != nil {
			t.Fatalf("failed to write .env: %v", err)
		}
	})

	// Configure shared files using bt shared add
	t.Run("configure shared files", func(t *testing.T) {
		runBtSuccess(t, projectDir, "shared", "add", ".env", "--type", "symlink")
	})

	// Add new worktree (should apply shared config)
	t.Run("add worktree with shared files", func(t *testing.T) {
		runBtSuccess(t, projectDir, "add", "-b", "feature/shared")

		featureDir := filepath.Join(projectDir, "feature", "shared")
		envPath := filepath.Join(featureDir, ".env")

		// Check that .env exists (as symlink)
		assertFileExists(t, envPath)
		assertIsSymlink(t, envPath)

		// Verify content is the same
		content, err := os.ReadFile(envPath)
		if err != nil {
			t.Fatalf("failed to read .env in feature: %v", err)
		}

		if string(content) != "SECRET=test123\nAPI_KEY=abc" {
			t.Errorf("unexpected .env content: %s", string(content))
		}
	})

	// Verify shared file is shown in status
	t.Run("status shows shared config", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "status")

		assertOutputContains(t, stdout, "Shared files")
		assertOutputContains(t, stdout, ".env")
		assertOutputContains(t, stdout, "symlink")
	})
}

// TestSharedFileCopy tests copy type shared files
func TestSharedFileCopy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "shared-copy")

	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "copy-test")
	projectDir := filepath.Join(tempDir, "copy-test")

	// Find main worktree
	var mainWorktree string
	if isDirectory(filepath.Join(projectDir, "main")) {
		mainWorktree = filepath.Join(projectDir, "main")
	} else {
		mainWorktree = filepath.Join(projectDir, "master")
	}

	// Create file to copy
	testFile := filepath.Join(mainWorktree, "config.local")
	_ = os.WriteFile(testFile, []byte("local config"), 0644)

	// Configure as copy using bt shared add
	runBtSuccess(t, projectDir, "shared", "add", "config.local", "--type", "copy")

	// Add worktree
	t.Run("copy type creates regular file", func(t *testing.T) {
		runBtSuccess(t, projectDir, "add", "-b", "feature/copy")

		featureDir := filepath.Join(projectDir, "feature", "copy")
		copiedFile := filepath.Join(featureDir, "config.local")

		assertFileExists(t, copiedFile)

		// Should NOT be a symlink
		info, err := os.Lstat(copiedFile)
		if err != nil {
			t.Fatalf("failed to stat file: %v", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			t.Error("expected regular file, got symlink")
		}

		// Content should match
		content, _ := os.ReadFile(copiedFile)
		if string(content) != "local config" {
			t.Errorf("unexpected content: %s", string(content))
		}
	})
}
