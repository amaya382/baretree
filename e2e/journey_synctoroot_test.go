package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

// TestJourneySyncToRoot tests sync-to-root functionality
func TestJourneySyncToRoot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "synctoroot")

	// Clone
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "synctoroot-test")
	projectDir := filepath.Join(tempDir, "synctoroot-test")

	// Find the default branch worktree (main or master)
	var mainWorktree string
	var defaultBranch string
	if isDirectory(filepath.Join(projectDir, "main")) {
		mainWorktree = filepath.Join(projectDir, "main")
		defaultBranch = "main"
	} else if isDirectory(filepath.Join(projectDir, "master")) {
		mainWorktree = filepath.Join(projectDir, "master")
		defaultBranch = "master"
	} else {
		t.Fatal("could not find main or master worktree")
	}

	// Create test files in main worktree
	t.Run("setup test files", func(t *testing.T) {
		// Create CLAUDE.md file
		claudeMdPath := filepath.Join(mainWorktree, "CLAUDE.md")
		err := os.WriteFile(claudeMdPath, []byte("# Test CLAUDE.md\n"), 0644)
		if err != nil {
			t.Fatalf("failed to write CLAUDE.md: %v", err)
		}

		// Create .claude directory
		claudeDir := filepath.Join(mainWorktree, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatalf("failed to create .claude directory: %v", err)
		}

		// Create a file inside .claude directory
		rulesPath := filepath.Join(claudeDir, "rules.md")
		if err := os.WriteFile(rulesPath, []byte("# Rules\n"), 0644); err != nil {
			t.Fatalf("failed to write rules.md: %v", err)
		}

		// Create nested file for custom target test
		docsDir := filepath.Join(mainWorktree, "docs")
		if err := os.MkdirAll(docsDir, 0755); err != nil {
			t.Fatalf("failed to create docs directory: %v", err)
		}
		guidePath := filepath.Join(docsDir, "guide.md")
		if err := os.WriteFile(guidePath, []byte("# Guide\n"), 0644); err != nil {
			t.Fatalf("failed to write guide.md: %v", err)
		}
	})

	// Test adding sync-to-root for a file
	t.Run("add file sync-to-root", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "sync-to-root", "add", "CLAUDE.md")

		assertOutputContains(t, stdout, "Adding sync-to-root")
		assertOutputContains(t, stdout, "CLAUDE.md")

		// Check symlink was created
		symlinkPath := filepath.Join(projectDir, "CLAUDE.md")
		assertFileExists(t, symlinkPath)
		assertIsSymlink(t, symlinkPath)

		// Verify content is accessible
		content, err := os.ReadFile(symlinkPath)
		if err != nil {
			t.Fatalf("failed to read symlinked CLAUDE.md: %v", err)
		}
		if string(content) != "# Test CLAUDE.md\n" {
			t.Errorf("unexpected content: %s", string(content))
		}
	})

	// Test adding sync-to-root for a directory
	t.Run("add directory sync-to-root", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "sync-to-root", "add", ".claude")

		assertOutputContains(t, stdout, "Adding sync-to-root")
		assertOutputContains(t, stdout, ".claude")

		// Check symlink was created
		symlinkPath := filepath.Join(projectDir, ".claude")
		assertFileExists(t, symlinkPath)
		assertIsSymlink(t, symlinkPath)

		// Verify content inside directory is accessible
		rulesPath := filepath.Join(symlinkPath, "rules.md")
		content, err := os.ReadFile(rulesPath)
		if err != nil {
			t.Fatalf("failed to read rules.md via symlinked directory: %v", err)
		}
		if string(content) != "# Rules\n" {
			t.Errorf("unexpected content: %s", string(content))
		}
	})

	// Test adding sync-to-root with custom target
	t.Run("add sync-to-root with custom target", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "sync-to-root", "add", "docs/guide.md", "guide.md")

		assertOutputContains(t, stdout, "Adding sync-to-root")
		assertOutputContains(t, stdout, "docs/guide.md")
		assertOutputContains(t, stdout, "guide.md")

		// Check symlink was created at custom target
		symlinkPath := filepath.Join(projectDir, "guide.md")
		assertFileExists(t, symlinkPath)
		assertIsSymlink(t, symlinkPath)

		// Verify content
		content, err := os.ReadFile(symlinkPath)
		if err != nil {
			t.Fatalf("failed to read guide.md via symlink: %v", err)
		}
		if string(content) != "# Guide\n" {
			t.Errorf("unexpected content: %s", string(content))
		}
	})

	// Test list command
	t.Run("list sync-to-root entries", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "sync-to-root", "list")

		assertOutputContains(t, stdout, "Sync-to-root entries")
		assertOutputContains(t, stdout, "CLAUDE.md")
		assertOutputContains(t, stdout, ".claude")
		assertOutputContains(t, stdout, "docs/guide.md")
		assertOutputContains(t, stdout, "[OK]")
	})

	// Test remove command
	t.Run("remove sync-to-root entry", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "sync-to-root", "remove", "CLAUDE.md")

		assertOutputContains(t, stdout, "Removing sync-to-root")
		assertOutputContains(t, stdout, "removed")

		// Symlink should be gone
		symlinkPath := filepath.Join(projectDir, "CLAUDE.md")
		assertFileNotExists(t, symlinkPath)

		// Original file should still exist
		assertFileExists(t, filepath.Join(mainWorktree, "CLAUDE.md"))
	})

	// Test apply command after remove
	t.Run("apply recreates missing symlinks", func(t *testing.T) {
		// Add CLAUDE.md back
		runBtSuccess(t, projectDir, "sync-to-root", "add", "CLAUDE.md")

		// Manually remove the symlink
		symlinkPath := filepath.Join(projectDir, "CLAUDE.md")
		os.Remove(symlinkPath)
		assertFileNotExists(t, symlinkPath)

		// Run apply
		stdout := runBtSuccess(t, projectDir, "sync-to-root", "apply")

		assertOutputContains(t, stdout, "Applying sync-to-root")
		assertOutputContains(t, stdout, "CLAUDE.md")
		assertOutputContains(t, stdout, defaultBranch)

		// Symlink should be recreated
		assertFileExists(t, symlinkPath)
		assertIsSymlink(t, symlinkPath)
	})

	// Test that status shows sync-to-root
	t.Run("status shows sync-to-root", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "status")

		assertOutputContains(t, stdout, "Sync-to-root")
	})
}

