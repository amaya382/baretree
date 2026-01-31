package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupRemoteRepo creates a bare "origin" repository with remote branches
func setupRemoteRepo(t *testing.T, tempDir string) string {
	t.Helper()

	// Create bare origin repository
	originPath := filepath.Join(tempDir, "origin.git")
	cmd := exec.Command("git", "init", "--bare", originPath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create bare origin: %v", err)
	}

	// Set default branch to main
	cmd = exec.Command("git", "symbolic-ref", "HEAD", "refs/heads/main")
	cmd.Dir = originPath
	_ = cmd.Run()

	// Create a working repo and push some branches
	workPath := filepath.Join(tempDir, "work")
	cmd = exec.Command("git", "clone", originPath, workPath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to clone origin: %v", err)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = workPath
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = workPath
	_ = cmd.Run()

	// Create initial commit on main
	readmePath := filepath.Join(workPath, "README.md")
	_ = os.WriteFile(readmePath, []byte("# Test"), 0644)
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = workPath
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = workPath
	_ = cmd.Run()
	cmd = exec.Command("git", "push", "origin", "main")
	cmd.Dir = workPath
	_ = cmd.Run()

	// Create feature/remote branch
	cmd = exec.Command("git", "checkout", "-b", "feature/remote")
	cmd.Dir = workPath
	_ = cmd.Run()
	featurePath := filepath.Join(workPath, "feature.txt")
	_ = os.WriteFile(featurePath, []byte("feature"), 0644)
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = workPath
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "feature")
	cmd.Dir = workPath
	_ = cmd.Run()
	cmd = exec.Command("git", "push", "origin", "feature/remote")
	cmd.Dir = workPath
	_ = cmd.Run()

	return originPath
}

// TestAddRemoteBranch tests adding a remote branch as worktree
func TestAddRemoteBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "remote-add")
	originPath := setupRemoteRepo(t, tempDir)

	// Clone with baretree
	runBtSuccess(t, tempDir, "repo", "clone", originPath, "test-repo")
	projectDir := filepath.Join(tempDir, "test-repo")

	// Configure fetch refspec (needed for bare clone)
	bareDir := filepath.Join(projectDir, ".bare")
	cmd := exec.Command("git", "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	cmd.Dir = bareDir
	_ = cmd.Run()
	cmd = exec.Command("git", "fetch", "origin")
	cmd.Dir = bareDir
	_ = cmd.Run()

	// Delete local branch to ensure we test remote tracking
	// (bare clone may have created local branches)
	cmd = exec.Command("git", "branch", "-D", "feature/remote")
	cmd.Dir = bareDir
	_ = cmd.Run() // Ignore error if branch doesn't exist

	t.Run("add remote branch auto-detect", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "add", "feature/remote")

		assertOutputContains(t, stdout, "Tracking remote branch")
		assertOutputContains(t, stdout, "origin/feature/remote")
		assertOutputContains(t, stdout, "Worktree created")

		// Verify worktree exists
		assertFileExists(t, filepath.Join(projectDir, "feature", "remote"))

		// Verify tracking is set up
		cmd := exec.Command("git", "branch", "-vv")
		cmd.Dir = bareDir
		output, _ := cmd.Output()
		assertOutputContains(t, string(output), "origin/feature/remote")
	})
}

// TestAddRemoteBranchExplicit tests adding a remote branch with explicit remote/branch format
func TestAddRemoteBranchExplicit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "remote-explicit")
	originPath := setupRemoteRepo(t, tempDir)

	runBtSuccess(t, tempDir, "repo", "clone", originPath, "test-repo")
	projectDir := filepath.Join(tempDir, "test-repo")

	// Configure fetch refspec
	bareDir := filepath.Join(projectDir, ".bare")
	cmd := exec.Command("git", "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	cmd.Dir = bareDir
	_ = cmd.Run()
	cmd = exec.Command("git", "fetch", "origin")
	cmd.Dir = bareDir
	_ = cmd.Run()

	t.Run("add with explicit origin/branch format", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "add", "origin/feature/remote")

		assertOutputContains(t, stdout, "Worktree created")
		assertFileExists(t, filepath.Join(projectDir, "feature", "remote"))
	})
}

// TestAddWithFetch tests the --fetch option
func TestAddWithFetch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "remote-fetch")
	originPath := setupRemoteRepo(t, tempDir)

	runBtSuccess(t, tempDir, "repo", "clone", originPath, "test-repo")
	projectDir := filepath.Join(tempDir, "test-repo")

	// Configure fetch refspec
	bareDir := filepath.Join(projectDir, ".bare")
	cmd := exec.Command("git", "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	cmd.Dir = bareDir
	_ = cmd.Run()

	// Create a new branch on origin after clone
	workPath := filepath.Join(tempDir, "work")
	cmd = exec.Command("git", "checkout", "-b", "feature/new-after-clone")
	cmd.Dir = workPath
	_ = cmd.Run()
	newFilePath := filepath.Join(workPath, "new.txt")
	_ = os.WriteFile(newFilePath, []byte("new"), 0644)
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = workPath
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "new feature")
	cmd.Dir = workPath
	_ = cmd.Run()
	cmd = exec.Command("git", "push", "origin", "feature/new-after-clone")
	cmd.Dir = workPath
	_ = cmd.Run()

	t.Run("add with --fetch gets new remote branches", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "add", "--fetch", "feature/new-after-clone")

		assertOutputContains(t, stdout, "Fetching from remotes")
		assertOutputContains(t, stdout, "Tracking remote branch")
		assertOutputContains(t, stdout, "Worktree created")
		assertFileExists(t, filepath.Join(projectDir, "feature", "new-after-clone"))
	})
}

// TestAddBranchNotFound tests error when branch doesn't exist
func TestAddBranchNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "branch-not-found")

	runBtSuccess(t, tempDir, "repo", "init", "test-repo")
	projectDir := filepath.Join(tempDir, "test-repo")

	t.Run("add non-existent branch shows helpful error", func(t *testing.T) {
		_, stderr := runBtExpectError(t, projectDir, "add", "nonexistent-branch")

		assertOutputContains(t, stderr, "not found")
		assertOutputContains(t, stderr, "bt add -b nonexistent-branch")
	})
}

// TestAddLocalBranchPriority tests that local branches take priority
func TestAddLocalBranchPriority(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "local-priority")
	originPath := setupRemoteRepo(t, tempDir)

	runBtSuccess(t, tempDir, "repo", "clone", originPath, "test-repo")
	projectDir := filepath.Join(tempDir, "test-repo")

	// Configure fetch refspec and fetch
	bareDir := filepath.Join(projectDir, ".bare")
	cmd := exec.Command("git", "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	cmd.Dir = bareDir
	_ = cmd.Run()
	cmd = exec.Command("git", "fetch", "origin")
	cmd.Dir = bareDir
	_ = cmd.Run()

	// Create a local branch with different content
	cmd = exec.Command("git", "branch", "feature/remote")
	cmd.Dir = bareDir
	_ = cmd.Run()

	t.Run("local branch takes priority over remote", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "add", "feature/remote")

		// Should NOT say "Tracking remote branch" since local exists
		assertOutputNotContains(t, stdout, "Tracking remote branch")
		assertOutputContains(t, stdout, "Worktree created")
	})
}
