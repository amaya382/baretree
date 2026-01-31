package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestMigrate_InPlace tests in-place migration with --in-place flag
func TestMigrate_InPlace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-inplace")

	// Setup: create a regular git repository
	repoDir := filepath.Join(tempDir, "test-repo")
	setupGitRepo(t, repoDir)

	t.Run("migrate in-place", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "repo", "migrate", ".", "-i")

		assertOutputContains(t, stdout, "Migration successful")

		// Check baretree structure was created
		assertFileExists(t, filepath.Join(repoDir, ".bare"))
		assertFileExists(t, filepath.Join(repoDir, ".bare"))
		assertFileExists(t, filepath.Join(repoDir, "master"))

		// Original .git should be gone
		assertFileNotExists(t, filepath.Join(repoDir, ".git"))
	})

	t.Run("worktree is functional", func(t *testing.T) {
		worktreeDir := filepath.Join(repoDir, "master")
		stdout := runGitSuccess(t, worktreeDir, "status")
		assertOutputContains(t, stdout, "On branch master")
	})
}

// TestMigrate_Destination tests migration with --destination flag
func TestMigrate_Destination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-dest")

	// Setup: create a regular git repository
	repoDir := filepath.Join(tempDir, "source-repo")
	destDir := filepath.Join(tempDir, "dest-repo")
	setupGitRepo(t, repoDir)

	t.Run("migrate to destination", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "repo", "migrate", ".", "-d", destDir)

		assertOutputContains(t, stdout, "Migration successful")

		// Check destination has baretree structure
		assertFileExists(t, filepath.Join(destDir, ".bare"))
		assertFileExists(t, filepath.Join(destDir, ".bare"))
		assertFileExists(t, filepath.Join(destDir, "master"))

		// Original repository should still exist
		assertFileExists(t, filepath.Join(repoDir, ".git"))
	})

	t.Run("destination worktree is functional", func(t *testing.T) {
		worktreeDir := filepath.Join(destDir, "master")
		stdout := runGitSuccess(t, worktreeDir, "status")
		assertOutputContains(t, stdout, "On branch master")
	})
}

// TestMigrate_PreservesWorkingTreeState tests that unstaged, staged, and untracked files are preserved
func TestMigrate_PreservesWorkingTreeState(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-state")

	// Setup: create a regular git repository with working tree state
	repoDir := filepath.Join(tempDir, "test-repo")
	setupGitRepo(t, repoDir)

	// Create working tree state
	// 1. Unstaged changes
	writeFile(t, filepath.Join(repoDir, "file1.txt"), "modified content")

	// 2. Staged new file
	writeFile(t, filepath.Join(repoDir, "staged.txt"), "new staged file")
	runGitSuccess(t, repoDir, "add", "staged.txt")

	// 3. Staged file with additional unstaged changes
	writeFile(t, filepath.Join(repoDir, "staged.txt"), "staged file with more changes")

	// 4. Untracked file
	writeFile(t, filepath.Join(repoDir, "untracked.txt"), "untracked file")

	// Get original git status
	originalStatus := runGitSuccess(t, repoDir, "status", "--porcelain")

	t.Run("migrate preserves state with -d", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "migrated-repo")
		runBtSuccess(t, repoDir, "repo", "migrate", ".", "-d", destDir)

		// Check git status in migrated worktree
		worktreeDir := filepath.Join(destDir, "master")
		newStatus := runGitSuccess(t, worktreeDir, "status", "--porcelain")

		if originalStatus != newStatus {
			t.Errorf("working tree state not preserved\noriginal:\n%s\nmigrated:\n%s", originalStatus, newStatus)
		}

		// Verify file contents
		assertFileContent(t, filepath.Join(worktreeDir, "file1.txt"), "modified content")
		assertFileContent(t, filepath.Join(worktreeDir, "staged.txt"), "staged file with more changes")
		assertFileContent(t, filepath.Join(worktreeDir, "untracked.txt"), "untracked file")
	})
}

// TestMigrate_PreservesWorkingTreeState_InPlace tests state preservation for in-place migration
func TestMigrate_PreservesWorkingTreeState_InPlace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-state-inplace")

	// Setup: create a regular git repository with working tree state
	repoDir := filepath.Join(tempDir, "test-repo")
	setupGitRepo(t, repoDir)

	// Create working tree state
	writeFile(t, filepath.Join(repoDir, "file1.txt"), "modified content")
	writeFile(t, filepath.Join(repoDir, "staged.txt"), "new staged file")
	runGitSuccess(t, repoDir, "add", "staged.txt")
	writeFile(t, filepath.Join(repoDir, "untracked.txt"), "untracked file")

	// Get original git status
	originalStatus := runGitSuccess(t, repoDir, "status", "--porcelain")

	t.Run("in-place migrate preserves state", func(t *testing.T) {
		runBtSuccess(t, repoDir, "repo", "migrate", ".", "-i")

		// Check git status in migrated worktree
		worktreeDir := filepath.Join(repoDir, "master")
		newStatus := runGitSuccess(t, worktreeDir, "status", "--porcelain")

		if originalStatus != newStatus {
			t.Errorf("working tree state not preserved\noriginal:\n%s\nmigrated:\n%s", originalStatus, newStatus)
		}
	})
}

