package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

// TestConfigDefaultBranch_Get tests getting the default branch
func TestConfigDefaultBranch_Get(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "config-default-branch-get")

	// Initialize a new repository
	runBtSuccess(t, tempDir, "repo", "init", "test-project")
	projectDir := filepath.Join(tempDir, "test-project")

	t.Run("get default branch", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "config", "default-branch")

		// Default branch should be "main"
		assertOutputContains(t, stdout, "main")
	})
}

// TestConfigDefaultBranch_Set tests setting the default branch
func TestConfigDefaultBranch_Set(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "config-default-branch-set")

	// Initialize a new repository
	runBtSuccess(t, tempDir, "repo", "init", "test-project")
	projectDir := filepath.Join(tempDir, "test-project")

	t.Run("set default branch", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "config", "default-branch", "develop")

		assertOutputContains(t, stdout, "Default branch set to 'develop'")
	})

	t.Run("verify default branch was set", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "config", "default-branch")

		assertOutputContains(t, stdout, "develop")
	})

	t.Run("verify git config was updated", func(t *testing.T) {
		bareDir := filepath.Join(projectDir, ".git")
		defaultBranch := getGitConfig(t, bareDir, "baretree.defaultbranch")
		if defaultBranch != "develop" {
			t.Errorf("expected baretree.defaultbranch to be 'develop', got %q", defaultBranch)
		}
	})
}

// TestConfigDefaultBranch_SetMultipleTimes tests setting the default branch multiple times
func TestConfigDefaultBranch_SetMultipleTimes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "config-default-branch-multi")

	// Initialize a new repository
	runBtSuccess(t, tempDir, "repo", "init", "test-project")
	projectDir := filepath.Join(tempDir, "test-project")

	t.Run("set default branch to develop", func(t *testing.T) {
		runBtSuccess(t, projectDir, "config", "default-branch", "develop")
		stdout := runBtSuccess(t, projectDir, "config", "default-branch")
		assertOutputContains(t, stdout, "develop")
	})

	t.Run("set default branch back to main", func(t *testing.T) {
		runBtSuccess(t, projectDir, "config", "default-branch", "main")
		stdout := runBtSuccess(t, projectDir, "config", "default-branch")
		assertOutputContains(t, stdout, "main")
	})

	t.Run("set default branch to master", func(t *testing.T) {
		runBtSuccess(t, projectDir, "config", "default-branch", "master")
		stdout := runBtSuccess(t, projectDir, "config", "default-branch")
		assertOutputContains(t, stdout, "master")
	})
}

// TestConfigDefaultBranch_FromWorktree tests running command from within a worktree
func TestConfigDefaultBranch_FromWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "config-default-branch-worktree")

	// Initialize a new repository
	runBtSuccess(t, tempDir, "repo", "init", "test-project")
	projectDir := filepath.Join(tempDir, "test-project")
	mainDir := filepath.Join(projectDir, "main")

	t.Run("get from worktree", func(t *testing.T) {
		stdout := runBtSuccess(t, mainDir, "config", "default-branch")
		assertOutputContains(t, stdout, "main")
	})

	t.Run("set from worktree", func(t *testing.T) {
		runBtSuccess(t, mainDir, "config", "default-branch", "develop")
		stdout := runBtSuccess(t, mainDir, "config", "default-branch")
		assertOutputContains(t, stdout, "develop")
	})
}

// TestConfigDefaultBranch_NotInBaretreeRepo tests error when not in a baretree repository
func TestConfigDefaultBranch_NotInBaretreeRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "config-default-branch-not-baretree")

	t.Run("get fails outside baretree repo", func(t *testing.T) {
		_, stderr := runBtExpectError(t, tempDir, "config", "default-branch")
		assertOutputContains(t, stderr, "not in a baretree repository")
	})

	t.Run("set fails outside baretree repo", func(t *testing.T) {
		_, stderr := runBtExpectError(t, tempDir, "config", "default-branch", "develop")
		assertOutputContains(t, stderr, "not in a baretree repository")
	})
}

