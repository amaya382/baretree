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

	t.Run("behind=continue continues with warning when behind", func(t *testing.T) {
		// auto-fetch will update remote refs, making local main behind origin/main
		stdout := runBtSuccess(t, projectDir, "add", "-b", "feat/behind-test", "--behind=continue")

		// Should show warning about being behind but still proceed
		assertOutputContains(t, stdout, "Warning: 'main' is")
		assertOutputContains(t, stdout, "behind its upstream")
		assertOutputContains(t, stdout, "Worktree created")
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
		assertOutputContains(t, stdout, "Based on 'origin/feature/remote (remote)'")
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

		assertOutputContains(t, stdout, "Based on 'main (local)'")
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

		assertOutputContains(t, stdout, "Based on HEAD (main)")
		assertOutputContains(t, stdout, "Worktree created")
	})
}

// TestAddNewBranchWithCommitBase tests --base with a commit hash
func TestAddNewBranchWithCommitBase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "commit-base")

	runBtSuccess(t, tempDir, "repo", "init", "test-repo")
	projectDir := filepath.Join(tempDir, "test-repo")
	bareDir := filepath.Join(projectDir, ".git")

	// Get the full commit hash of HEAD
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = bareDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get commit hash: %v", err)
	}
	fullHash := strings.TrimSpace(string(output))

	t.Run("new branch based on full commit hash", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "add", "-b", "feat/from-commit", "--base", fullHash)

		assertOutputContains(t, stdout, "(commit)")
		assertOutputContains(t, stdout, "Worktree created")
		assertFileExists(t, filepath.Join(projectDir, "feat", "from-commit"))
	})

	t.Run("new branch based on short commit hash", func(t *testing.T) {
		shortHash := fullHash[:7]
		stdout := runBtSuccess(t, projectDir, "add", "-b", "feat/from-short-commit", "--base", shortHash)

		assertOutputContains(t, stdout, "(commit)")
		assertOutputContains(t, stdout, "Worktree created")
		assertFileExists(t, filepath.Join(projectDir, "feat", "from-short-commit"))
	})

	t.Run("error when base is invalid commit hash", func(t *testing.T) {
		_, stderr := runBtExpectError(t, projectDir, "add", "-b", "feat/bad-hash", "--base", "deadbeef00deadbeef00")

		assertOutputContains(t, stderr, "not found")
	})
}

// setupBehindRepo creates a repository where the local main branch is behind its upstream.
// Returns (projectDir, bareDir).
func setupBehindRepo(t *testing.T, prefix string) (string, string) {
	t.Helper()

	tempDir := createTempDir(t, prefix)
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

	return projectDir, bareDir
}

// TestAddBehindFlagContinue tests --behind=continue when base branch is behind
func TestAddBehindFlagContinue(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	projectDir, _ := setupBehindRepo(t, "behind-continue")

	t.Run("behind=continue proceeds with warning", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "add", "-b", "feat/behind-continue", "--behind=continue")

		assertOutputContains(t, stdout, "Warning: 'main' is")
		assertOutputContains(t, stdout, "behind its upstream")
		assertOutputContains(t, stdout, "Worktree created")
	})
}

// TestAddBehindFlagPull tests --behind=pull when base branch is behind.
// Also guards against a regression where --behind=pull advanced only the ref
// via update-ref, leaving the checked-out main worktree with the pre-pull
// snapshot on disk (files reported as unstaged changes).
func TestAddBehindFlagPull(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	projectDir, bareDir := setupBehindRepo(t, "behind-pull")
	mainWorktree := filepath.Join(projectDir, "main")

	t.Run("behind=pull pulls and then creates worktree", func(t *testing.T) {
		beforeCount := strings.TrimSpace(runGitSuccess(t, bareDir, "rev-list", "--count", "main"))

		stdout := runBtSuccess(t, projectDir, "add", "-b", "feat/behind-pull", "--behind=pull")

		assertOutputContains(t, stdout, "Warning: 'main' is")
		assertOutputContains(t, stdout, "Pulling 'main'")
		assertOutputContains(t, stdout, "is now up to date")
		assertOutputContains(t, stdout, "Worktree created")

		afterCount := strings.TrimSpace(runGitSuccess(t, bareDir, "rev-list", "--count", "main"))
		if afterCount == beforeCount {
			t.Errorf("expected main to be pulled (commit count unchanged: %s)", beforeCount)
		}

		// The main worktree must reflect the pulled commit: HEAD matches the
		// branch, the new file exists, and there are no phantom unstaged changes.
		mainHead := strings.TrimSpace(runGitSuccess(t, mainWorktree, "rev-parse", "HEAD"))
		branchHead := strings.TrimSpace(runGitSuccess(t, bareDir, "rev-parse", "main"))
		if mainHead != branchHead {
			t.Errorf("main worktree HEAD %s does not match branch head %s", mainHead, branchHead)
		}
		assertFileExists(t, filepath.Join(mainWorktree, "extra.txt"))
		if status := runGitSuccess(t, mainWorktree, "status", "--porcelain"); status != "" {
			t.Errorf("expected main worktree to be clean after pull, got:\n%s", status)
		}
	})
}