// TestMigrate_WithExistingWorktrees tests migration of a repository that already has worktrees
func TestMigrate_WithExistingWorktrees(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-worktrees")

	// Setup: create a git repository with existing worktrees
	repoDir := filepath.Join(tempDir, "repo-with-worktrees")
	setupGitRepo(t, repoDir)

	// Create a branch and worktree using git directly
	runGitSuccess(t, repoDir, "branch", "feature-branch")
	worktreeDir := filepath.Join(tempDir, "feature-worktree")
	runGitSuccess(t, repoDir, "worktree", "add", worktreeDir, "feature-branch")

	// Add a file to the worktree
	writeFile(t, filepath.Join(worktreeDir, "feature.txt"), "feature content")
	runGitSuccess(t, worktreeDir, "add", "feature.txt")
	runGitSuccess(t, worktreeDir, "commit", "-m", "Add feature file")

	// Add unstaged changes to the worktree to verify state preservation
	writeFile(t, filepath.Join(worktreeDir, "feature.txt"), "modified feature content")

	t.Run("migrate with destination automatically converts external worktrees", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "migrated-repo")
		stdout := runBtSuccess(t, repoDir, "repo", "migrate", ".", "-d", destDir)

		assertOutputContains(t, stdout, "Migration successful")
		assertOutputContains(t, stdout, "External worktrees to migrate: 1")

		// Check baretree structure
		assertFileExists(t, filepath.Join(destDir, ".bare"))
		assertFileExists(t, filepath.Join(destDir, "master"))

		// The feature-branch worktree should be automatically migrated
		assertFileExists(t, filepath.Join(destDir, "feature-branch"))

		// The committed file should be in the migrated worktree
		assertFileExists(t, filepath.Join(destDir, "feature-branch", "feature.txt"))
		assertFileContent(t, filepath.Join(destDir, "feature-branch", "feature.txt"), "modified feature content")

		// Verify the worktree is functional
		stdout = runGitSuccess(t, filepath.Join(destDir, "feature-branch"), "status")
		assertOutputContains(t, stdout, "On branch feature-branch")
	})
}

// TestMigrate_WithExistingWorktrees_InPlace tests in-place migration with existing worktrees
func TestMigrate_WithExistingWorktrees_InPlace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-worktrees-inplace")

	// Setup: create a git repository with existing worktrees
	repoDir := filepath.Join(tempDir, "repo-with-worktrees")
	setupGitRepo(t, repoDir)

	// Create a branch and worktree using git directly
	runGitSuccess(t, repoDir, "branch", "develop")
	worktreeDir := filepath.Join(tempDir, "develop-worktree")
	runGitSuccess(t, repoDir, "worktree", "add", worktreeDir, "develop")

	// Add a file to the worktree
	writeFile(t, filepath.Join(worktreeDir, "develop.txt"), "develop content")
	runGitSuccess(t, worktreeDir, "add", "develop.txt")
	runGitSuccess(t, worktreeDir, "commit", "-m", "Add develop file")

	// Add unstaged changes to verify state preservation
	writeFile(t, filepath.Join(worktreeDir, "develop.txt"), "modified develop content")

	t.Run("in-place migrate automatically converts external worktrees", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "repo", "migrate", ".", "-i")

		assertOutputContains(t, stdout, "Migration successful")
		assertOutputContains(t, stdout, "External worktrees to migrate: 1")

		// Check baretree structure
		assertFileExists(t, filepath.Join(repoDir, ".bare"))
		assertFileExists(t, filepath.Join(repoDir, "master"))

		// The develop worktree should be automatically migrated into baretree structure
		assertFileExists(t, filepath.Join(repoDir, "develop"))

		// The file should be accessible with modified content
		assertFileExists(t, filepath.Join(repoDir, "develop", "develop.txt"))
		assertFileContent(t, filepath.Join(repoDir, "develop", "develop.txt"), "modified develop content")

		// Original external worktree location should be removed
		assertFileNotExists(t, worktreeDir)

		// Verify the worktree is functional
		stdout = runGitSuccess(t, filepath.Join(repoDir, "develop"), "status")
		assertOutputContains(t, stdout, "On branch develop")
	})
}

