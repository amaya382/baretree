package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// getDefaultBranchDir returns the default branch directory (main for baretree repo)
func getDefaultBranchDir(projectDir string) string {
	// The test repo (baretree) uses main as default branch
	return filepath.Join(projectDir, "main")
}

// TestPostCreateAdd tests bt post-create add command
func TestPostCreateAdd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "postcreate-add")

	// Clone a repository
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "my-project")
	projectDir := filepath.Join(tempDir, "my-project")

	t.Run("add post-create file with symlink", func(t *testing.T) {
		// Create a file in default branch worktree
		defaultDir := getDefaultBranchDir(projectDir)
		writeFile(t, filepath.Join(defaultDir, ".env"), "SECRET=value")

		stdout := runBtSuccess(t, projectDir, "post-create", "add", "symlink", ".env")
		assertOutputContains(t, stdout, "Post-create action added")
	})

	t.Run("post-create file is symlinked when adding worktree", func(t *testing.T) {
		runBtSuccess(t, projectDir, "add", "-b", "feature/test")
		featureDir := filepath.Join(projectDir, "feature", "test")

		// Check .env is a symlink
		envPath := filepath.Join(featureDir, ".env")
		assertFileExists(t, envPath)
		assertIsSymlink(t, envPath)

		// Check content is accessible
		content, err := os.ReadFile(envPath)
		if err != nil {
			t.Fatalf("failed to read .env: %v", err)
		}
		if string(content) != "SECRET=value" {
			t.Errorf("expected content 'SECRET=value', got %q", string(content))
		}
	})
}

// TestPostCreateAddManaged tests bt post-create add (managed is default)
func TestPostCreateAddManaged(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "postcreate-managed")

	// Clone a repository
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "my-project")
	projectDir := filepath.Join(tempDir, "my-project")

	t.Run("add managed post-create file", func(t *testing.T) {
		// Create a file in main worktree
		defaultDir := getDefaultBranchDir(projectDir)
		writeFile(t, filepath.Join(defaultDir, ".env"), "SECRET=managed")

		// managed is now the default, so no flag needed
		stdout := runBtSuccess(t, projectDir, "post-create", "add", "symlink", ".env")
		assertOutputContains(t, stdout, "Post-create action added")

		// File should be moved to .shared
		assertFileExists(t, filepath.Join(projectDir, ".shared", ".env"))

		// Main worktree should have symlink
		assertIsSymlink(t, filepath.Join(defaultDir, ".env"))
	})

	t.Run("managed file symlinked in new worktree", func(t *testing.T) {
		runBtSuccess(t, projectDir, "add", "-b", "feature/managed")
		featureDir := filepath.Join(projectDir, "feature", "managed")

		envPath := filepath.Join(featureDir, ".env")
		assertFileExists(t, envPath)
		assertIsSymlink(t, envPath)
	})
}

// TestPostCreateAddConflict tests conflict detection
func TestPostCreateAddConflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "postcreate-conflict")

	// Clone a repository
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "my-project")
	projectDir := filepath.Join(tempDir, "my-project")

	// Create a worktree first
	runBtSuccess(t, projectDir, "add", "-b", "feature/conflict")
	featureDir := filepath.Join(projectDir, "feature", "conflict")

	// Create file in main
	defaultDir := getDefaultBranchDir(projectDir)
	writeFile(t, filepath.Join(defaultDir, ".env"), "SECRET=main")

	// Create conflicting file in feature
	writeFile(t, filepath.Join(featureDir, ".env"), "SECRET=feature")

	t.Run("fails when conflict exists", func(t *testing.T) {
		_, stderr := runBtFailure(t, projectDir, "post-create", "add", "symlink", ".env")
		assertOutputContains(t, stderr, "conflicts detected")
	})
}

// TestPostCreateRemove tests bt post-create remove
func TestPostCreateRemove(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "postcreate-remove")

	// Clone a repository
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "my-project")
	projectDir := filepath.Join(tempDir, "my-project")

	// Setup: add post-create file and create worktree
	defaultDir := getDefaultBranchDir(projectDir)
	writeFile(t, filepath.Join(defaultDir, ".env"), "SECRET=value")
	runBtSuccess(t, projectDir, "post-create", "add", "symlink", ".env")
	runBtSuccess(t, projectDir, "add", "-b", "feature/remove")

	featureDir := filepath.Join(projectDir, "feature", "remove")
	envPath := filepath.Join(featureDir, ".env")

	t.Run("remove post-create file removes symlinks", func(t *testing.T) {
		// Verify symlink exists
		assertFileExists(t, envPath)
		assertIsSymlink(t, envPath)

		// Remove post-create configuration
		stdout := runBtSuccess(t, projectDir, "post-create", "remove", ".env")
		assertOutputContains(t, stdout, "Post-create action removed")

		// Symlink should be removed
		assertFileNotExists(t, envPath)
	})
}

// TestPostCreateList tests bt post-create list
func TestPostCreateList(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "postcreate-list")

	// Clone a repository
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "my-project")
	projectDir := filepath.Join(tempDir, "my-project")

	t.Run("list shows no post-create actions initially", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "post-create", "list")
		assertOutputContains(t, stdout, "No post-create actions configured")
	})

	t.Run("list shows configured post-create files", func(t *testing.T) {
		defaultDir := getDefaultBranchDir(projectDir)
		writeFile(t, filepath.Join(defaultDir, ".env"), "SECRET=value")
		runBtSuccess(t, projectDir, "post-create", "add", "symlink", ".env")

		stdout := runBtSuccess(t, projectDir, "post-create", "list")
		assertOutputContains(t, stdout, ".env")
		assertOutputContains(t, stdout, "symlink")
	})
}

