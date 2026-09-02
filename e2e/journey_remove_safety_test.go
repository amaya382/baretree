package e2e

import (
	"path/filepath"
	"testing"
)

// TestRemove_UnmergedBranchSafety verifies that `bt rm --with-branch` aborts
// before touching anything when the branch is not fully merged into the
// default branch, and that --force lets the removal proceed and drops both
// the worktree and the branch.
func TestRemove_UnmergedBranchSafety(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "remove-unmerged")

	runBtSuccess(t, tempDir, "repo", "init", "safety-repo")
	projectDir := filepath.Join(tempDir, "safety-repo")
	mainDir := filepath.Join(projectDir, "main")

	// Seed main with an initial commit so branches can diverge.
	writeFile(t, filepath.Join(mainDir, "seed.txt"), "seed")
	runGitSuccess(t, mainDir, "add", "seed.txt")
	runGitSuccess(t, mainDir, "commit", "-m", "seed")

	// Add a feature branch and give it a commit not present on main.
	runBtSuccess(t, projectDir, "add", "-b", "feature/unmerged")
	featureDir := filepath.Join(projectDir, "feature", "unmerged")
	writeFile(t, filepath.Join(featureDir, "unmerged.txt"), "unmerged work")
	runGitSuccess(t, featureDir, "add", "unmerged.txt")
	runGitSuccess(t, featureDir, "commit", "-m", "unmerged commit")

	t.Run("aborts and keeps both worktree and branch", func(t *testing.T) {
		_, stderr := runBtFailure(t, projectDir, "rm", "feature/unmerged", "--with-branch")

		assertOutputContains(t, stderr, "not fully merged")
		assertOutputContains(t, stderr, "--force")

		// Nothing was touched.
		assertFileExists(t, featureDir)
		stdout := runBtSuccess(t, projectDir, "list")
		assertOutputContains(t, stdout, "feature/unmerged")
	})

	t.Run("force removes both worktree and branch", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "rm", "feature/unmerged", "--with-branch", "--force")

		assertOutputContains(t, stdout, "Worktree removed")
		assertOutputContains(t, stdout, "Branch 'feature/unmerged' deleted")

		assertFileNotExists(t, featureDir)
		listStdout := runBtSuccess(t, projectDir, "list")
		assertOutputNotContains(t, listStdout, "feature/unmerged")
	})
}

// TestRemove_UncommittedChangesSafety verifies that `bt rm` aborts before
// touching the worktree when it has uncommitted changes, and that --force
// bypasses the check.
func TestRemove_UncommittedChangesSafety(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "remove-dirty")

	runBtSuccess(t, tempDir, "repo", "init", "dirty-repo")
	projectDir := filepath.Join(tempDir, "dirty-repo")
	mainDir := filepath.Join(projectDir, "main")

	writeFile(t, filepath.Join(mainDir, "seed.txt"), "seed")
	runGitSuccess(t, mainDir, "add", "seed.txt")
	runGitSuccess(t, mainDir, "commit", "-m", "seed")

	runBtSuccess(t, projectDir, "add", "-b", "feature/dirty")
	featureDir := filepath.Join(projectDir, "feature", "dirty")

	// Leave an uncommitted (untracked) file behind.
	writeFile(t, filepath.Join(featureDir, "scratch.txt"), "wip")

	t.Run("aborts without touching worktree", func(t *testing.T) {
		_, stderr := runBtFailure(t, projectDir, "rm", "feature/dirty")

		assertOutputContains(t, stderr, "uncommitted changes")
		assertOutputContains(t, stderr, "--force")
		assertFileExists(t, featureDir)
	})

	t.Run("force drops the worktree", func(t *testing.T) {
		runBtSuccess(t, projectDir, "rm", "feature/dirty", "--force")
		assertFileNotExists(t, featureDir)
	})
}

// TestRemove_UncommittedAndUnmergedReportedTogether verifies that both
// blockers are surfaced in a single pre-flight message when they coexist.
func TestRemove_UncommittedAndUnmergedReportedTogether(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "remove-both")

	runBtSuccess(t, tempDir, "repo", "init", "both-repo")
	projectDir := filepath.Join(tempDir, "both-repo")
	mainDir := filepath.Join(projectDir, "main")

	writeFile(t, filepath.Join(mainDir, "seed.txt"), "seed")
	runGitSuccess(t, mainDir, "add", "seed.txt")
	runGitSuccess(t, mainDir, "commit", "-m", "seed")

	runBtSuccess(t, projectDir, "add", "-b", "feature/both")
	featureDir := filepath.Join(projectDir, "feature", "both")

	writeFile(t, filepath.Join(featureDir, "unmerged.txt"), "unmerged")
	runGitSuccess(t, featureDir, "add", "unmerged.txt")
	runGitSuccess(t, featureDir, "commit", "-m", "unmerged commit")

	// Now dirty the tree as well.
	writeFile(t, filepath.Join(featureDir, "scratch.txt"), "wip")

	_, stderr := runBtFailure(t, projectDir, "rm", "feature/both", "--with-branch")

	assertOutputContains(t, stderr, "uncommitted changes")
	assertOutputContains(t, stderr, "not fully merged")
	assertFileExists(t, featureDir)
}