// TestMigrate_WithMultipleExternalWorktrees tests migration with multiple external worktrees
func TestMigrate_WithMultipleExternalWorktrees(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-multi-worktrees")

	// Setup: create a git repository with multiple external worktrees
	repoDir := filepath.Join(tempDir, "repo-with-worktrees")
	setupGitRepo(t, repoDir)

	// Create multiple branches and worktrees
	runGitSuccess(t, repoDir, "branch", "feature-a")
	runGitSuccess(t, repoDir, "branch", "feature-b")

	worktreeDirA := filepath.Join(tempDir, "worktree-a")
	worktreeDirB := filepath.Join(tempDir, "worktree-b")
	runGitSuccess(t, repoDir, "worktree", "add", worktreeDirA, "feature-a")
	runGitSuccess(t, repoDir, "worktree", "add", worktreeDirB, "feature-b")

	// Add files to each worktree
	writeFile(t, filepath.Join(worktreeDirA, "feature-a.txt"), "feature A content")
	runGitSuccess(t, worktreeDirA, "add", "feature-a.txt")
	runGitSuccess(t, worktreeDirA, "commit", "-m", "Add feature A file")

	writeFile(t, filepath.Join(worktreeDirB, "feature-b.txt"), "feature B content")
	runGitSuccess(t, worktreeDirB, "add", "feature-b.txt")
	runGitSuccess(t, worktreeDirB, "commit", "-m", "Add feature B file")

	t.Run("in-place migrate converts all external worktrees", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "repo", "migrate", ".", "-i")

		assertOutputContains(t, stdout, "Migration successful")
		assertOutputContains(t, stdout, "External worktrees to migrate: 2")

		// Check baretree structure
		assertFileExists(t, filepath.Join(repoDir, ".bare"))
		assertFileExists(t, filepath.Join(repoDir, "master"))

		// Both feature worktrees should be automatically migrated
		assertFileExists(t, filepath.Join(repoDir, "feature-a"))
		assertFileExists(t, filepath.Join(repoDir, "feature-b"))

		// Files should be in place
		assertFileContent(t, filepath.Join(repoDir, "feature-a", "feature-a.txt"), "feature A content")
		assertFileContent(t, filepath.Join(repoDir, "feature-b", "feature-b.txt"), "feature B content")

		// Original worktree locations should be removed
		assertFileNotExists(t, worktreeDirA)
		assertFileNotExists(t, worktreeDirB)

		// Verify all worktrees are functional
		stdout = runGitSuccess(t, filepath.Join(repoDir, "feature-a"), "status")
		assertOutputContains(t, stdout, "On branch feature-a")

		stdout = runGitSuccess(t, filepath.Join(repoDir, "feature-b"), "status")
		assertOutputContains(t, stdout, "On branch feature-b")
	})
}

// TestMigrate_WithHierarchicalBranchWorktree tests migration with worktrees using hierarchical branch names
func TestMigrate_WithHierarchicalBranchWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-hier-worktree")

	// Setup
	repoDir := filepath.Join(tempDir, "repo")
	setupGitRepo(t, repoDir)

	// Create a hierarchical branch name
	runGitSuccess(t, repoDir, "branch", "feature/new-feature")
	worktreeDir := filepath.Join(tempDir, "feature-worktree")
	runGitSuccess(t, repoDir, "worktree", "add", worktreeDir, "feature/new-feature")

	writeFile(t, filepath.Join(worktreeDir, "feature.txt"), "new feature content")
	runGitSuccess(t, worktreeDir, "add", "feature.txt")
	runGitSuccess(t, worktreeDir, "commit", "-m", "Add feature file")

	t.Run("in-place migrate handles hierarchical branch names", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "repo", "migrate", ".", "-i")

		assertOutputContains(t, stdout, "Migration successful")

		// Check baretree structure
		assertFileExists(t, filepath.Join(repoDir, ".bare"))
		assertFileExists(t, filepath.Join(repoDir, "master"))

		// The hierarchical worktree should be migrated with proper directory structure
		assertFileExists(t, filepath.Join(repoDir, "feature", "new-feature"))
		assertFileContent(t, filepath.Join(repoDir, "feature", "new-feature", "feature.txt"), "new feature content")

		// Original worktree location should be removed
		assertFileNotExists(t, worktreeDir)

		// Verify the worktree is functional
		stdout = runGitSuccess(t, filepath.Join(repoDir, "feature", "new-feature"), "status")
		assertOutputContains(t, stdout, "On branch feature/new-feature")
	})
}

// TestMigrate_RequiresFlag tests that either -i or -d is required
func TestMigrate_RequiresFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-noflag")
	repoDir := filepath.Join(tempDir, "test-repo")
	setupGitRepo(t, repoDir)

	t.Run("fails without flag", func(t *testing.T) {
		_, stderr := runBtFailure(t, repoDir, "repo", "migrate", ".")
		assertOutputContains(t, stderr, "--in-place")
		assertOutputContains(t, stderr, "--destination")
	})
}

// TestMigrate_ToRoot tests migration with --to-root flag
func TestMigrate_ToRoot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-toroot")

	// Setup: create a regular git repository with a remote
	repoDir := filepath.Join(tempDir, "source-repo")
	baretreeRoot := filepath.Join(tempDir, "baretree-root")
	setupGitRepoWithRemote(t, repoDir, "git@github.com:testuser/testrepo.git")

	t.Run("migrate to root", func(t *testing.T) {
		env := map[string]string{
			"BARETREE_ROOT": baretreeRoot,
		}
		stdout, _, err := runBtWithEnv(t, repoDir, env, "repo", "migrate", ".", "-r")
		if err != nil {
			t.Fatalf("migrate to root failed: %v", err)
		}

		assertOutputContains(t, stdout, "github.com/testuser/testrepo")

		// Check destination has baretree structure
		destDir := filepath.Join(baretreeRoot, "github.com", "testuser", "testrepo")
		assertFileExists(t, filepath.Join(destDir, ".bare"))
		assertFileExists(t, filepath.Join(destDir, "master"))

		// Original repository should be removed
		assertFileNotExists(t, repoDir)
	})

	t.Run("destination worktree is functional", func(t *testing.T) {
		worktreeDir := filepath.Join(baretreeRoot, "github.com", "testuser", "testrepo", "master")
		stdout := runGitSuccess(t, worktreeDir, "status")
		assertOutputContains(t, stdout, "On branch master")
	})
}

