package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMigrate_DefaultBranchDetection_InPlace tests that migrate detects the default branch
// and creates a worktree for it when migrating from a feature branch
func TestMigrate_DefaultBranchDetection_InPlace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-default-branch")

	// Setup: create a git repository with main and a feature branch
	repoDir := filepath.Join(tempDir, "test-repo")
	setupGitRepoWithMainBranch(t, repoDir)

	// Create a feature branch and checkout
	runGitSuccess(t, repoDir, "checkout", "-b", "feature/test")
	writeFile(t, filepath.Join(repoDir, "feature.txt"), "feature content")
	runGitSuccess(t, repoDir, "add", "feature.txt")
	runGitSuccess(t, repoDir, "commit", "-m", "Add feature")

	t.Run("migrate from feature branch creates main worktree", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "repo", "migrate", ".", "-i")

		assertOutputContains(t, stdout, "Migration successful")
		assertOutputContains(t, stdout, "Default branch: main (detected from remote)")

		// Check that both worktrees were created
		assertFileExists(t, filepath.Join(repoDir, "feature/test"))
		assertFileExists(t, filepath.Join(repoDir, "main"))

		// Verify current branch worktree is functional
		featureWorktree := filepath.Join(repoDir, "feature/test")
		stdout = runGitSuccess(t, featureWorktree, "status")
		assertOutputContains(t, stdout, "On branch feature/test")

		// Verify main worktree is functional
		mainWorktree := filepath.Join(repoDir, "main")
		stdout = runGitSuccess(t, mainWorktree, "status")
		assertOutputContains(t, stdout, "On branch main")
	})

	t.Run("default branch is set correctly in config", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "status")
		assertOutputContains(t, stdout, "Default branch: main")
	})
}

// TestMigrate_DefaultBranchDetection_FromMain tests that when migrating from main branch,
// no additional worktree is created
func TestMigrate_DefaultBranchDetection_FromMain(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-from-main")

	// Setup: create a git repository on main branch
	repoDir := filepath.Join(tempDir, "test-repo")
	setupGitRepoWithMainBranch(t, repoDir)

	t.Run("migrate from main creates only main worktree", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "repo", "migrate", ".", "-i")

		assertOutputContains(t, stdout, "Migration successful")
		// Should NOT contain "Default branch: main (detected from remote)" since we're already on main
		assertOutputNotContains(t, stdout, "detected from remote")

		// Check that only main worktree was created
		assertFileExists(t, filepath.Join(repoDir, "main"))

		// Verify there's only one worktree (plus bare repo)
		stdout = runBtSuccess(t, repoDir, "status")
		assertOutputContains(t, stdout, "main")
	})
}

// TestMigrate_DefaultBranchDetection_Destination tests default branch detection with --destination flag
func TestMigrate_DefaultBranchDetection_Destination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-default-branch-dest")

	// Setup: create a git repository with main and a feature branch
	repoDir := filepath.Join(tempDir, "source-repo")
	destDir := filepath.Join(tempDir, "dest-repo")
	setupGitRepoWithMainBranch(t, repoDir)

	// Create a feature branch and checkout
	runGitSuccess(t, repoDir, "checkout", "-b", "feature/auth")
	writeFile(t, filepath.Join(repoDir, "auth.txt"), "auth content")
	runGitSuccess(t, repoDir, "add", "auth.txt")
	runGitSuccess(t, repoDir, "commit", "-m", "Add auth")

	t.Run("migrate to destination from feature branch creates main worktree", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "repo", "migrate", ".", "-d", destDir)

		assertOutputContains(t, stdout, "Migration successful")
		assertOutputContains(t, stdout, "Default branch: main (detected from remote)")

		// Check that both worktrees were created at destination
		assertFileExists(t, filepath.Join(destDir, "feature/auth"))
		assertFileExists(t, filepath.Join(destDir, "main"))

		// Verify feature branch worktree is functional
		featureWorktree := filepath.Join(destDir, "feature/auth")
		stdout = runGitSuccess(t, featureWorktree, "status")
		assertOutputContains(t, stdout, "On branch feature/auth")

		// Verify main worktree is functional
		mainWorktree := filepath.Join(destDir, "main")
		stdout = runGitSuccess(t, mainWorktree, "status")
		assertOutputContains(t, stdout, "On branch main")
	})
}

// TestMigrate_DefaultBranchDetection_FallbackToMaster tests fallback to master branch
func TestMigrate_DefaultBranchDetection_FallbackToMaster(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-master-fallback")

	// Setup: create a git repository with master branch (no remote)
	repoDir := filepath.Join(tempDir, "test-repo")
	setupGitRepo(t, repoDir) // This creates a repo with master branch

	// Create a feature branch and checkout
	runGitSuccess(t, repoDir, "checkout", "-b", "feature/test")
	writeFile(t, filepath.Join(repoDir, "feature.txt"), "feature content")
	runGitSuccess(t, repoDir, "add", "feature.txt")
	runGitSuccess(t, repoDir, "commit", "-m", "Add feature")

	t.Run("migrate fallback to master when no remote", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "repo", "migrate", ".", "-i")

		assertOutputContains(t, stdout, "Migration successful")

		// Check that both worktrees were created (feature/test and master)
		assertFileExists(t, filepath.Join(repoDir, "feature/test"))
		assertFileExists(t, filepath.Join(repoDir, "master"))

		// Verify master worktree is functional
		masterWorktree := filepath.Join(repoDir, "master")
		stdout = runGitSuccess(t, masterWorktree, "status")
		assertOutputContains(t, stdout, "On branch master")
	})
}

