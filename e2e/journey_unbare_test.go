package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

// TestUnbare_Basic tests basic unbare functionality
func TestUnbare_Basic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "unbare-basic")

	// Setup: create a baretree repository
	repoDir := filepath.Join(tempDir, "test-repo")
	setupBaretreeRepo(t, repoDir)

	t.Run("unbare creates standalone repository", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "standalone")
		runBtSuccess(t, repoDir, "unbare", "master", destDir)

		// Check it's a regular git repository
		assertFileExists(t, filepath.Join(destDir, ".git"))
		if !isDirectory(filepath.Join(destDir, ".git")) {
			t.Errorf("expected .git to be a directory")
		}

		// Check files exist
		assertFileExists(t, filepath.Join(destDir, "file1.txt"))

		// Check it's functional
		stdout := runGitSuccess(t, destDir, "status")
		assertOutputContains(t, stdout, "On branch master")
	})
}

// TestUnbare_PreservesWorkingTreeState tests that working tree state is preserved
func TestUnbare_PreservesWorkingTreeState(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "unbare-state")

	// Setup
	repoDir := filepath.Join(tempDir, "test-repo")
	setupBaretreeRepo(t, repoDir)

	worktreeDir := filepath.Join(repoDir, "master")

	// Create working tree state
	// 1. Unstaged changes
	writeFile(t, filepath.Join(worktreeDir, "file1.txt"), "modified content")

	// 2. Staged new file
	writeFile(t, filepath.Join(worktreeDir, "staged.txt"), "new staged file")
	runGitSuccess(t, worktreeDir, "add", "staged.txt")

	// 3. Untracked file
	writeFile(t, filepath.Join(worktreeDir, "untracked.txt"), "untracked file")

	// Get original status
	originalStatus := runGitSuccess(t, worktreeDir, "status", "--porcelain")

	t.Run("unbare preserves working tree state", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "standalone")
		runBtSuccess(t, repoDir, "unbare", "master", destDir)

		newStatus := runGitSuccess(t, destDir, "status", "--porcelain")

		if originalStatus != newStatus {
			t.Errorf("working tree state not preserved\noriginal:\n%s\nunbared:\n%s", originalStatus, newStatus)
		}

		// Verify file contents
		assertFileContent(t, filepath.Join(destDir, "file1.txt"), "modified content")
		assertFileContent(t, filepath.Join(destDir, "staged.txt"), "new staged file")
		assertFileContent(t, filepath.Join(destDir, "untracked.txt"), "untracked file")
	})
}

// TestUnbare_PreservesDeletedFiles tests that deleted files state is preserved
func TestUnbare_PreservesDeletedFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "unbare-deleted")

	// Setup
	repoDir := filepath.Join(tempDir, "test-repo")
	setupBaretreeRepo(t, repoDir)

	worktreeDir := filepath.Join(repoDir, "master")

	// Add more files to delete
	writeFile(t, filepath.Join(worktreeDir, "to-delete-unstaged.txt"), "will be deleted unstaged")
	writeFile(t, filepath.Join(worktreeDir, "to-delete-staged.txt"), "will be deleted staged")
	runGitSuccess(t, worktreeDir, "add", ".")
	runGitSuccess(t, worktreeDir, "commit", "-m", "Add files to delete")

	// Delete files
	os.Remove(filepath.Join(worktreeDir, "to-delete-unstaged.txt"))
	runGitSuccess(t, worktreeDir, "rm", "to-delete-staged.txt")

	originalStatus := runGitSuccess(t, worktreeDir, "status", "--porcelain")

	t.Run("unbare preserves deleted files state", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "standalone")
		runBtSuccess(t, repoDir, "unbare", "master", destDir)

		newStatus := runGitSuccess(t, destDir, "status", "--porcelain")

		if originalStatus != newStatus {
			t.Errorf("deleted files state not preserved\noriginal:\n%s\nunbared:\n%s", originalStatus, newStatus)
		}

		// Verify deleted files don't exist
		assertFileNotExists(t, filepath.Join(destDir, "to-delete-unstaged.txt"))
		assertFileNotExists(t, filepath.Join(destDir, "to-delete-staged.txt"))
	})
}

// TestUnbare_PreservesSymlinks tests that symlinks are preserved
func TestUnbare_PreservesSymlinks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "unbare-symlinks")

	// Setup
	repoDir := filepath.Join(tempDir, "test-repo")
	setupBaretreeRepo(t, repoDir)

	worktreeDir := filepath.Join(repoDir, "master")

	// Create symlinks
	writeFile(t, filepath.Join(worktreeDir, "target.txt"), "target content")
	if err := os.Symlink("target.txt", filepath.Join(worktreeDir, "link.txt")); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	runGitSuccess(t, worktreeDir, "add", ".")
	runGitSuccess(t, worktreeDir, "commit", "-m", "Add symlinks")

	t.Run("unbare preserves symlinks", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "standalone")
		runBtSuccess(t, repoDir, "unbare", "master", destDir)

		assertIsSymlink(t, filepath.Join(destDir, "link.txt"))
		linkTarget, err := os.Readlink(filepath.Join(destDir, "link.txt"))
		if err != nil {
			t.Errorf("failed to read symlink: %v", err)
		} else if linkTarget != "target.txt" {
			t.Errorf("symlink target mismatch: expected 'target.txt', got %q", linkTarget)
		}
	})
}

