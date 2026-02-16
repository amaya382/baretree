package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	bareDir := filepath.Join(projectDir, ".git")
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
	bareDir := filepath.Join(projectDir, ".git")
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

// TestAddAutoFetch tests that auto-fetch is the default when remotes are configured
func TestAddAutoFetch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "remote-fetch")
	originPath := setupRemoteRepo(t, tempDir)

	runBtSuccess(t, tempDir, "repo", "clone", originPath, "test-repo")
	projectDir := filepath.Join(tempDir, "test-repo")

	// Configure fetch refspec
	bareDir := filepath.Join(projectDir, ".git")
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

	t.Run("auto-fetch gets new remote branches by default", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "add", "feature/new-after-clone")

		assertOutputContains(t, stdout, "Fetching from remotes")
		assertOutputContains(t, stdout, "Tracking remote branch")
		assertOutputContains(t, stdout, "Worktree created")
		assertFileExists(t, filepath.Join(projectDir, "feature", "new-after-clone"))
	})
}

// TestAddNoFetch tests the --no-fetch option skips auto-fetch
func TestAddNoFetch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "no-fetch")
	originPath := setupRemoteRepo(t, tempDir)

	runBtSuccess(t, tempDir, "repo", "clone", originPath, "test-repo")
	projectDir := filepath.Join(tempDir, "test-repo")

	// Configure fetch refspec
	bareDir := filepath.Join(projectDir, ".git")
	cmd := exec.Command("git", "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	cmd.Dir = bareDir
	_ = cmd.Run()

	// Create a new branch on origin after clone (not yet fetched)
	workPath := filepath.Join(tempDir, "work")
	cmd = exec.Command("git", "checkout", "-b", "feature/unfetched")
	cmd.Dir = workPath
	_ = cmd.Run()
	newFilePath := filepath.Join(workPath, "unfetched.txt")
	_ = os.WriteFile(newFilePath, []byte("unfetched"), 0644)
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = workPath
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "unfetched feature")
	cmd.Dir = workPath
	_ = cmd.Run()
	cmd = exec.Command("git", "push", "origin", "feature/unfetched")
	cmd.Dir = workPath
	_ = cmd.Run()

	t.Run("no-fetch skips auto-fetch so branch is not found", func(t *testing.T) {
		_, stderr := runBtExpectError(t, projectDir, "add", "--no-fetch", "feature/unfetched")

		assertOutputNotContains(t, stderr, "Fetching from remotes")
		assertOutputContains(t, stderr, "not found")
	})
}