// TestSyncToRootErrors tests error cases for sync-to-root
func TestSyncToRootErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "synctoroot-errors")

	// Clone
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "error-test")
	projectDir := filepath.Join(tempDir, "error-test")

	// Find main worktree
	var mainWorktree string
	if isDirectory(filepath.Join(projectDir, "main")) {
		mainWorktree = filepath.Join(projectDir, "main")
	} else {
		mainWorktree = filepath.Join(projectDir, "master")
	}

	t.Run("error on non-existent source", func(t *testing.T) {
		_, stderr := runBtFailure(t, projectDir, "sync-to-root", "add", "nonexistent.md")

		assertOutputContains(t, stderr, "source does not exist")
	})

	t.Run("error on existing non-symlink target", func(t *testing.T) {
		// Create test file in main worktree
		testFile := filepath.Join(mainWorktree, "test.md")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		// Create a regular file in repo root with same name
		rootFile := filepath.Join(projectDir, "test.md")
		if err := os.WriteFile(rootFile, []byte("conflicting"), 0644); err != nil {
			t.Fatalf("failed to write root file: %v", err)
		}

		_, stderr := runBtFailure(t, projectDir, "sync-to-root", "add", "test.md")

		assertOutputContains(t, stderr, "already exists")
		assertOutputContains(t, stderr, "not a symlink")
	})

	t.Run("error on duplicate entry", func(t *testing.T) {
		// Create test file
		testFile2 := filepath.Join(mainWorktree, "test2.md")
		if err := os.WriteFile(testFile2, []byte("test2"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		// Add it
		runBtSuccess(t, projectDir, "sync-to-root", "add", "test2.md")

		// Try to add again
		_, stderr := runBtFailure(t, projectDir, "sync-to-root", "add", "test2.md")

		assertOutputContains(t, stderr, "already configured")
	})
}

// TestSyncToRootForce tests the --force flag
func TestSyncToRootForce(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "synctoroot-force")

	// Clone
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "force-test")
	projectDir := filepath.Join(tempDir, "force-test")

	// Find main worktree
	var mainWorktree string
	if isDirectory(filepath.Join(projectDir, "main")) {
		mainWorktree = filepath.Join(projectDir, "main")
	} else {
		mainWorktree = filepath.Join(projectDir, "master")
	}

	t.Run("force overwrites wrong symlink", func(t *testing.T) {
		// Create test file in main worktree
		testFile := filepath.Join(mainWorktree, "force-test.md")
		if err := os.WriteFile(testFile, []byte("correct content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		// Create a symlink pointing to wrong location
		wrongTarget := filepath.Join(projectDir, "force-test.md")
		if err := os.Symlink("/tmp/wrong-target", wrongTarget); err != nil {
			t.Fatalf("failed to create wrong symlink: %v", err)
		}

		// Without force, should fail
		_, stderr := runBtFailure(t, projectDir, "sync-to-root", "add", "force-test.md")
		assertOutputContains(t, stderr, "wrong location")

		// With force, should succeed
		runBtSuccess(t, projectDir, "sync-to-root", "add", "force-test.md", "--force")

		// Verify symlink now points to correct location
		content, err := os.ReadFile(wrongTarget)
		if err != nil {
			t.Fatalf("failed to read via symlink: %v", err)
		}
		if string(content) != "correct content" {
			t.Errorf("unexpected content: %s", string(content))
		}
	})
}