// TestUnbare_PreservesHiddenFiles tests that hidden files are preserved
func TestUnbare_PreservesHiddenFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "unbare-hidden")

	// Setup
	repoDir := filepath.Join(tempDir, "test-repo")
	setupBaretreeRepo(t, repoDir)

	worktreeDir := filepath.Join(repoDir, "master")

	// Create hidden files
	writeFile(t, filepath.Join(worktreeDir, ".hidden"), "hidden content")
	hiddenDir := filepath.Join(worktreeDir, ".hiddendir")
	if err := os.MkdirAll(hiddenDir, 0755); err != nil {
		t.Fatalf("failed to create hidden dir: %v", err)
	}
	writeFile(t, filepath.Join(hiddenDir, "inside.txt"), "inside hidden dir")

	runGitSuccess(t, worktreeDir, "add", ".")
	runGitSuccess(t, worktreeDir, "commit", "-m", "Add hidden files")

	t.Run("unbare preserves hidden files", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "standalone")
		runBtSuccess(t, repoDir, "unbare", "master", destDir)

		assertFileExists(t, filepath.Join(destDir, ".hidden"))
		assertFileContent(t, filepath.Join(destDir, ".hidden"), "hidden content")
		assertFileExists(t, filepath.Join(destDir, ".hiddendir", "inside.txt"))
	})
}

// TestUnbare_PreservesGitignored tests that .gitignore'd files are preserved
func TestUnbare_PreservesGitignored(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "unbare-gitignore")

	// Setup
	repoDir := filepath.Join(tempDir, "test-repo")
	setupBaretreeRepo(t, repoDir)

	worktreeDir := filepath.Join(repoDir, "master")

	// Create .gitignore
	writeFile(t, filepath.Join(worktreeDir, ".gitignore"), "*.log\n.env\nnode_modules/\n")
	runGitSuccess(t, worktreeDir, "add", ".gitignore")
	runGitSuccess(t, worktreeDir, "commit", "-m", "Add .gitignore")

	// Create ignored files
	writeFile(t, filepath.Join(worktreeDir, "debug.log"), "log content")
	writeFile(t, filepath.Join(worktreeDir, ".env"), "SECRET=value")
	nodeModules := filepath.Join(worktreeDir, "node_modules")
	if err := os.MkdirAll(nodeModules, 0755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}
	writeFile(t, filepath.Join(nodeModules, "package.json"), "{}")

	t.Run("unbare preserves gitignored files", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "standalone")
		runBtSuccess(t, repoDir, "unbare", "master", destDir)

		// Gitignored files should be preserved
		assertFileExists(t, filepath.Join(destDir, "debug.log"))
		assertFileContent(t, filepath.Join(destDir, "debug.log"), "log content")
		assertFileExists(t, filepath.Join(destDir, ".env"))
		assertFileContent(t, filepath.Join(destDir, ".env"), "SECRET=value")
		assertFileExists(t, filepath.Join(destDir, "node_modules", "package.json"))
	})
}

// TestUnbare_PreservesRenamedFiles tests that renamed files are preserved
func TestUnbare_PreservesRenamedFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "unbare-renamed")

	// Setup
	repoDir := filepath.Join(tempDir, "test-repo")
	setupBaretreeRepo(t, repoDir)

	worktreeDir := filepath.Join(repoDir, "master")

	// Create file to rename
	writeFile(t, filepath.Join(worktreeDir, "oldname.txt"), "renamed content")
	runGitSuccess(t, worktreeDir, "add", "oldname.txt")
	runGitSuccess(t, worktreeDir, "commit", "-m", "Add file to rename")

	// Rename (staged)
	runGitSuccess(t, worktreeDir, "mv", "oldname.txt", "newname.txt")

	originalStatus := runGitSuccess(t, worktreeDir, "status", "--porcelain")

	t.Run("unbare preserves renamed files", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "standalone")
		runBtSuccess(t, repoDir, "unbare", "master", destDir)

		newStatus := runGitSuccess(t, destDir, "status", "--porcelain")

		if originalStatus != newStatus {
			t.Errorf("renamed file state not preserved\noriginal:\n%s\nunbared:\n%s", originalStatus, newStatus)
		}

		assertFileNotExists(t, filepath.Join(destDir, "oldname.txt"))
		assertFileExists(t, filepath.Join(destDir, "newname.txt"))
		assertFileContent(t, filepath.Join(destDir, "newname.txt"), "renamed content")
	})
}

