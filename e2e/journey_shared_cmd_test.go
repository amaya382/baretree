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

// TestSharedAdd tests bt shared add command
func TestSharedAdd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "shared-add")

	// Clone a repository
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "my-project")
	projectDir := filepath.Join(tempDir, "my-project")

	t.Run("add shared file with symlink", func(t *testing.T) {
		// Create a file in default branch worktree
		defaultDir := getDefaultBranchDir(projectDir)
		writeFile(t, filepath.Join(defaultDir, ".env"), "SECRET=value")

		stdout := runBtSuccess(t, projectDir, "shared", "add", ".env", "--type", "symlink")
		assertOutputContains(t, stdout, "Shared configuration added")
	})

	t.Run("shared file is symlinked when adding worktree", func(t *testing.T) {
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

// TestSharedAddManaged tests bt shared add --managed
func TestSharedAddManaged(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "shared-managed")

	// Clone a repository
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "my-project")
	projectDir := filepath.Join(tempDir, "my-project")

	t.Run("add managed shared file", func(t *testing.T) {
		// Create a file in main worktree
		defaultDir := getDefaultBranchDir(projectDir)
		writeFile(t, filepath.Join(defaultDir, ".env"), "SECRET=managed")

		// managed is now the default, so no flag needed
		stdout := runBtSuccess(t, projectDir, "shared", "add", ".env", "--type", "symlink")
		assertOutputContains(t, stdout, "Shared configuration added")

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

// TestSharedAddConflict tests conflict detection
func TestSharedAddConflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "shared-conflict")

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
		_, stderr := runBtFailure(t, projectDir, "shared", "add", ".env", "--type", "symlink")
		assertOutputContains(t, stderr, "conflicts detected")
	})
}

// TestSharedRemove tests bt shared remove
func TestSharedRemove(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "shared-remove")

	// Clone a repository
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "my-project")
	projectDir := filepath.Join(tempDir, "my-project")

	// Setup: add shared file and create worktree
	defaultDir := getDefaultBranchDir(projectDir)
	writeFile(t, filepath.Join(defaultDir, ".env"), "SECRET=value")
	runBtSuccess(t, projectDir, "shared", "add", ".env", "--type", "symlink")
	runBtSuccess(t, projectDir, "add", "-b", "feature/remove")

	featureDir := filepath.Join(projectDir, "feature", "remove")
	envPath := filepath.Join(featureDir, ".env")

	t.Run("remove shared file removes symlinks", func(t *testing.T) {
		// Verify symlink exists
		assertFileExists(t, envPath)
		assertIsSymlink(t, envPath)

		// Remove shared configuration
		stdout := runBtSuccess(t, projectDir, "shared", "remove", ".env")
		assertOutputContains(t, stdout, "Shared configuration removed")

		// Symlink should be removed
		assertFileNotExists(t, envPath)
	})
}

// TestSharedList tests bt shared list
func TestSharedList(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "shared-list")

	// Clone a repository
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "my-project")
	projectDir := filepath.Join(tempDir, "my-project")

	t.Run("list shows no shared files initially", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "shared", "list")
		assertOutputContains(t, stdout, "No shared files configured")
	})

	t.Run("list shows configured shared files", func(t *testing.T) {
		defaultDir := getDefaultBranchDir(projectDir)
		writeFile(t, filepath.Join(defaultDir, ".env"), "SECRET=value")
		runBtSuccess(t, projectDir, "shared", "add", ".env", "--type", "symlink")

		stdout := runBtSuccess(t, projectDir, "shared", "list")
		assertOutputContains(t, stdout, ".env")
		assertOutputContains(t, stdout, "symlink")
	})
}

// TestSharedApply tests bt shared apply
func TestSharedApply(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "shared-apply")

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
	setGitConfig(t, bareDir, "baretree.shared", ".env:symlink")

	t.Run("apply applies config to existing worktrees", func(t *testing.T) {
		featureDir := filepath.Join(projectDir, "feature", "apply")
		envPath := filepath.Join(featureDir, ".env")

		// Should not exist yet
		assertFileNotExists(t, envPath)

		// Apply shared config
		stdout := runBtSuccess(t, projectDir, "shared", "apply")
		assertOutputContains(t, stdout, "Applied")

		// Now should exist as symlink
		assertFileExists(t, envPath)
		assertIsSymlink(t, envPath)
	})
}

// TestSharedApplyConflict tests conflict detection in apply
func TestSharedApplyConflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "shared-apply-conflict")

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
	setGitConfig(t, bareDir, "baretree.shared", ".env:symlink")

	t.Run("apply fails with conflict", func(t *testing.T) {
		_, stderr := runBtFailure(t, projectDir, "shared", "apply")
		assertOutputContains(t, stderr, "conflicts detected")
	})
}

// TestStatusShowsSharedInfo tests that bt status shows shared file status
func TestStatusShowsSharedInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "status-shared")

	// Clone a repository
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "my-project")
	projectDir := filepath.Join(tempDir, "my-project")

	// Add shared file
	defaultDir := getDefaultBranchDir(projectDir)
	writeFile(t, filepath.Join(defaultDir, ".env"), "SECRET=value")
	runBtSuccess(t, projectDir, "shared", "add", ".env", "--type", "symlink")

	// Add worktree
	runBtSuccess(t, projectDir, "add", "-b", "feature/status")

	t.Run("status shows shared files with applied status", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "status")
		assertOutputContains(t, stdout, "Shared files:")
		assertOutputContains(t, stdout, ".env")
		assertOutputContains(t, stdout, "applied")
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