// TestConfigDefaultBranch_Unset tests unsetting the default branch
func TestConfigDefaultBranch_Unset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "config-default-branch-unset")

	// Initialize a new repository
	runBtSuccess(t, tempDir, "repo", "init", "test-project")
	projectDir := filepath.Join(tempDir, "test-project")

	t.Run("set default branch to develop", func(t *testing.T) {
		runBtSuccess(t, projectDir, "config", "default-branch", "develop")
		stdout := runBtSuccess(t, projectDir, "config", "default-branch")
		assertOutputContains(t, stdout, "develop")
	})

	t.Run("unset default branch", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "config", "default-branch", "--unset")
		assertOutputContains(t, stdout, "Default branch setting removed")
	})

	t.Run("verify default branch reverts to main", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "config", "default-branch")
		assertOutputContains(t, stdout, "main")
	})

	t.Run("verify git config was removed", func(t *testing.T) {
		bareDir := filepath.Join(projectDir, ".git")
		defaultBranch := getGitConfig(t, bareDir, "baretree.defaultbranch")
		if defaultBranch != "" {
			t.Errorf("expected baretree.defaultbranch to be empty, got %q", defaultBranch)
		}
	})
}

// TestConfigDefaultBranch_UnsetWithArg tests error when using --unset with branch argument
func TestConfigDefaultBranch_UnsetWithArg(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "config-default-branch-unset-arg")

	// Initialize a new repository
	runBtSuccess(t, tempDir, "repo", "init", "test-project")
	projectDir := filepath.Join(tempDir, "test-project")

	t.Run("unset with branch argument fails", func(t *testing.T) {
		_, stderr := runBtExpectError(t, projectDir, "config", "default-branch", "--unset", "develop")
		assertOutputContains(t, stderr, "cannot specify branch name with --unset flag")
	})
}

// TestConfigDefaultBranch_UpdatesSyncToRoot tests that changing default branch updates sync-to-root symlinks
func TestConfigDefaultBranch_UpdatesSyncToRoot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "config-default-branch-synctoroot")

	// Initialize a new repository
	runBtSuccess(t, tempDir, "repo", "init", "test-project")
	projectDir := filepath.Join(tempDir, "test-project")
	mainDir := filepath.Join(projectDir, "main")

	// Create a file in main worktree
	testFile := filepath.Join(mainDir, "CLAUDE.md")
	err := os.WriteFile(testFile, []byte("# Claude Config"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	t.Run("setup sync-to-root", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "sync-to-root", "add", "CLAUDE.md")
		assertOutputContains(t, stdout, "CLAUDE.md")

		// Verify symlink points to main
		symlinkPath := filepath.Join(projectDir, "CLAUDE.md")
		linkTarget, err := os.Readlink(symlinkPath)
		if err != nil {
			t.Fatalf("failed to read symlink: %v", err)
		}
		if linkTarget != "main/CLAUDE.md" {
			t.Errorf("expected symlink to point to main/CLAUDE.md, got %s", linkTarget)
		}
	})

	// Create develop branch worktree with same file
	t.Run("create develop worktree", func(t *testing.T) {
		runBtSuccess(t, projectDir, "add", "-b", "develop")

		developDir := filepath.Join(projectDir, "develop")
		testFileInDevelop := filepath.Join(developDir, "CLAUDE.md")
		err := os.WriteFile(testFileInDevelop, []byte("# Claude Config (develop)"), 0644)
		if err != nil {
			t.Fatalf("failed to write test file in develop: %v", err)
		}
	})

	t.Run("change default branch to develop", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "config", "default-branch", "develop")
		assertOutputContains(t, stdout, "Default branch set to 'develop'")
		assertOutputContains(t, stdout, "Updating sync-to-root symlinks")
	})

	t.Run("verify symlink now points to develop", func(t *testing.T) {
		symlinkPath := filepath.Join(projectDir, "CLAUDE.md")
		linkTarget, err := os.Readlink(symlinkPath)
		if err != nil {
			t.Fatalf("failed to read symlink: %v", err)
		}
		if linkTarget != "develop/CLAUDE.md" {
			t.Errorf("expected symlink to point to develop/CLAUDE.md, got %s", linkTarget)
		}
	})

	t.Run("change default branch back to main", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "config", "default-branch", "main")
		assertOutputContains(t, stdout, "Default branch set to 'main'")
		assertOutputContains(t, stdout, "Updating sync-to-root symlinks")

		// Verify symlink points back to main
		symlinkPath := filepath.Join(projectDir, "CLAUDE.md")
		linkTarget, err := os.Readlink(symlinkPath)
		if err != nil {
			t.Fatalf("failed to read symlink: %v", err)
		}
		if linkTarget != "main/CLAUDE.md" {
			t.Errorf("expected symlink to point to main/CLAUDE.md, got %s", linkTarget)
		}
	})
}