// TestMigrate_ToRoot_WithPath tests migration with --to-root and --path flags
func TestMigrate_ToRoot_WithPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-toroot-path")

	// Setup: create a regular git repository without a remote
	repoDir := filepath.Join(tempDir, "source-repo")
	baretreeRoot := filepath.Join(tempDir, "baretree-root")
	setupGitRepo(t, repoDir)

	t.Run("migrate to root with explicit path", func(t *testing.T) {
		env := map[string]string{
			"BARETREE_ROOT": baretreeRoot,
		}
		stdout, _, err := runBtWithEnv(t, repoDir, env, "repo", "migrate", ".", "-r", "--path", "gitlab.com/myorg/myproject")
		if err != nil {
			t.Fatalf("migrate to root with path failed: %v", err)
		}

		assertOutputContains(t, stdout, "gitlab.com/myorg/myproject")

		// Check destination has baretree structure
		destDir := filepath.Join(baretreeRoot, "gitlab.com", "myorg", "myproject")
		assertFileExists(t, filepath.Join(destDir, ".bare"))
		assertFileExists(t, filepath.Join(destDir, "master"))

		// Original repository should be removed
		assertFileNotExists(t, repoDir)
	})
}

// TestMigrate_ToRoot_ExistingBaretree tests migration of existing baretree repository with --to-root
func TestMigrate_ToRoot_ExistingBaretree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-toroot-baretree")

	// Setup: create a baretree repository
	sourceDir := filepath.Join(tempDir, "source-baretree")
	baretreeRoot := filepath.Join(tempDir, "baretree-root")

	// First create a regular git repo and migrate it in-place
	setupGitRepoWithRemote(t, sourceDir, "git@github.com:testuser/existing-baretree.git")
	runBtSuccess(t, sourceDir, "repo", "migrate", ".", "-i")

	// Verify it's a baretree repo now
	assertFileExists(t, filepath.Join(sourceDir, ".bare"))
	assertFileExists(t, filepath.Join(sourceDir, "master"))

	t.Run("migrate existing baretree to root", func(t *testing.T) {
		env := map[string]string{
			"BARETREE_ROOT": baretreeRoot,
		}
		stdout, _, err := runBtWithEnv(t, sourceDir, env, "repo", "migrate", ".", "-r")
		if err != nil {
			t.Fatalf("migrate existing baretree to root failed: %v", err)
		}

		assertOutputContains(t, stdout, "moved successfully")

		// Check destination has baretree structure
		destDir := filepath.Join(baretreeRoot, "github.com", "testuser", "existing-baretree")
		assertFileExists(t, filepath.Join(destDir, ".bare"))
		assertFileExists(t, filepath.Join(destDir, "master"))

		// Original location should be removed
		assertFileNotExists(t, sourceDir)
	})

	t.Run("moved worktree is functional", func(t *testing.T) {
		worktreeDir := filepath.Join(baretreeRoot, "github.com", "testuser", "existing-baretree", "master")
		stdout := runGitSuccess(t, worktreeDir, "status")
		assertOutputContains(t, stdout, "On branch master")
	})
}

// TestMigrate_ToRoot_PreservesState tests that --to-root preserves working tree state
func TestMigrate_ToRoot_PreservesState(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-toroot-state")

	// Setup: create a regular git repository with working tree state
	repoDir := filepath.Join(tempDir, "source-repo")
	baretreeRoot := filepath.Join(tempDir, "baretree-root")
	setupGitRepoWithRemote(t, repoDir, "git@github.com:testuser/staterepo.git")

	// Create working tree state
	writeFile(t, filepath.Join(repoDir, "file1.txt"), "modified content")
	writeFile(t, filepath.Join(repoDir, "staged.txt"), "new staged file")
	runGitSuccess(t, repoDir, "add", "staged.txt")
	writeFile(t, filepath.Join(repoDir, "untracked.txt"), "untracked file")

	// Get original git status
	originalStatus := runGitSuccess(t, repoDir, "status", "--porcelain")

	t.Run("migrate to root preserves state", func(t *testing.T) {
		env := map[string]string{
			"BARETREE_ROOT": baretreeRoot,
		}
		_, _, err := runBtWithEnv(t, repoDir, env, "repo", "migrate", ".", "-r")
		if err != nil {
			t.Fatalf("migrate to root failed: %v", err)
		}

		// Check git status in migrated worktree
		worktreeDir := filepath.Join(baretreeRoot, "github.com", "testuser", "staterepo", "master")
		newStatus := runGitSuccess(t, worktreeDir, "status", "--porcelain")

		if originalStatus != newStatus {
			t.Errorf("working tree state not preserved\noriginal:\n%s\nmigrated:\n%s", originalStatus, newStatus)
		}

		// Verify file contents
		assertFileContent(t, filepath.Join(worktreeDir, "file1.txt"), "modified content")
		assertFileContent(t, filepath.Join(worktreeDir, "staged.txt"), "new staged file")
		assertFileContent(t, filepath.Join(worktreeDir, "untracked.txt"), "untracked file")
	})
}

