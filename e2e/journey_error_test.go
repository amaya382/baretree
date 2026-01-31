package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

// TestJourney7_ErrorHandling tests various error scenarios
func TestJourney7_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "journey7")

	// Test 1: Clone with invalid URL
	t.Run("clone invalid url", func(t *testing.T) {
		_, stderr := runBtFailure(t, tempDir, "repo", "clone", "https://invalid-url-that-does-not-exist.example.com/repo.git")

		// Should have some error message
		if stderr == "" {
			t.Error("expected error message for invalid URL")
		}
	})

	// Setup a valid repo for remaining tests
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "error-test")
	projectDir := filepath.Join(tempDir, "error-test")

	// Test 2: Add worktree with existing branch name using -b
	t.Run("add with existing branch", func(t *testing.T) {
		// First add creates the branch
		runBtSuccess(t, projectDir, "add", "-b", "test-branch")

		// Second add with same name should fail
		_, _ = runBtFailure(t, projectDir, "add", "-b", "test-branch")
	})

	// Test 3: Remove non-existent worktree
	t.Run("remove non-existent worktree", func(t *testing.T) {
		_, _ = runBtFailure(t, projectDir, "remove", "non-existent-worktree")
	})

	// Test 4: Commands outside baretree repo
	t.Run("commands outside baretree repo", func(t *testing.T) {
		outsideDir := createTempDir(t, "outside")

		_, _ = runBtFailure(t, outsideDir, "list")
		_, _ = runBtFailure(t, outsideDir, "add", "-b", "test")
		_, _ = runBtFailure(t, outsideDir, "status")
	})

	// Test 5: cd to non-existent worktree
	t.Run("cd to non-existent worktree", func(t *testing.T) {
		_, _ = runBtFailure(t, projectDir, "cd", "non-existent-branch")
	})

	// Test 6: Clone to existing directory
	t.Run("clone to existing directory", func(t *testing.T) {
		// Create a directory with the same name
		existingDir := filepath.Join(tempDir, "existing-dir")
		_ = os.MkdirAll(existingDir, 0755)

		_, _ = runBtFailure(t, tempDir, "repo", "clone", TestRepo, "existing-dir")
	})

	// Test 7: Repair non-existent worktree
	t.Run("repair non-existent worktree", func(t *testing.T) {
		_, _ = runBtFailure(t, projectDir, "repair", "non-existent-worktree")
	})

	// Test 8: Migrate non-git directory
	t.Run("migrate non-git directory", func(t *testing.T) {
		nonGitDir := createTempDir(t, "non-git")
		_, _ = runBtFailure(t, nonGitDir, "repo", "migrate", ".")
	})
}

// TestErrorMessages tests that error messages are helpful
func TestErrorMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "error-messages")

	// Commands outside baretree should mention "not in a baretree repository"
	t.Run("not in baretree repo message", func(t *testing.T) {
		_, stderr := runBtFailure(t, tempDir, "list")
		assertOutputContains(t, stderr, "baretree")
	})
}