// TestMigrate_DefaultBranchDetection_FallbackToCurrentBranch tests that when no remote,
// main, or master exists, the current branch is used as the default
func TestMigrate_DefaultBranchDetection_FallbackToCurrentBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-current-fallback")

	// Setup: create a git repository with only 'develop' branch (no main, no master, no remote)
	repoDir := filepath.Join(tempDir, "test-repo")
	setupGitRepoWithCustomBranch(t, repoDir, "develop")

	t.Run("migrate uses current branch as default when no main/master/remote", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "repo", "migrate", ".", "-i")

		assertOutputContains(t, stdout, "Migration successful")
		// Should NOT create additional worktree since current branch becomes default
		assertOutputNotContains(t, stdout, "detected from remote")

		// Check that only develop worktree was created
		assertFileExists(t, filepath.Join(repoDir, "develop"))
		assertFileNotExists(t, filepath.Join(repoDir, "main"))
		assertFileNotExists(t, filepath.Join(repoDir, "master"))

		// Verify develop worktree is functional
		developWorktree := filepath.Join(repoDir, "develop")
		stdout = runGitSuccess(t, developWorktree, "status")
		assertOutputContains(t, stdout, "On branch develop")

		// Verify default branch is set to develop
		stdout = runBtSuccess(t, repoDir, "status")
		assertOutputContains(t, stdout, "Default branch: develop")
	})
}

// TestMigrate_DefaultBranchDetection_FallbackFromFeatureToCurrentBranch tests that when
// no remote, main, or master exists, the current branch becomes default (no additional worktree)
func TestMigrate_DefaultBranchDetection_FallbackFromFeatureToCurrentBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-feature-fallback")

	// Setup: create a git repository with 'develop' branch, then checkout to feature branch
	repoDir := filepath.Join(tempDir, "test-repo")
	setupGitRepoWithCustomBranch(t, repoDir, "develop")

	// Create a feature branch and checkout
	runGitSuccess(t, repoDir, "checkout", "-b", "feature/new")
	writeFile(t, filepath.Join(repoDir, "feature.txt"), "feature content")
	runGitSuccess(t, repoDir, "add", "feature.txt")
	runGitSuccess(t, repoDir, "commit", "-m", "Add feature")

	t.Run("migrate from feature falls back to current branch when no main/master/remote", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "repo", "migrate", ".", "-i")

		assertOutputContains(t, stdout, "Migration successful")
		// No main/master to detect, so current branch (feature/new) becomes default
		assertOutputNotContains(t, stdout, "detected from remote")

		// Check that only feature/new worktree was created (no develop worktree)
		assertFileExists(t, filepath.Join(repoDir, "feature/new"))
		assertFileNotExists(t, filepath.Join(repoDir, "develop"))
		assertFileNotExists(t, filepath.Join(repoDir, "main"))
		assertFileNotExists(t, filepath.Join(repoDir, "master"))

		// Verify feature worktree is functional
		featureWorktree := filepath.Join(repoDir, "feature/new")
		stdout = runGitSuccess(t, featureWorktree, "status")
		assertOutputContains(t, stdout, "On branch feature/new")

		// Verify default branch is set to feature/new
		stdout = runBtSuccess(t, repoDir, "status")
		assertOutputContains(t, stdout, "Default branch: feature/new")
	})
}

// setupGitRepoWithCustomBranch creates a git repository with a custom branch name (no remote)
func setupGitRepoWithCustomBranch(t *testing.T, dir string, branchName string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	runGitSuccess(t, dir, "init", "-b", branchName)
	writeFile(t, filepath.Join(dir, "file1.txt"), "initial content")
	runGitSuccess(t, dir, "add", "file1.txt")
	runGitSuccess(t, dir, "commit", "-m", "Initial commit")
}

// setupGitRepoWithMainBranch creates a git repository with main as the default branch
// and a remote origin/HEAD pointing to main
func setupGitRepoWithMainBranch(t *testing.T, dir string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	runGitSuccess(t, dir, "init", "-b", "main")
	writeFile(t, filepath.Join(dir, "file1.txt"), "initial content")
	runGitSuccess(t, dir, "add", "file1.txt")
	runGitSuccess(t, dir, "commit", "-m", "Initial commit")

	// Add a remote and set origin/HEAD to main
	// We create a bare repo as the "remote" to simulate origin/HEAD
	remoteDir := filepath.Join(filepath.Dir(dir), "remote.git")
	runGitSuccess(t, filepath.Dir(dir), "clone", "--bare", dir, remoteDir)
	runGitSuccess(t, dir, "remote", "add", "origin", remoteDir)
	runGitSuccess(t, dir, "fetch", "origin")
	// Set the symbolic ref for origin/HEAD
	runGitSuccess(t, dir, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
}