// TestConfigDefaultBranch_UnsetUpdatesSyncToRoot tests that --unset updates sync-to-root symlinks
func TestConfigDefaultBranch_UnsetUpdatesSyncToRoot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "config-default-branch-unset-synctoroot")

	// Initialize a new repository
	runBtSuccess(t, tempDir, "repo", "init", "test-project")
	projectDir := filepath.Join(tempDir, "test-project")
	mainDir := filepath.Join(projectDir, "main")

	// Create a file in main worktree
	testFile := filepath.Join(mainDir, "CLAUDE.md")
	err := os.WriteFile(testFile, []byte("# Claude Config"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create develop branch worktree with same file
	runBtSuccess(t, projectDir, "add", "-b", "develop")
	developDir := filepath.Join(projectDir, "develop")
	testFileInDevelop := filepath.Join(developDir, "CLAUDE.md")
	_ = os.WriteFile(testFileInDevelop, []byte("# Claude Config (develop)"), 0644)

	// Set default branch to develop
	runBtSuccess(t, projectDir, "config", "default-branch", "develop")

	// Setup sync-to-root (will point to develop)
	runBtSuccess(t, projectDir, "sync-to-root", "add", "CLAUDE.md")

	t.Run("verify symlink points to develop", func(t *testing.T) {
		symlinkPath := filepath.Join(projectDir, "CLAUDE.md")
		linkTarget, err := os.Readlink(symlinkPath)
		if err != nil {
			t.Fatalf("failed to read symlink: %v", err)
		}
		if linkTarget != "develop/CLAUDE.md" {
			t.Errorf("expected symlink to point to develop/CLAUDE.md, got %s", linkTarget)
		}
	})

	t.Run("unset default branch reverts symlink to main", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "config", "default-branch", "--unset")
		assertOutputContains(t, stdout, "Default branch setting removed")
		assertOutputContains(t, stdout, "Updating sync-to-root symlinks")

		// Verify symlink points back to main
		symlinkPath := filepath.Join(projectDir, "CLAUDE.md")
		linkTarget, err := os.Readlink(symlinkPath)
		if err != nil {
			t.Fatalf("failed to read symlink: %v", err)
		}
		if linkTarget != "main/CLAUDE.md" {
			t.Errorf("expected symlink to point to main/CLAUDE.md, got %s", linkTarget)
		}
	})
}

// TestConfigDefaultBranch_NoSyncToRoot tests that changing default branch with no sync-to-root doesn't show update message
func TestConfigDefaultBranch_NoSyncToRoot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "config-default-branch-no-synctoroot")

	// Initialize a new repository
	runBtSuccess(t, tempDir, "repo", "init", "test-project")
	projectDir := filepath.Join(tempDir, "test-project")

	t.Run("change default branch without sync-to-root", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "config", "default-branch", "develop")
		assertOutputContains(t, stdout, "Default branch set to 'develop'")
		assertOutputNotContains(t, stdout, "Updating sync-to-root symlinks")
	})
}

// TestConfigDefaultBranch_SameBranch tests that setting the same branch doesn't trigger update
func TestConfigDefaultBranch_SameBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "config-default-branch-same")

	// Initialize a new repository
	runBtSuccess(t, tempDir, "repo", "init", "test-project")
	projectDir := filepath.Join(tempDir, "test-project")

	t.Run("setting same branch shows already set message", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "config", "default-branch", "main")
		assertOutputContains(t, stdout, "Default branch is already 'main'")
		assertOutputNotContains(t, stdout, "Updating sync-to-root symlinks")
	})
}

// TestConfigDefaultBranch_Help tests help output
func TestConfigDefaultBranch_Help(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "config-default-branch-help")

	t.Run("help shows usage", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "config", "default-branch", "--help")

		assertOutputContains(t, stdout, "Get or set the default branch")
		assertOutputContains(t, stdout, "bt config default-branch")
		assertOutputContains(t, stdout, "bt config default-branch main")
		assertOutputContains(t, stdout, "--unset")
	})
}