// TestAddUpstreamBehindWarning tests the upstream behind warning in non-TTY mode
func TestAddUpstreamBehindWarning(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "upstream-behind")
	originPath := setupRemoteRepo(t, tempDir)

	runBtSuccess(t, tempDir, "repo", "clone", originPath, "test-repo")
	projectDir := filepath.Join(tempDir, "test-repo")

	bareDir := filepath.Join(projectDir, ".git")

	// Configure fetch refspec and fetch
	cmd := exec.Command("git", "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	cmd.Dir = bareDir
	_ = cmd.Run()
	cmd = exec.Command("git", "fetch", "origin")
	cmd.Dir = bareDir
	_ = cmd.Run()

	// Set upstream for main branch
	cmd = exec.Command("git", "config", "branch.main.remote", "origin")
	cmd.Dir = bareDir
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "branch.main.merge", "refs/heads/main")
	cmd.Dir = bareDir
	_ = cmd.Run()

	// Push a new commit to origin after clone
	workPath := filepath.Join(tempDir, "work")
	cmd = exec.Command("git", "checkout", "main")
	cmd.Dir = workPath
	_ = cmd.Run()
	newFilePath := filepath.Join(workPath, "extra.txt")
	_ = os.WriteFile(newFilePath, []byte("extra"), 0644)
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = workPath
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "extra commit")
	cmd.Dir = workPath
	_ = cmd.Run()
	cmd = exec.Command("git", "push", "origin", "main")
	cmd.Dir = workPath
	_ = cmd.Run()

	t.Run("aborts when default branch is behind upstream without force", func(t *testing.T) {
		// auto-fetch will update remote refs, making local main behind origin/main
		stdout, stderr := runBtExpectError(t, projectDir, "add", "-b", "feat/behind-test")

		// Should show warning about being behind
		combined := stdout + stderr
		assertOutputContains(t, combined, "Warning: 'main' is")
		assertOutputContains(t, combined, "behind its upstream")
	})

	t.Run("force skips behind check", func(t *testing.T) {
		// --force should skip the behind check and create the worktree
		stdout := runBtSuccess(t, projectDir, "add", "-b", "feat/behind-force", "--force")

		// Should not show warning
		assertOutputNotContains(t, stdout, "Warning:")
		assertOutputContains(t, stdout, "Worktree created")
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
	bareDir := filepath.Join(projectDir, ".git")
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

// TestAddNewBranchWithRemoteBase tests that --base with a remote-only branch
// correctly resolves the branch and creates the intended new branch name
func TestAddNewBranchWithRemoteBase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "remote-base")
	originPath := setupRemoteRepo(t, tempDir)

	runBtSuccess(t, tempDir, "repo", "clone", originPath, "test-repo")
	projectDir := filepath.Join(tempDir, "test-repo")

	// Configure fetch refspec and fetch
	bareDir := filepath.Join(projectDir, ".git")
	cmd := exec.Command("git", "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	cmd.Dir = bareDir
	_ = cmd.Run()
	cmd = exec.Command("git", "fetch", "origin")
	cmd.Dir = bareDir
	_ = cmd.Run()

	// Delete local feature/remote branch to ensure it's remote-only
	cmd = exec.Command("git", "branch", "-D", "feature/remote")
	cmd.Dir = bareDir
	_ = cmd.Run()

	t.Run("new branch based on remote-only branch", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "add", "-b", "feat/new", "--base", "feature/remote")

		// Should show the resolved remote ref as base
		assertOutputContains(t, stdout, "Based on 'origin/feature/remote'")
		assertOutputContains(t, stdout, "Worktree created")

		// Verify worktree was created with correct branch name
		assertFileExists(t, filepath.Join(projectDir, "feat", "new"))

		// Verify the branch name is feat/new, not feature/remote (DWIM bug fix)
		cmd := exec.Command("git", "branch", "--list", "feat/new")
		cmd.Dir = bareDir
		output, err := cmd.Output()
		if err != nil || !strings.Contains(string(output), "feat/new") {
			t.Errorf("expected branch 'feat/new' to exist, got: %s", string(output))
		}
	})
}

// TestAddNewBranchWithLocalBase tests --base with a local branch
func TestAddNewBranchWithLocalBase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "local-base")
	originPath := setupRemoteRepo(t, tempDir)

	runBtSuccess(t, tempDir, "repo", "clone", originPath, "test-repo")
	projectDir := filepath.Join(tempDir, "test-repo")

	t.Run("new branch based on local main", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "add", "-b", "feat/from-main", "--base", "main")

		assertOutputContains(t, stdout, "Based on 'main'")
		assertOutputContains(t, stdout, "Worktree created")
		assertFileExists(t, filepath.Join(projectDir, "feat", "from-main"))
	})
}

// TestAddNewBranchWithNonexistentBase tests --base with a branch that doesn't exist
func TestAddNewBranchWithNonexistentBase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "nonexistent-base")

	runBtSuccess(t, tempDir, "repo", "init", "test-repo")
	projectDir := filepath.Join(tempDir, "test-repo")

	t.Run("error when base branch does not exist", func(t *testing.T) {
		_, stderr := runBtExpectError(t, projectDir, "add", "-b", "feat/new", "--base", "nonexistent")

		assertOutputContains(t, stderr, "base branch 'nonexistent' not found")
	})
}

// TestAddNewBranchShowsBaseInfo tests that creating a new branch shows base information
func TestAddNewBranchShowsBaseInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "base-info")

	runBtSuccess(t, tempDir, "repo", "init", "test-repo")
	projectDir := filepath.Join(tempDir, "test-repo")

	t.Run("new branch without --base shows HEAD", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "add", "-b", "feat/no-base")

		assertOutputContains(t, stdout, "Based on HEAD")
		assertOutputContains(t, stdout, "Worktree created")
	})
}