// TestPostCreateApply tests bt post-create apply
func TestPostCreateApply(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "postcreate-apply")

	// Clone a repository
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "my-project")
	projectDir := filepath.Join(tempDir, "my-project")

	// Create worktree first
	runBtSuccess(t, projectDir, "add", "-b", "feature/apply")

	// Create file in main
	defaultDir := getDefaultBranchDir(projectDir)
	writeFile(t, filepath.Join(defaultDir, ".env"), "SECRET=value")

	// Manually add to git-config (simulating manual config edit)
	bareDir := filepath.Join(projectDir, ".git")
	setGitConfig(t, bareDir, "baretree.postcreate", ".env:symlink")

	t.Run("apply applies config to existing worktrees", func(t *testing.T) {
		featureDir := filepath.Join(projectDir, "feature", "apply")
		envPath := filepath.Join(featureDir, ".env")

		// Should not exist yet
		assertFileNotExists(t, envPath)

		// Apply post-create config
		stdout := runBtSuccess(t, projectDir, "post-create", "apply")
		assertOutputContains(t, stdout, "Applied")

		// Now should exist as symlink
		assertFileExists(t, envPath)
		assertIsSymlink(t, envPath)
	})
}

// TestPostCreateApplyConflict tests conflict detection in apply
func TestPostCreateApplyConflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "postcreate-apply-conflict")

	// Clone a repository
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "my-project")
	projectDir := filepath.Join(tempDir, "my-project")

	// Create worktree and file
	runBtSuccess(t, projectDir, "add", "-b", "feature/conflict")
	featureDir := filepath.Join(projectDir, "feature", "conflict")

	// Create file in main
	defaultDir := getDefaultBranchDir(projectDir)
	writeFile(t, filepath.Join(defaultDir, ".env"), "SECRET=main")

	// Create conflicting file in feature
	writeFile(t, filepath.Join(featureDir, ".env"), "SECRET=feature")

	// Manually add to git-config
	bareDir := filepath.Join(projectDir, ".git")
	setGitConfig(t, bareDir, "baretree.postcreate", ".env:symlink")

	t.Run("apply fails with conflict", func(t *testing.T) {
		_, stderr := runBtFailure(t, projectDir, "post-create", "apply")
		assertOutputContains(t, stderr, "conflicts detected")
	})
}

// TestStatusShowsPostCreateInfo tests that bt status shows post-create status
func TestStatusShowsPostCreateInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "status-postcreate")

	// Clone a repository
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "my-project")
	projectDir := filepath.Join(tempDir, "my-project")

	// Add post-create file
	defaultDir := getDefaultBranchDir(projectDir)
	writeFile(t, filepath.Join(defaultDir, ".env"), "SECRET=value")
	runBtSuccess(t, projectDir, "post-create", "add", "symlink", ".env")

	// Add worktree
	runBtSuccess(t, projectDir, "add", "-b", "feature/status")

	t.Run("status shows post-create files with applied status", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "status")
		assertOutputContains(t, stdout, "Post-create actions:")
		assertOutputContains(t, stdout, ".env")
		assertOutputContains(t, stdout, "applied")
	})
}

// TestPostCreateAddCommand tests bt post-create add command (command type)
func TestPostCreateAddCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "postcreate-command")

	// Clone a repository
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "my-project")
	projectDir := filepath.Join(tempDir, "my-project")

	t.Run("add post-create command", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "post-create", "add", "command", "echo test")
		assertOutputContains(t, stdout, "Post-create command added")
	})

	t.Run("list shows command", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "post-create", "list")
		assertOutputContains(t, stdout, "command")
		assertOutputContains(t, stdout, "echo test")
	})

	t.Run("remove command", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "post-create", "remove", "echo test")
		assertOutputContains(t, stdout, "Post-create command removed")
	})
}

// TestPostCreateCommandExecution tests command execution on worktree creation
func TestPostCreateCommandExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "postcreate-exec")

	// Clone a repository
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "my-project")
	projectDir := filepath.Join(tempDir, "my-project")

	// Add command that creates a marker file
	runBtSuccess(t, projectDir, "post-create", "add", "command", "touch .command-executed")

	t.Run("command is executed when creating worktree", func(t *testing.T) {
		runBtSuccess(t, projectDir, "add", "-b", "feature/exec")

		featureDir := filepath.Join(projectDir, "feature", "exec")
		markerPath := filepath.Join(featureDir, ".command-executed")

		// Marker file should exist (command was executed)
		assertFileExists(t, markerPath)
	})
}

// TestPostCreateCommandFailure tests graceful handling of command failure
func TestPostCreateCommandFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "postcreate-fail")

	// Clone a repository
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "my-project")
	projectDir := filepath.Join(tempDir, "my-project")

	// Add command that will fail
	runBtSuccess(t, projectDir, "post-create", "add", "command", "false")

	t.Run("worktree is still created even when command fails", func(t *testing.T) {
		// bt add should succeed (command failure is a warning, not error)
		runBtSuccess(t, projectDir, "add", "-b", "feature/fail")

		featureDir := filepath.Join(projectDir, "feature", "fail")

		// Worktree should exist
		if !isDirectory(featureDir) {
			t.Errorf("expected worktree to exist at %s", featureDir)
		}
	})
}

// setGitConfig sets a git config value in the bare repository
func setGitConfig(t *testing.T, bareDir, key, value string) {
	t.Helper()
	cmd := exec.Command("git", "config", "--file", filepath.Join(bareDir, "config"), "--add", key, value)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to set git config %s: %v", key, err)
	}
}