// TestMigrate_ToRoot_RequiresRemoteOrPath tests that --to-root fails without remote or --path
func TestMigrate_ToRoot_RequiresRemoteOrPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-toroot-noremote")
	repoDir := filepath.Join(tempDir, "test-repo")
	baretreeRoot := filepath.Join(tempDir, "baretree-root")
	setupGitRepo(t, repoDir) // No remote

	t.Run("fails without remote or path", func(t *testing.T) {
		env := map[string]string{
			"BARETREE_ROOT": baretreeRoot,
		}
		_, stderr, err := runBtWithEnv(t, repoDir, env, "repo", "migrate", ".", "-r")
		if err == nil {
			t.Fatal("expected error but got success")
		}
		assertOutputContains(t, stderr, "no git remotes")
		assertOutputContains(t, stderr, "--path")
	})
}

// TestMigrate_ToRoot_DestinationExists tests that --to-root fails if destination exists
func TestMigrate_ToRoot_DestinationExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-toroot-exists")
	repoDir := filepath.Join(tempDir, "source-repo")
	baretreeRoot := filepath.Join(tempDir, "baretree-root")
	setupGitRepoWithRemote(t, repoDir, "git@github.com:testuser/existingrepo.git")

	// Pre-create the destination directory
	destDir := filepath.Join(baretreeRoot, "github.com", "testuser", "existingrepo")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("failed to create destination: %v", err)
	}

	t.Run("fails if destination exists", func(t *testing.T) {
		env := map[string]string{
			"BARETREE_ROOT": baretreeRoot,
		}
		_, stderr, err := runBtWithEnv(t, repoDir, env, "repo", "migrate", ".", "-r")
		if err == nil {
			t.Fatal("expected error but got success")
		}
		assertOutputContains(t, stderr, "destination already exists")
	})
}

// Helper functions

// setupGitRepo creates a basic git repository
func setupGitRepo(t *testing.T, dir string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	runGitSuccess(t, dir, "init")
	writeFile(t, filepath.Join(dir, "file1.txt"), "initial content")
	runGitSuccess(t, dir, "add", "file1.txt")
	runGitSuccess(t, dir, "commit", "-m", "Initial commit")
}

// setupGitRepoWithRemote creates a git repository with a remote
func setupGitRepoWithRemote(t *testing.T, dir string, remoteURL string) {
	t.Helper()

	setupGitRepo(t, dir)
	runGitSuccess(t, dir, "remote", "add", "origin", remoteURL)
}

// writeFile writes content to a file
func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

// assertFileContent checks that a file has the expected content
func assertFileContent(t *testing.T, path string, expected string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("failed to read file %s: %v", path, err)
		return
	}

	if string(content) != expected {
		t.Errorf("file %s content mismatch\nexpected: %q\ngot: %q", path, expected, string(content))
	}
}

// runBtWithEnv runs bt with custom environment variables
func runBtWithEnv(t *testing.T, workDir string, env map[string]string, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	cmd := exec.Command(btBinary, args...)
	cmd.Dir = workDir

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// TestMigrate_PreservesDeletedFiles tests that deleted files (staged and unstaged) are preserved
func TestMigrate_PreservesDeletedFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-deleted")

	// Setup: create a git repository with multiple files
	repoDir := filepath.Join(tempDir, "test-repo")
	setupGitRepo(t, repoDir)

	// Add more files to delete later
	writeFile(t, filepath.Join(repoDir, "to-delete-unstaged.txt"), "will be deleted unstaged")
	writeFile(t, filepath.Join(repoDir, "to-delete-staged.txt"), "will be deleted staged")
	runGitSuccess(t, repoDir, "add", ".")
	runGitSuccess(t, repoDir, "commit", "-m", "Add files to delete")

	// Delete files
	os.Remove(filepath.Join(repoDir, "to-delete-unstaged.txt")) // unstaged deletion
	runGitSuccess(t, repoDir, "rm", "to-delete-staged.txt")     // staged deletion

	// Get original git status
	originalStatus := runGitSuccess(t, repoDir, "status", "--porcelain")

	t.Run("migrate preserves deleted files state with -d", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "migrated-repo")
		runBtSuccess(t, repoDir, "repo", "migrate", ".", "-d", destDir)

		// Check git status in migrated worktree
		worktreeDir := filepath.Join(destDir, "master")
		newStatus := runGitSuccess(t, worktreeDir, "status", "--porcelain")

		if originalStatus != newStatus {
			t.Errorf("deleted files state not preserved\noriginal:\n%s\nmigrated:\n%s", originalStatus, newStatus)
		}

		// Verify deleted files don't exist
		assertFileNotExists(t, filepath.Join(worktreeDir, "to-delete-unstaged.txt"))
		assertFileNotExists(t, filepath.Join(worktreeDir, "to-delete-staged.txt"))
	})
}