// TestUnbare_PreservesSubdirectoryFiles tests that files in subdirectories are preserved
func TestUnbare_PreservesSubdirectoryFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "unbare-subdir")

	// Setup
	repoDir := filepath.Join(tempDir, "test-repo")
	setupBaretreeRepo(t, repoDir)

	worktreeDir := filepath.Join(repoDir, "master")

	// Create nested structure
	deepDir := filepath.Join(worktreeDir, "level1", "level2", "level3")
	if err := os.MkdirAll(deepDir, 0755); err != nil {
		t.Fatalf("failed to create deep dir: %v", err)
	}
	writeFile(t, filepath.Join(worktreeDir, "level1", "l1.txt"), "level 1")
	writeFile(t, filepath.Join(deepDir, "l3.txt"), "level 3")

	runGitSuccess(t, worktreeDir, "add", ".")
	runGitSuccess(t, worktreeDir, "commit", "-m", "Add nested files")

	// Add untracked file in subdirectory
	writeFile(t, filepath.Join(worktreeDir, "level1", "untracked.txt"), "untracked in subdir")

	t.Run("unbare preserves subdirectory files", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "standalone")
		runBtSuccess(t, repoDir, "unbare", "master", destDir)

		assertFileContent(t, filepath.Join(destDir, "level1", "l1.txt"), "level 1")
		assertFileContent(t, filepath.Join(destDir, "level1", "level2", "level3", "l3.txt"), "level 3")
		assertFileContent(t, filepath.Join(destDir, "level1", "untracked.txt"), "untracked in subdir")
	})
}

// TestUnbare_PreservesEmptyDirectories tests that empty directories are preserved
func TestUnbare_PreservesEmptyDirectories(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "unbare-emptydir")

	// Setup
	repoDir := filepath.Join(tempDir, "test-repo")
	setupBaretreeRepo(t, repoDir)

	worktreeDir := filepath.Join(repoDir, "master")

	// Create empty directories
	emptyDir := filepath.Join(worktreeDir, "empty")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("failed to create empty dir: %v", err)
	}

	nestedEmptyDir := filepath.Join(worktreeDir, "parent", "child", "empty")
	if err := os.MkdirAll(nestedEmptyDir, 0755); err != nil {
		t.Fatalf("failed to create nested empty dir: %v", err)
	}

	t.Run("unbare preserves empty directories", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "standalone")
		runBtSuccess(t, repoDir, "unbare", "master", destDir)

		assertFileExists(t, filepath.Join(destDir, "empty"))
		if !isDirectory(filepath.Join(destDir, "empty")) {
			t.Errorf("expected 'empty' to be a directory")
		}

		assertFileExists(t, filepath.Join(destDir, "parent", "child", "empty"))
		if !isDirectory(filepath.Join(destDir, "parent", "child", "empty")) {
			t.Errorf("expected 'parent/child/empty' to be a directory")
		}
	})
}

// TestUnbare_WithSubmodule tests that git submodules are preserved
func TestUnbare_WithSubmodule(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "unbare-submodule")

	// Setup submodule repo
	submoduleRepo := filepath.Join(tempDir, "submodule-repo")
	setupGitRepo(t, submoduleRepo)
	writeFile(t, filepath.Join(submoduleRepo, "submodule-file.txt"), "submodule content")
	runGitSuccess(t, submoduleRepo, "add", ".")
	runGitSuccess(t, submoduleRepo, "commit", "-m", "Add submodule file")

	// Setup main baretree repo with submodule
	repoDir := filepath.Join(tempDir, "main-repo")
	setupGitRepo(t, repoDir)

	// Add submodule (use -c to allow file:// protocol for local paths)
	runGitSuccess(t, repoDir, "-c", "protocol.file.allow=always", "submodule", "add", submoduleRepo, "libs/mylib")
	runGitSuccess(t, repoDir, "commit", "-m", "Add submodule")

	// Migrate to baretree
	runBtSuccess(t, repoDir, "repo", "migrate", ".", "-i")

	t.Run("unbare preserves submodule", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "standalone")
		runBtSuccess(t, repoDir, "unbare", "master", destDir)

		// Check .gitmodules exists
		assertFileExists(t, filepath.Join(destDir, ".gitmodules"))

		// Check submodule directory exists
		assertFileExists(t, filepath.Join(destDir, "libs", "mylib"))

		// Check submodule content
		assertFileExists(t, filepath.Join(destDir, "libs", "mylib", "submodule-file.txt"))
		assertFileContent(t, filepath.Join(destDir, "libs", "mylib", "submodule-file.txt"), "submodule content")
	})
}

// TestUnbare_DestinationExists tests that unbare fails if destination exists
func TestUnbare_DestinationExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "unbare-exists")

	// Setup
	repoDir := filepath.Join(tempDir, "test-repo")
	setupBaretreeRepo(t, repoDir)

	destDir := filepath.Join(tempDir, "existing")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}

	t.Run("fails if destination exists", func(t *testing.T) {
		_, stderr := runBtFailure(t, repoDir, "unbare", "master", destDir)
		assertOutputContains(t, stderr, "destination already exists")
	})
}

// Helper function to create a baretree repository
func setupBaretreeRepo(t *testing.T, dir string) {
	t.Helper()

	// First create a regular git repo
	setupGitRepo(t, dir)

	// Migrate to baretree in-place
	runBtSuccess(t, dir, "repo", "migrate", ".", "-i")
}
