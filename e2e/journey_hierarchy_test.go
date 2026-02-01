package e2e

import (
	"path/filepath"
	"testing"
)

// TestJourney8_HierarchicalBranchNames tests branch names with slashes create proper directory hierarchy
func TestJourney8_HierarchicalBranchNames(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "journey8")

	// Clone
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "hierarchy-test")
	projectDir := filepath.Join(tempDir, "hierarchy-test")

	// Add branches with various hierarchy levels
	// Note: Git does not allow creating refs like "feature/auth" and "feature/auth/login"
	// at the same time because refs are stored as files/directories.
	// So we use unique names that don't conflict.
	t.Run("create hierarchical worktrees", func(t *testing.T) {
		// Two-level hierarchy
		runBtSuccess(t, projectDir, "add", "-b", "feature/auth")
		assertFileExists(t, filepath.Join(projectDir, "feature", "auth"))

		// Different two-level hierarchy (not conflicting with feature/auth)
		runBtSuccess(t, projectDir, "add", "-b", "feature/api")
		assertFileExists(t, filepath.Join(projectDir, "feature", "api"))

		// Three-level hierarchy (different prefix)
		runBtSuccess(t, projectDir, "add", "-b", "bugfix/urgent/cors")
		assertFileExists(t, filepath.Join(projectDir, "bugfix", "urgent", "cors"))

		// Another three-level hierarchy
		runBtSuccess(t, projectDir, "add", "-b", "release/v1/hotfix")
		assertFileExists(t, filepath.Join(projectDir, "release", "v1", "hotfix"))
	})

	// Verify directory structure
	t.Run("verify directory structure", func(t *testing.T) {
		// feature/ should be a directory
		assertFileExists(t, filepath.Join(projectDir, "feature"))
		if !isDirectory(filepath.Join(projectDir, "feature")) {
			t.Error("feature should be a directory")
		}

		// feature/auth should be a worktree directory
		assertFileExists(t, filepath.Join(projectDir, "feature", "auth"))
		if !isDirectory(filepath.Join(projectDir, "feature", "auth")) {
			t.Error("feature/auth should be a directory")
		}

		// bugfix/urgent/ should be a directory
		assertFileExists(t, filepath.Join(projectDir, "bugfix", "urgent"))
		if !isDirectory(filepath.Join(projectDir, "bugfix", "urgent")) {
			t.Error("bugfix/urgent should be a directory")
		}
	})

	// List should show all worktrees correctly
	t.Run("list shows hierarchical names", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "list")

		assertOutputContains(t, stdout, "feature/auth")
		assertOutputContains(t, stdout, "feature/api")
		assertOutputContains(t, stdout, "bugfix/urgent/cors")
		assertOutputContains(t, stdout, "release/v1/hotfix")
	})

	// cd should work with hierarchical names
	t.Run("cd to hierarchical worktrees", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "cd", "feature/auth")
		expectedPath := filepath.Join(projectDir, "feature", "auth")
		assertOutputContains(t, stdout, expectedPath)

		stdout = runBtSuccess(t, projectDir, "cd", "bugfix/urgent/cors")
		expectedPath = filepath.Join(projectDir, "bugfix", "urgent", "cors")
		assertOutputContains(t, stdout, expectedPath)
	})

	// Remove hierarchical worktree
	t.Run("remove hierarchical worktree", func(t *testing.T) {
		runBtSuccess(t, projectDir, "remove", "release/v1/hotfix", "--force")
		assertFileNotExists(t, filepath.Join(projectDir, "release", "v1", "hotfix"))

		// List should not show removed worktree
		stdout := runBtSuccess(t, projectDir, "list")
		assertOutputNotContains(t, stdout, "release/v1/hotfix")
	})
}

// TestRefConflict tests that conflicting branch names show a user-friendly error
// Git does not allow refs like "feat" and "feat/xxx" to coexist because refs are stored as files/directories
func TestRefConflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	t.Run("parent ref exists when creating child", func(t *testing.T) {
		tempDir := createTempDir(t, "ref-conflict-1")

		runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "conflict-test-1")
		projectDir := filepath.Join(tempDir, "conflict-test-1")

		// Create parent branch first
		runBtSuccess(t, projectDir, "add", "-b", "feat")
		assertFileExists(t, filepath.Join(projectDir, "feat"))

		// Try to create child branch - should fail with user-friendly error
		_, stderr := runBtExpectError(t, projectDir, "add", "-b", "feat/child")

		// Check error message contains useful information
		assertOutputContains(t, stderr, "feat")
		assertOutputContains(t, stderr, "conflict")
	})

	t.Run("child ref exists when creating parent", func(t *testing.T) {
		tempDir := createTempDir(t, "ref-conflict-2")

		runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "conflict-test-2")
		projectDir := filepath.Join(tempDir, "conflict-test-2")

		// Create child branch first
		runBtSuccess(t, projectDir, "add", "-b", "feature/auth")
		assertFileExists(t, filepath.Join(projectDir, "feature", "auth"))

		// Try to create parent branch - should fail with user-friendly error
		_, stderr := runBtExpectError(t, projectDir, "add", "-b", "feature")

		// Check error message contains useful information
		assertOutputContains(t, stderr, "feature")
		assertOutputContains(t, stderr, "conflict")
	})

	t.Run("deeply nested conflict", func(t *testing.T) {
		tempDir := createTempDir(t, "ref-conflict-3")

		runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "conflict-test-3")
		projectDir := filepath.Join(tempDir, "conflict-test-3")

		// Create deep branch first
		runBtSuccess(t, projectDir, "add", "-b", "a/b/c")
		assertFileExists(t, filepath.Join(projectDir, "a", "b", "c"))

		// Try to create intermediate branch - should fail
		_, stderr := runBtExpectError(t, projectDir, "add", "-b", "a/b")
		assertOutputContains(t, stderr, "conflict")

		// Try to create root branch - should fail
		_, stderr = runBtExpectError(t, projectDir, "add", "-b", "a")
		assertOutputContains(t, stderr, "conflict")
	})
}

// TestPathOutput tests that bt list --paths outputs correct paths
func TestPathOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "path-output")

	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "path-test")
	projectDir := filepath.Join(tempDir, "path-test")

	runBtSuccess(t, projectDir, "add", "-b", "feature/test")

	t.Run("list --paths outputs paths only", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "list", "--paths")

		// Should contain paths
		assertOutputContains(t, stdout, projectDir)

		// Should not contain decorations like @ or [M]
		assertOutputNotContains(t, stdout, "@")
		assertOutputNotContains(t, stdout, "[M]")
	})
}