// TestMigrate_PreservesDeletedFiles_InPlace tests deleted files preservation for in-place migration
func TestMigrate_PreservesDeletedFiles_InPlace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-deleted-inplace")

	// Setup
	repoDir := filepath.Join(tempDir, "test-repo")
	setupGitRepo(t, repoDir)

	writeFile(t, filepath.Join(repoDir, "to-delete.txt"), "will be deleted")
	runGitSuccess(t, repoDir, "add", ".")
	runGitSuccess(t, repoDir, "commit", "-m", "Add file to delete")

	// Delete file (unstaged)
	os.Remove(filepath.Join(repoDir, "to-delete.txt"))

	originalStatus := runGitSuccess(t, repoDir, "status", "--porcelain")

	t.Run("in-place migrate preserves deleted files state", func(t *testing.T) {
		runBtSuccess(t, repoDir, "repo", "migrate", ".", "-i")

		worktreeDir := filepath.Join(repoDir, "master")
		newStatus := runGitSuccess(t, worktreeDir, "status", "--porcelain")

		if originalStatus != newStatus {
			t.Errorf("deleted files state not preserved\noriginal:\n%s\nmigrated:\n%s", originalStatus, newStatus)
		}
	})
}

// TestMigrate_PreservesSymlinks tests that symlinks are preserved
func TestMigrate_PreservesSymlinks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-symlinks")

	// Setup
	repoDir := filepath.Join(tempDir, "test-repo")
	setupGitRepo(t, repoDir)

	// Create a symlink (relative)
	targetFile := filepath.Join(repoDir, "target.txt")
	writeFile(t, targetFile, "target content")
	symlinkPath := filepath.Join(repoDir, "link.txt")
	if err := os.Symlink("target.txt", symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Create a directory and symlink to it
	subDir := filepath.Join(repoDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	writeFile(t, filepath.Join(subDir, "subfile.txt"), "subfile content")
	dirLinkPath := filepath.Join(repoDir, "dirlink")
	if err := os.Symlink("subdir", dirLinkPath); err != nil {
		t.Fatalf("failed to create dir symlink: %v", err)
	}

	runGitSuccess(t, repoDir, "add", ".")
	runGitSuccess(t, repoDir, "commit", "-m", "Add symlinks")

	t.Run("migrate preserves symlinks with -d", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "migrated-repo")
		runBtSuccess(t, repoDir, "repo", "migrate", ".", "-d", destDir)

		worktreeDir := filepath.Join(destDir, "master")

		// Check symlink to file
		assertIsSymlink(t, filepath.Join(worktreeDir, "link.txt"))
		linkTarget, err := os.Readlink(filepath.Join(worktreeDir, "link.txt"))
		if err != nil {
			t.Errorf("failed to read symlink: %v", err)
		} else if linkTarget != "target.txt" {
			t.Errorf("symlink target mismatch: expected 'target.txt', got %q", linkTarget)
		}

		// Check symlink to directory
		assertIsSymlink(t, filepath.Join(worktreeDir, "dirlink"))
	})
}

// TestMigrate_PreservesHiddenFiles tests that hidden files (dotfiles) are preserved
func TestMigrate_PreservesHiddenFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-hidden")

	// Setup
	repoDir := filepath.Join(tempDir, "test-repo")
	setupGitRepo(t, repoDir)

	// Create hidden files
	writeFile(t, filepath.Join(repoDir, ".hidden"), "hidden content")
	writeFile(t, filepath.Join(repoDir, ".dotfile"), "dotfile content")

	// Create hidden directory with files
	hiddenDir := filepath.Join(repoDir, ".hiddendir")
	if err := os.MkdirAll(hiddenDir, 0755); err != nil {
		t.Fatalf("failed to create hidden dir: %v", err)
	}
	writeFile(t, filepath.Join(hiddenDir, "inside.txt"), "inside hidden dir")

	runGitSuccess(t, repoDir, "add", ".")
	runGitSuccess(t, repoDir, "commit", "-m", "Add hidden files")

	t.Run("migrate preserves hidden files with -d", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "migrated-repo")
		runBtSuccess(t, repoDir, "repo", "migrate", ".", "-d", destDir)

		worktreeDir := filepath.Join(destDir, "master")

		assertFileExists(t, filepath.Join(worktreeDir, ".hidden"))
		assertFileContent(t, filepath.Join(worktreeDir, ".hidden"), "hidden content")
		assertFileExists(t, filepath.Join(worktreeDir, ".dotfile"))
		assertFileExists(t, filepath.Join(worktreeDir, ".hiddendir", "inside.txt"))
	})
}