// TestAddBehindFlagPullDirtyWorktree ensures that when the pulled branch is
// checked out in a dirty worktree, bt add --behind=pull aborts and leaves the
// worktree (and its branch ref) untouched instead of silently diverging.
func TestAddBehindFlagPullDirtyWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	projectDir, bareDir := setupBehindRepo(t, "behind-pull-dirty")
	mainWorktree := filepath.Join(projectDir, "main")

	// Dirty extra.txt in the main worktree so a fast-forward that touches it
	// would refuse rather than merging around the change. setupBehindRepo's
	// upstream advance introduces extra.txt, so it is the exact file the pull
	// needs to update.
	dirtyFile := filepath.Join(mainWorktree, "extra.txt")
	if err := os.WriteFile(dirtyFile, []byte("dirty local changes"), 0644); err != nil {
		t.Fatalf("failed to dirty main worktree: %v", err)
	}
	runGitSuccess(t, mainWorktree, "add", "extra.txt")

	branchBefore := strings.TrimSpace(runGitSuccess(t, bareDir, "rev-parse", "main"))

	stdout, stderr := runBtExpectError(t, projectDir, "add", "-b", "feat/dirty-pull", "--behind=pull")
	combined := stdout + stderr
	assertOutputContains(t, combined, "Warning: 'main' is")
	assertOutputContains(t, combined, "failed to fast-forward 'main'")

	// The branch ref must not have moved.
	branchAfter := strings.TrimSpace(runGitSuccess(t, bareDir, "rev-parse", "main"))
	if branchAfter != branchBefore {
		t.Errorf("branch main advanced despite dirty worktree: %s -> %s", branchBefore, branchAfter)
	}

	// Local edits must be preserved.
	content, err := os.ReadFile(dirtyFile)
	if err != nil {
		t.Fatalf("failed to read extra.txt: %v", err)
	}
	if string(content) != "dirty local changes" {
		t.Errorf("expected dirty local changes to survive, got: %q", string(content))
	}

	// The new worktree must not have been created.
	assertFileNotExists(t, filepath.Join(projectDir, "feat", "dirty-pull"))
}

