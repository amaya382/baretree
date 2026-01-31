package e2e

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// TestRename_Basic tests basic rename functionality
func TestRename_Basic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "rename-basic")

	// Initialize and add a worktree
	runBtSuccess(t, tempDir, "repo", "init", "rename-test")
	projectDir := filepath.Join(tempDir, "rename-test")
	runBtSuccess(t, projectDir, "add", "-b", "feature/old")

	t.Run("rename worktree with two arguments", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "rename", "feature/old", "feature/new")

		assertOutputContains(t, stdout, "Successfully renamed")
		assertOutputContains(t, stdout, "feature/old")
		assertOutputContains(t, stdout, "feature/new")

		// Verify old path doesn't exist
		assertFileNotExists(t, filepath.Join(projectDir, "feature", "old"))

		// Verify new path exists
		assertFileExists(t, filepath.Join(projectDir, "feature", "new"))

		// Verify branch was renamed
		stdout = runBtSuccess(t, projectDir, "list")
		assertOutputContains(t, stdout, "feature/new")
		assertOutputNotContains(t, stdout, "feature/old")
	})
}

// TestRename_CurrentWorktree tests renaming current worktree
func TestRename_CurrentWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "rename-current")

	runBtSuccess(t, tempDir, "repo", "init", "rename-test")
	projectDir := filepath.Join(tempDir, "rename-test")
	runBtSuccess(t, projectDir, "add", "-b", "feature/current")

	worktreeDir := filepath.Join(projectDir, "feature", "current")

	t.Run("rename from inside worktree", func(t *testing.T) {
		stdout := runBtSuccess(t, worktreeDir, "rename", "feature/renamed")

		assertOutputContains(t, stdout, "Successfully renamed")

		// Verify new path exists
		assertFileExists(t, filepath.Join(projectDir, "feature", "renamed"))

		// Verify in list
		stdout = runBtSuccess(t, projectDir, "list")
		assertOutputContains(t, stdout, "feature/renamed")
	})
}

// TestRename_HierarchicalToFlat tests renaming from hierarchical to flat name
func TestRename_HierarchicalToFlat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "rename-hierarchy")

	runBtSuccess(t, tempDir, "repo", "init", "rename-test")
	projectDir := filepath.Join(tempDir, "rename-test")
	runBtSuccess(t, projectDir, "add", "-b", "feature/nested/deep")

	t.Run("rename hierarchical to flat", func(t *testing.T) {
		runBtSuccess(t, projectDir, "rename", "feature/nested/deep", "flat-branch")

		// Verify new path exists
		assertFileExists(t, filepath.Join(projectDir, "flat-branch"))

		// Verify old path doesn't exist
		assertFileNotExists(t, filepath.Join(projectDir, "feature", "nested", "deep"))

		// Empty parent directories should be cleaned up
		assertFileNotExists(t, filepath.Join(projectDir, "feature", "nested"))
	})
}

// TestRename_FlatToHierarchical tests renaming from flat to hierarchical name
func TestRename_FlatToHierarchical(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "rename-flat-to-hier")

	runBtSuccess(t, tempDir, "repo", "init", "rename-test")
	projectDir := filepath.Join(tempDir, "rename-test")
	runBtSuccess(t, projectDir, "add", "-b", "simple")

	t.Run("rename flat to hierarchical", func(t *testing.T) {
		runBtSuccess(t, projectDir, "rename", "simple", "feature/auth/login")

		// Verify new path exists
		assertFileExists(t, filepath.Join(projectDir, "feature", "auth", "login"))

		// Verify old path doesn't exist
		assertFileNotExists(t, filepath.Join(projectDir, "simple"))
	})
}

// TestRename_ErrorCases tests error handling
func TestRename_ErrorCases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	t.Run("rename non-existent worktree", func(t *testing.T) {
		tempDir := createTempDir(t, "rename-error-nonexist")
		runBtSuccess(t, tempDir, "repo", "init", "test")
		projectDir := filepath.Join(tempDir, "test")

		_, stderr := runBtExpectError(t, projectDir, "rename", "nonexistent", "new-name")
		assertOutputContains(t, stderr, "worktree not found")
	})

	t.Run("rename to existing name", func(t *testing.T) {
		tempDir := createTempDir(t, "rename-error-exists")
		runBtSuccess(t, tempDir, "repo", "init", "test")
		projectDir := filepath.Join(tempDir, "test")
		runBtSuccess(t, projectDir, "add", "-b", "branch-a")
		runBtSuccess(t, projectDir, "add", "-b", "branch-b")

		_, stderr := runBtExpectError(t, projectDir, "rename", "branch-a", "branch-b")
		assertOutputContains(t, stderr, "already exists")
	})

	t.Run("rename with same name", func(t *testing.T) {
		tempDir := createTempDir(t, "rename-error-same")
		runBtSuccess(t, tempDir, "repo", "init", "test")
		projectDir := filepath.Join(tempDir, "test")
		runBtSuccess(t, projectDir, "add", "-b", "branch")

		_, stderr := runBtExpectError(t, projectDir, "rename", "branch", "branch")
		assertOutputContains(t, stderr, "same")
	})

	t.Run("rename inconsistent worktree shows repair message", func(t *testing.T) {
		tempDir := createTempDir(t, "rename-error-inconsistent")
		runBtSuccess(t, tempDir, "repo", "init", "test")
		projectDir := filepath.Join(tempDir, "test")
		runBtSuccess(t, projectDir, "add", "-b", "feature/test")

		// Manually rename branch to create inconsistency
		bareDir := filepath.Join(projectDir, ".bare")
		cmd := exec.Command("git", "branch", "-m", "feature/test", "different-name")
		cmd.Dir = bareDir
		_ = cmd.Run()

		_, stderr := runBtExpectError(t, projectDir, "rename", "feature/test", "feature/new")
		assertOutputContains(t, stderr, "inconsistent")
		assertOutputContains(t, stderr, "repair")
	})
}