// TestMigrate_PreservesGitignored tests that .gitignore'd files are preserved
func TestMigrate_PreservesGitignored(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-gitignore")

	// Setup
	repoDir := filepath.Join(tempDir, "test-repo")
	setupGitRepo(t, repoDir)

	// Create .gitignore
	writeFile(t, filepath.Join(repoDir, ".gitignore"), "*.log\n.env\nnode_modules/\n")
	runGitSuccess(t, repoDir, "add", ".gitignore")
	runGitSuccess(t, repoDir, "commit", "-m", "Add .gitignore")

	// Create ignored files
	writeFile(t, filepath.Join(repoDir, "debug.log"), "log content")
	writeFile(t, filepath.Join(repoDir, ".env"), "SECRET=value")
	nodeModules := filepath.Join(repoDir, "node_modules")
	if err := os.MkdirAll(nodeModules, 0755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}
	writeFile(t, filepath.Join(nodeModules, "package.json"), "{}")

	t.Run("migrate preserves gitignored files with -d", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "migrated-repo")
		runBtSuccess(t, repoDir, "repo", "migrate", ".", "-d", destDir)

		worktreeDir := filepath.Join(destDir, "master")

		// Gitignored files should be preserved
		assertFileExists(t, filepath.Join(worktreeDir, "debug.log"))
		assertFileContent(t, filepath.Join(worktreeDir, "debug.log"), "log content")
		assertFileExists(t, filepath.Join(worktreeDir, ".env"))
		assertFileContent(t, filepath.Join(worktreeDir, ".env"), "SECRET=value")
		assertFileExists(t, filepath.Join(worktreeDir, "node_modules", "package.json"))
	})
}

// TestMigrate_PreservesRenamedFiles tests that renamed files (staged) are preserved
func TestMigrate_PreservesRenamedFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-renamed")

	// Setup
	repoDir := filepath.Join(tempDir, "test-repo")
	setupGitRepo(t, repoDir)

	// Create a file to rename
	writeFile(t, filepath.Join(repoDir, "oldname.txt"), "renamed content")
	runGitSuccess(t, repoDir, "add", "oldname.txt")
	runGitSuccess(t, repoDir, "commit", "-m", "Add file to rename")

	// Rename the file (staged)
	runGitSuccess(t, repoDir, "mv", "oldname.txt", "newname.txt")

	originalStatus := runGitSuccess(t, repoDir, "status", "--porcelain")

	t.Run("migrate preserves renamed files with -d", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "migrated-repo")
		runBtSuccess(t, repoDir, "repo", "migrate", ".", "-d", destDir)

		worktreeDir := filepath.Join(destDir, "master")
		newStatus := runGitSuccess(t, worktreeDir, "status", "--porcelain")

		if originalStatus != newStatus {
			t.Errorf("renamed file state not preserved\noriginal:\n%s\nmigrated:\n%s", originalStatus, newStatus)
		}

		// Old name should not exist, new name should exist
		assertFileNotExists(t, filepath.Join(worktreeDir, "oldname.txt"))
		assertFileExists(t, filepath.Join(worktreeDir, "newname.txt"))
		assertFileContent(t, filepath.Join(worktreeDir, "newname.txt"), "renamed content")
	})
}

// TestMigrate_PreservesSubdirectoryFiles tests that files in subdirectories are preserved
func TestMigrate_PreservesSubdirectoryFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-subdir")

	// Setup
	repoDir := filepath.Join(tempDir, "test-repo")
	setupGitRepo(t, repoDir)

	// Create nested directory structure
	deepDir := filepath.Join(repoDir, "level1", "level2", "level3")
	if err := os.MkdirAll(deepDir, 0755); err != nil {
		t.Fatalf("failed to create deep dir: %v", err)
	}
	writeFile(t, filepath.Join(repoDir, "level1", "l1.txt"), "level 1")
	writeFile(t, filepath.Join(repoDir, "level1", "level2", "l2.txt"), "level 2")
	writeFile(t, filepath.Join(deepDir, "l3.txt"), "level 3")

	runGitSuccess(t, repoDir, "add", ".")
	runGitSuccess(t, repoDir, "commit", "-m", "Add nested files")

	// Add modified file in subdirectory (unstaged)
	writeFile(t, filepath.Join(deepDir, "l3.txt"), "modified level 3")

	// Add new untracked file in subdirectory
	writeFile(t, filepath.Join(repoDir, "level1", "level2", "untracked.txt"), "untracked in subdir")

	originalStatus := runGitSuccess(t, repoDir, "status", "--porcelain")

	t.Run("migrate preserves subdirectory files with -d", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "migrated-repo")
		runBtSuccess(t, repoDir, "repo", "migrate", ".", "-d", destDir)

		worktreeDir := filepath.Join(destDir, "master")
		newStatus := runGitSuccess(t, worktreeDir, "status", "--porcelain")

		if originalStatus != newStatus {
			t.Errorf("subdirectory state not preserved\noriginal:\n%s\nmigrated:\n%s", originalStatus, newStatus)
		}

		// Check all files exist with correct content
		assertFileContent(t, filepath.Join(worktreeDir, "level1", "l1.txt"), "level 1")
		assertFileContent(t, filepath.Join(worktreeDir, "level1", "level2", "l2.txt"), "level 2")
		assertFileContent(t, filepath.Join(worktreeDir, "level1", "level2", "level3", "l3.txt"), "modified level 3")
		assertFileContent(t, filepath.Join(worktreeDir, "level1", "level2", "untracked.txt"), "untracked in subdir")
	})
}