// TestAddBehindFlagPullNoWorktree covers --behind=pull when the target branch
// has no worktree yet (bt add <existing-branch> --behind=pull). The bare-safe
// update-ref path is expected: the ref advances even without a worktree.
func TestAddBehindFlagPullNoWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	projectDir, bareDir := setupBehindRepo(t, "behind-pull-no-wt")
	tempDir := filepath.Dir(projectDir)
	workPath := filepath.Join(tempDir, "work")

	// Create a fresh remote branch (no local worktree) and advance it once so
	// the eventual local tracking branch will be behind its upstream.
	const remoteBranch = "feat/no-wt-target"
	runGitSuccess(t, workPath, "checkout", "-b", remoteBranch)
	if err := os.WriteFile(filepath.Join(workPath, "no-wt-v1.txt"), []byte("v1"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGitSuccess(t, workPath, "add", ".")
	runGitSuccess(t, workPath, "commit", "-m", "no-wt v1")
	runGitSuccess(t, workPath, "push", "origin", remoteBranch)
	// Fetch so the bare repo learns about the new remote branch.
	runGitSuccess(t, bareDir, "fetch", "origin")

	// Create a local tracking branch without a worktree.
	runGitSuccess(t, bareDir, "branch", remoteBranch, "origin/"+remoteBranch)
	runGitSuccess(t, bareDir, "config", "branch."+remoteBranch+".remote", "origin")
	runGitSuccess(t, bareDir, "config", "branch."+remoteBranch+".merge", "refs/heads/"+remoteBranch)

	// Advance the remote so the local branch is behind its upstream.
	if err := os.WriteFile(filepath.Join(workPath, "no-wt-v2.txt"), []byte("v2"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGitSuccess(t, workPath, "add", ".")
	runGitSuccess(t, workPath, "commit", "-m", "no-wt v2")
	runGitSuccess(t, workPath, "push", "origin", remoteBranch)

	branchBefore := strings.TrimSpace(runGitSuccess(t, bareDir, "rev-parse", remoteBranch))

	stdout := runBtSuccess(t, projectDir, "add", remoteBranch, "--behind=pull")
	assertOutputContains(t, stdout, "Pulling '"+remoteBranch+"'")
	assertOutputContains(t, stdout, "is now up to date")
	assertOutputContains(t, stdout, "Worktree created")

	// Refresh remote-tracking refs before comparing (bt add already fetched,
	// but this documents the expected invariant plainly).
	runGitSuccess(t, bareDir, "fetch", "origin")
	branchAfter := strings.TrimSpace(runGitSuccess(t, bareDir, "rev-parse", remoteBranch))
	upstreamAfter := strings.TrimSpace(runGitSuccess(t, bareDir, "rev-parse", "origin/"+remoteBranch))
	if branchAfter == branchBefore {
		t.Errorf("%s did not advance: still at %s", remoteBranch, branchBefore)
	}
	if branchAfter != upstreamAfter {
		t.Errorf("%s did not fast-forward to upstream: branch=%s upstream=%s",
			remoteBranch, branchAfter, upstreamAfter)
	}
	assertFileExists(t, filepath.Join(projectDir, remoteBranch, "no-wt-v2.txt"))
}

// TestAddBehindFlagPullFromOtherWorktree ensures that even when bt add is
// invoked from a worktree other than the one being pulled, PullBranch still
// resolves the correct checked-out worktree and keeps it in sync with the ref.
func TestAddBehindFlagPullFromOtherWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	projectDir, bareDir := setupBehindRepo(t, "behind-pull-cross")
	mainWorktree := filepath.Join(projectDir, "main")

	// Create a side worktree first (main is behind, so use --behind=continue
	// to keep the setup separate from the pull under test).
	runBtSuccess(t, projectDir, "add", "-b", "feat/side", "--behind=continue")
	sideWorktree := filepath.Join(projectDir, "feat", "side")

	stdout := runBtSuccess(t, sideWorktree, "add", "-b", "feat/cross-target", "--behind=pull")
	assertOutputContains(t, stdout, "Pulling 'main'")
	assertOutputContains(t, stdout, "Worktree created")

	// Main worktree must be sync'd with the branch even though bt was invoked
	// from a different working directory.
	mainHead := strings.TrimSpace(runGitSuccess(t, mainWorktree, "rev-parse", "HEAD"))
	branchHead := strings.TrimSpace(runGitSuccess(t, bareDir, "rev-parse", "main"))
	if mainHead != branchHead {
		t.Errorf("main worktree HEAD %s does not match branch head %s", mainHead, branchHead)
	}
	assertFileExists(t, filepath.Join(mainWorktree, "extra.txt"))
	if status := runGitSuccess(t, mainWorktree, "status", "--porcelain"); status != "" {
		t.Errorf("expected main worktree to be clean after cross-worktree pull, got:\n%s", status)
	}
}

// TestAddBehindFlagAbort tests --behind=abort when base branch is behind
func TestAddBehindFlagAbort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	projectDir, _ := setupBehindRepo(t, "behind-abort")

	t.Run("behind=abort aborts when behind", func(t *testing.T) {
		stdout, stderr := runBtExpectError(t, projectDir, "add", "-b", "feat/behind-abort", "--behind=abort")

		combined := stdout + stderr
		assertOutputContains(t, combined, "Warning: 'main' is")
		assertOutputContains(t, combined, "aborted")
	})
}

// TestAddBehindFlagInvalid tests --behind with invalid value
func TestAddBehindFlagInvalid(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "behind-invalid")
	runBtSuccess(t, tempDir, "repo", "init", "test-repo")
	projectDir := filepath.Join(tempDir, "test-repo")

	t.Run("invalid behind value shows error", func(t *testing.T) {
		_, stderr := runBtExpectError(t, projectDir, "add", "-b", "feat/invalid", "--behind=invalid")

		assertOutputContains(t, stderr, "invalid value for --behind")
	})
}