// TestMigrate_PreservesEmptyDirectories tests that empty directories are preserved
func TestMigrate_PreservesEmptyDirectories(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-emptydir")

	// Setup
	repoDir := filepath.Join(tempDir, "test-repo")
	setupGitRepo(t, repoDir)

	// Create empty directories
	emptyDir := filepath.Join(repoDir, "empty")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("failed to create empty dir: %v", err)
	}

	nestedEmptyDir := filepath.Join(repoDir, "parent", "child", "empty")
	if err := os.MkdirAll(nestedEmptyDir, 0755); err != nil {
		t.Fatalf("failed to create nested empty dir: %v", err)
	}

	t.Run("migrate preserves empty directories with -d", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "migrated-repo")
		runBtSuccess(t, repoDir, "repo", "migrate", ".", "-d", destDir)

		worktreeDir := filepath.Join(destDir, "master")

		// Empty directories should be preserved
		assertFileExists(t, filepath.Join(worktreeDir, "empty"))
		if !isDirectory(filepath.Join(worktreeDir, "empty")) {
			t.Errorf("expected 'empty' to be a directory")
		}

		assertFileExists(t, filepath.Join(worktreeDir, "parent", "child", "empty"))
		if !isDirectory(filepath.Join(worktreeDir, "parent", "child", "empty")) {
			t.Errorf("expected 'parent/child/empty' to be a directory")
		}
	})
}

// TestMigrate_WithSubmodule tests that git submodules are preserved
func TestMigrate_WithSubmodule(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-submodule")

	// Setup: create a repository to use as submodule
	submoduleRepo := filepath.Join(tempDir, "submodule-repo")
	setupGitRepo(t, submoduleRepo)
	writeFile(t, filepath.Join(submoduleRepo, "submodule-file.txt"), "submodule content")
	runGitSuccess(t, submoduleRepo, "add", ".")
	runGitSuccess(t, submoduleRepo, "commit", "-m", "Add submodule file")

	// Setup: create main repository with submodule
	repoDir := filepath.Join(tempDir, "main-repo")
	setupGitRepo(t, repoDir)

	// Add submodule (use -c to allow file:// protocol for local paths)
	runGitSuccess(t, repoDir, "-c", "protocol.file.allow=always", "submodule", "add", submoduleRepo, "libs/mylib")
	runGitSuccess(t, repoDir, "commit", "-m", "Add submodule")

	t.Run("migrate preserves submodule with -d", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "migrated-repo")
		runBtSuccess(t, repoDir, "repo", "migrate", ".", "-d", destDir)

		worktreeDir := filepath.Join(destDir, "master")

		// Check .gitmodules exists
		assertFileExists(t, filepath.Join(worktreeDir, ".gitmodules"))

		// Check submodule directory exists
		assertFileExists(t, filepath.Join(worktreeDir, "libs", "mylib"))

		// Check submodule content
		assertFileExists(t, filepath.Join(worktreeDir, "libs", "mylib", "submodule-file.txt"))
		assertFileContent(t, filepath.Join(worktreeDir, "libs", "mylib", "submodule-file.txt"), "submodule content")

		// Check .git/modules exists in bare repo
		assertFileExists(t, filepath.Join(destDir, ".bare", "modules"))
	})
}

// TestMigrate_WithSubmodule_InPlace tests in-place migration with submodules
func TestMigrate_WithSubmodule_InPlace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-submodule-inplace")

	// Setup submodule repo
	submoduleRepo := filepath.Join(tempDir, "submodule-repo")
	setupGitRepo(t, submoduleRepo)
	writeFile(t, filepath.Join(submoduleRepo, "sub.txt"), "sub content")
	runGitSuccess(t, submoduleRepo, "add", ".")
	runGitSuccess(t, submoduleRepo, "commit", "-m", "Add sub file")

	// Setup main repo
	repoDir := filepath.Join(tempDir, "main-repo")
	setupGitRepo(t, repoDir)
	// Add submodule (use -c to allow file:// protocol for local paths)
	runGitSuccess(t, repoDir, "-c", "protocol.file.allow=always", "submodule", "add", submoduleRepo, "vendor/lib")
	runGitSuccess(t, repoDir, "commit", "-m", "Add submodule")

	t.Run("in-place migrate preserves submodule", func(t *testing.T) {
		runBtSuccess(t, repoDir, "repo", "migrate", ".", "-i")

		worktreeDir := filepath.Join(repoDir, "master")

		assertFileExists(t, filepath.Join(worktreeDir, ".gitmodules"))
		assertFileExists(t, filepath.Join(worktreeDir, "vendor", "lib", "sub.txt"))
		assertFileContent(t, filepath.Join(worktreeDir, "vendor", "lib", "sub.txt"), "sub content")
		assertFileExists(t, filepath.Join(repoDir, ".bare", "modules"))
	})
}
