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
		// .git should now be a bare repository
		assertFileExists(t, filepath.Join(repoDir, ".git"))
		assertFileExists(t, filepath.Join(repoDir, "master"))

		// Verify .git is now a bare repository
		isBare := runGitSuccess(t, repoDir, "--git-dir=.git", "rev-parse", "--is-bare-repository")
		if !strings.Contains(isBare, "true") {
			t.Errorf("expected .git to be a bare repository, but it isn't")
		}
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
		assertFileExists(t, filepath.Join(destDir, ".git"))
		assertFileExists(t, filepath.Join(destDir, ".git"))
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
		assertFileExists(t, filepath.Join(destDir, ".git"))
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
		assertFileExists(t, filepath.Join(repoDir, ".git"))
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
		assertFileExists(t, filepath.Join(repoDir, ".git"))
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
		assertFileExists(t, filepath.Join(repoDir, ".git"))
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
		assertFileExists(t, filepath.Join(destDir, ".git"))
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
		assertFileExists(t, filepath.Join(destDir, ".git"))
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
	assertFileExists(t, filepath.Join(sourceDir, ".git"))
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
		assertFileExists(t, filepath.Join(destDir, ".git"))
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

// TestMigrate_ToRoot_ExistingBaretreeWithHierarchicalWorktree tests migration of existing baretree with hierarchical worktrees
func TestMigrate_ToRoot_ExistingBaretreeWithHierarchicalWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-toroot-hier")

	// Setup: create a baretree repository with hierarchical worktree
	sourceDir := filepath.Join(tempDir, "source-baretree")
	baretreeRoot := filepath.Join(tempDir, "baretree-root")

	// Create a regular git repo and migrate it in-place
	setupGitRepoWithRemote(t, sourceDir, "git@github.com:testuser/hier-worktree.git")
	runBtSuccess(t, sourceDir, "repo", "migrate", ".", "-i")

	// Add a hierarchical worktree using git directly
	runGitSuccess(t, sourceDir, "--git-dir=.git", "worktree", "add", "feat/auth", "-b", "feat/auth")

	// Add content to the hierarchical worktree
	writeFile(t, filepath.Join(sourceDir, "feat", "auth", "auth.txt"), "auth feature content")
	runGitSuccess(t, filepath.Join(sourceDir, "feat", "auth"), "add", "auth.txt")
	runGitSuccess(t, filepath.Join(sourceDir, "feat", "auth"), "commit", "-m", "Add auth feature")

	// Verify setup
	assertFileExists(t, filepath.Join(sourceDir, "feat", "auth"))
	assertFileExists(t, filepath.Join(sourceDir, ".git", "worktrees", "auth"))

	t.Run("migrate existing baretree with hierarchical worktree to root", func(t *testing.T) {
		env := map[string]string{
			"BARETREE_ROOT": baretreeRoot,
		}
		stdout, _, err := runBtWithEnv(t, sourceDir, env, "repo", "migrate", ".", "-r")
		if err != nil {
			t.Fatalf("migrate existing baretree with hierarchical worktree to root failed: %v", err)
		}

		assertOutputContains(t, stdout, "moved successfully")

		// Check destination has baretree structure
		destDir := filepath.Join(baretreeRoot, "github.com", "testuser", "hier-worktree")
		assertFileExists(t, filepath.Join(destDir, ".git"))
		assertFileExists(t, filepath.Join(destDir, "master"))

		// Hierarchical worktree should exist at correct location
		assertFileExists(t, filepath.Join(destDir, "feat", "auth"))
		assertFileContent(t, filepath.Join(destDir, "feat", "auth", "auth.txt"), "auth feature content")

		// Original location should be removed
		assertFileNotExists(t, sourceDir)
	})

	t.Run("hierarchical worktree is functional after move", func(t *testing.T) {
		worktreeDir := filepath.Join(baretreeRoot, "github.com", "testuser", "hier-worktree", "feat", "auth")
		stdout := runGitSuccess(t, worktreeDir, "status")
		assertOutputContains(t, stdout, "On branch feat/auth")
	})

	t.Run("master worktree is functional after move", func(t *testing.T) {
		worktreeDir := filepath.Join(baretreeRoot, "github.com", "testuser", "hier-worktree", "master")
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
		assertFileExists(t, filepath.Join(destDir, ".git", "modules"))
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
		assertFileExists(t, filepath.Join(repoDir, ".git", "modules"))
	})
}

// TestMigrate_Destination_CustomWorktreeName tests migration with custom worktree directory name
func TestMigrate_Destination_CustomWorktreeName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-custom-wt-name")

	// Setup: create a git repository with custom worktree name (different from branch)
	repoDir := filepath.Join(tempDir, "source-repo")
	setupGitRepo(t, repoDir)

	// Create a branch and worktree with custom name
	runGitSuccess(t, repoDir, "branch", "my-feature")
	worktreeDir := filepath.Join(tempDir, "custom-wt-name")
	runGitSuccess(t, repoDir, "worktree", "add", worktreeDir, "my-feature")

	// Add a file to verify migration
	writeFile(t, filepath.Join(worktreeDir, "feature.txt"), "feature content")
	runGitSuccess(t, worktreeDir, "add", "feature.txt")
	runGitSuccess(t, worktreeDir, "commit", "-m", "Add feature file")

	t.Run("migrate with destination handles custom worktree names", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "dest-repo")
		runBtSuccess(t, repoDir, "repo", "migrate", ".", "-d", destDir)

		// The worktree should be migrated using branch name, not custom directory name
		assertFileExists(t, filepath.Join(destDir, "my-feature"))
		assertFileContent(t, filepath.Join(destDir, "my-feature", "feature.txt"), "feature content")

		// Verify the worktree is functional
		stdout := runGitSuccess(t, filepath.Join(destDir, "my-feature"), "status")
		assertOutputContains(t, stdout, "On branch my-feature")
	})
}

// TestMigrate_ToRoot_WithExternalWorktrees tests --to-root with external worktrees
func TestMigrate_ToRoot_WithExternalWorktrees(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-toroot-external")

	// Setup: create a git repository with external worktrees
	repoDir := filepath.Join(tempDir, "source-repo")
	baretreeRoot := filepath.Join(tempDir, "baretree-root")
	setupGitRepoWithRemote(t, repoDir, "git@github.com:testuser/external-wt.git")

	// Create external worktree
	runGitSuccess(t, repoDir, "branch", "feat/auth")
	worktreeDir := filepath.Join(tempDir, "external-wt")
	runGitSuccess(t, repoDir, "worktree", "add", worktreeDir, "feat/auth")

	writeFile(t, filepath.Join(worktreeDir, "auth.txt"), "auth content")
	runGitSuccess(t, worktreeDir, "add", "auth.txt")
	runGitSuccess(t, worktreeDir, "commit", "-m", "Add auth file")

	t.Run("migrate to root includes external worktrees", func(t *testing.T) {
		env := map[string]string{
			"BARETREE_ROOT": baretreeRoot,
		}
		stdout, _, err := runBtWithEnv(t, repoDir, env, "repo", "migrate", ".", "-r")
		if err != nil {
			t.Fatalf("migrate to root with external worktrees failed: %v", err)
		}

		assertOutputContains(t, stdout, "External worktrees to migrate: 1")

		// Check destination has baretree structure with external worktree
		destDir := filepath.Join(baretreeRoot, "github.com", "testuser", "external-wt")
		assertFileExists(t, filepath.Join(destDir, ".git"))
		assertFileExists(t, filepath.Join(destDir, "master"))
		assertFileExists(t, filepath.Join(destDir, "feat", "auth"))
		assertFileContent(t, filepath.Join(destDir, "feat", "auth", "auth.txt"), "auth content")

		// External worktree location should be removed
		assertFileNotExists(t, worktreeDir)

		// Original repository should be removed
		assertFileNotExists(t, repoDir)
	})

	t.Run("external worktree is functional after move", func(t *testing.T) {
		destDir := filepath.Join(baretreeRoot, "github.com", "testuser", "external-wt")
		stdout := runGitSuccess(t, filepath.Join(destDir, "feat", "auth"), "status")
		assertOutputContains(t, stdout, "On branch feat/auth")
	})
}

// TestMigrate_ToRoot_DeepHierarchicalBranch tests --to-root with deep hierarchical branch names
func TestMigrate_ToRoot_DeepHierarchicalBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-toroot-deep")

	// Setup
	repoDir := filepath.Join(tempDir, "source-repo")
	baretreeRoot := filepath.Join(tempDir, "baretree-root")
	setupGitRepoWithRemote(t, repoDir, "git@github.com:testuser/deep-hier.git")

	// Create deep hierarchical branch worktree
	runGitSuccess(t, repoDir, "branch", "feat/auth/v2/experimental")
	worktreeDir := filepath.Join(tempDir, "deep-wt")
	runGitSuccess(t, repoDir, "worktree", "add", worktreeDir, "feat/auth/v2/experimental")

	writeFile(t, filepath.Join(worktreeDir, "deep.txt"), "deep content")
	runGitSuccess(t, worktreeDir, "add", "deep.txt")
	runGitSuccess(t, worktreeDir, "commit", "-m", "Add deep file")

	t.Run("migrate handles deep hierarchical branch names", func(t *testing.T) {
		env := map[string]string{
			"BARETREE_ROOT": baretreeRoot,
		}
		_, _, err := runBtWithEnv(t, repoDir, env, "repo", "migrate", ".", "-r")
		if err != nil {
			t.Fatalf("migrate with deep hierarchical branch failed: %v", err)
		}

		destDir := filepath.Join(baretreeRoot, "github.com", "testuser", "deep-hier")
		assertFileExists(t, filepath.Join(destDir, "feat", "auth", "v2", "experimental"))
		assertFileContent(t, filepath.Join(destDir, "feat", "auth", "v2", "experimental", "deep.txt"), "deep content")
	})

	t.Run("deep hierarchical worktree is functional", func(t *testing.T) {
		destDir := filepath.Join(baretreeRoot, "github.com", "testuser", "deep-hier")
		stdout := runGitSuccess(t, filepath.Join(destDir, "feat", "auth", "v2", "experimental"), "status")
		assertOutputContains(t, stdout, "On branch feat/auth/v2/experimental")
	})
}

// TestMigrate_ToRoot_MultipleHierarchicalWorktrees tests --to-root with multiple hierarchical worktrees
func TestMigrate_ToRoot_MultipleHierarchicalWorktrees(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-toroot-multi-hier")

	// Setup: create a baretree repository with multiple hierarchical worktrees
	repoDir := filepath.Join(tempDir, "source-repo")
	baretreeRoot := filepath.Join(tempDir, "baretree-root")
	setupGitRepoWithRemote(t, repoDir, "git@github.com:testuser/multi-hier.git")
	runBtSuccess(t, repoDir, "repo", "migrate", ".", "-i")

	// Add multiple hierarchical worktrees
	runGitSuccess(t, repoDir, "--git-dir=.git", "worktree", "add", "feat/auth", "-b", "feat/auth")
	runGitSuccess(t, repoDir, "--git-dir=.git", "worktree", "add", "feat/billing", "-b", "feat/billing")
	runGitSuccess(t, repoDir, "--git-dir=.git", "worktree", "add", "fix/urgent/hotfix", "-b", "fix/urgent/hotfix")

	writeFile(t, filepath.Join(repoDir, "feat", "auth", "auth.txt"), "auth")
	writeFile(t, filepath.Join(repoDir, "feat", "billing", "billing.txt"), "billing")
	writeFile(t, filepath.Join(repoDir, "fix", "urgent", "hotfix", "hotfix.txt"), "hotfix")

	t.Run("migrate moves all hierarchical worktrees", func(t *testing.T) {
		env := map[string]string{
			"BARETREE_ROOT": baretreeRoot,
		}
		stdout, _, err := runBtWithEnv(t, repoDir, env, "repo", "migrate", ".", "-r")
		if err != nil {
			t.Fatalf("migrate with multiple hierarchical worktrees failed: %v", err)
		}

		assertOutputContains(t, stdout, "moved successfully")

		destDir := filepath.Join(baretreeRoot, "github.com", "testuser", "multi-hier")
		assertFileExists(t, filepath.Join(destDir, "feat", "auth", "auth.txt"))
		assertFileExists(t, filepath.Join(destDir, "feat", "billing", "billing.txt"))
		assertFileExists(t, filepath.Join(destDir, "fix", "urgent", "hotfix", "hotfix.txt"))
	})

	t.Run("all worktrees are functional", func(t *testing.T) {
		destDir := filepath.Join(baretreeRoot, "github.com", "testuser", "multi-hier")

		stdout := runGitSuccess(t, filepath.Join(destDir, "feat", "auth"), "status")
		assertOutputContains(t, stdout, "On branch feat/auth")

		stdout = runGitSuccess(t, filepath.Join(destDir, "feat", "billing"), "status")
		assertOutputContains(t, stdout, "On branch feat/billing")

		stdout = runGitSuccess(t, filepath.Join(destDir, "fix", "urgent", "hotfix"), "status")
		assertOutputContains(t, stdout, "On branch fix/urgent/hotfix")
	})
}

// TestMigrate_Destination_DetachedHead tests migration with detached HEAD worktree
func TestMigrate_Destination_DetachedHead(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-detached")

	// Setup
	repoDir := filepath.Join(tempDir, "source-repo")
	setupGitRepo(t, repoDir)

	// Make a second commit
	writeFile(t, filepath.Join(repoDir, "file1.txt"), "modified")
	runGitSuccess(t, repoDir, "add", ".")
	runGitSuccess(t, repoDir, "commit", "-m", "second commit")

	// Get first commit hash and create detached worktree
	firstCommit := runGitSuccess(t, repoDir, "rev-parse", "HEAD~1")
	firstCommit = strings.TrimSpace(firstCommit)
	worktreeDir := filepath.Join(tempDir, "detached-wt")
	runGitSuccess(t, repoDir, "worktree", "add", worktreeDir, firstCommit)

	// Add content to verify migration
	writeFile(t, filepath.Join(worktreeDir, "detached.txt"), "detached content")

	t.Run("migrate with destination handles detached HEAD", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "dest-repo")
		runBtSuccess(t, repoDir, "repo", "migrate", ".", "-d", destDir)

		// Check detached worktree exists
		assertFileExists(t, filepath.Join(destDir, "detached"))
		assertFileContent(t, filepath.Join(destDir, "detached", "detached.txt"), "detached content")

		// Verify the worktree is in detached state
		stdout := runGitSuccess(t, filepath.Join(destDir, "detached"), "status")
		assertOutputContains(t, stdout, "Not currently on any branch")
	})
}

// TestMigrate_ToRoot_PreservesWorkingStateWithHierarchicalWorktree tests state preservation
func TestMigrate_ToRoot_PreservesWorkingStateWithHierarchicalWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-toroot-state")

	// Setup
	repoDir := filepath.Join(tempDir, "source-repo")
	baretreeRoot := filepath.Join(tempDir, "baretree-root")
	setupGitRepoWithRemote(t, repoDir, "git@github.com:testuser/state.git")

	// Create hierarchical worktree
	runGitSuccess(t, repoDir, "branch", "feat/auth")
	worktreeDir := filepath.Join(tempDir, "external-wt")
	runGitSuccess(t, repoDir, "worktree", "add", worktreeDir, "feat/auth")

	// Create working state
	writeFile(t, filepath.Join(worktreeDir, "committed.txt"), "committed")
	runGitSuccess(t, worktreeDir, "add", "committed.txt")
	runGitSuccess(t, worktreeDir, "commit", "-m", "committed file")

	// Modify committed file (unstaged)
	writeFile(t, filepath.Join(worktreeDir, "committed.txt"), "modified committed")

	// Stage a new file
	writeFile(t, filepath.Join(worktreeDir, "staged.txt"), "staged content")
	runGitSuccess(t, worktreeDir, "add", "staged.txt")

	// Create untracked file
	writeFile(t, filepath.Join(worktreeDir, "untracked.txt"), "untracked content")

	// Get original status
	originalStatus := runGitSuccess(t, worktreeDir, "status", "--porcelain")

	t.Run("migrate preserves working state in hierarchical worktree", func(t *testing.T) {
		env := map[string]string{
			"BARETREE_ROOT": baretreeRoot,
		}
		_, _, err := runBtWithEnv(t, repoDir, env, "repo", "migrate", ".", "-r")
		if err != nil {
			t.Fatalf("migrate failed: %v", err)
		}

		destDir := filepath.Join(baretreeRoot, "github.com", "testuser", "state")
		newWorktree := filepath.Join(destDir, "feat", "auth")

		// Check status matches
		newStatus := runGitSuccess(t, newWorktree, "status", "--porcelain")
		if originalStatus != newStatus {
			t.Errorf("working tree state not preserved\noriginal:\n%s\nmigrated:\n%s", originalStatus, newStatus)
		}

		// Verify file contents
		assertFileContent(t, filepath.Join(newWorktree, "committed.txt"), "modified committed")
		assertFileContent(t, filepath.Join(newWorktree, "staged.txt"), "staged content")
		assertFileContent(t, filepath.Join(newWorktree, "untracked.txt"), "untracked content")
	})
}

// TestMigrate_ToRoot_WithSubmodule tests --to-root migration with submodules
func TestMigrate_ToRoot_WithSubmodule(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-toroot-submodule")

	// Setup: create a repository to use as submodule
	submoduleRepo := filepath.Join(tempDir, "submodule-repo")
	setupGitRepo(t, submoduleRepo)
	writeFile(t, filepath.Join(submoduleRepo, "submodule-file.txt"), "submodule content")
	runGitSuccess(t, submoduleRepo, "add", ".")
	runGitSuccess(t, submoduleRepo, "commit", "-m", "Add submodule file")

	// Setup: create main repository with submodule and remote
	repoDir := filepath.Join(tempDir, "main-repo")
	baretreeRoot := filepath.Join(tempDir, "baretree-root")
	setupGitRepoWithRemote(t, repoDir, "git@github.com:testuser/submodule-test.git")

	// Add submodule
	runGitSuccess(t, repoDir, "-c", "protocol.file.allow=always", "submodule", "add", submoduleRepo, "libs/mylib")
	runGitSuccess(t, repoDir, "commit", "-m", "Add submodule")

	t.Run("migrate to root preserves submodule", func(t *testing.T) {
		env := map[string]string{
			"BARETREE_ROOT": baretreeRoot,
		}
		_, _, err := runBtWithEnv(t, repoDir, env, "repo", "migrate", ".", "-r")
		if err != nil {
			t.Fatalf("migrate to root with submodule failed: %v", err)
		}

		// Check destination has baretree structure
		destDir := filepath.Join(baretreeRoot, "github.com", "testuser", "submodule-test")
		worktreeDir := filepath.Join(destDir, "master")

		// Check .gitmodules exists
		assertFileExists(t, filepath.Join(worktreeDir, ".gitmodules"))

		// Check submodule directory exists
		assertFileExists(t, filepath.Join(worktreeDir, "libs", "mylib"))

		// Check submodule content
		assertFileExists(t, filepath.Join(worktreeDir, "libs", "mylib", "submodule-file.txt"))
		assertFileContent(t, filepath.Join(worktreeDir, "libs", "mylib", "submodule-file.txt"), "submodule content")

		// Check .git/modules exists in bare repo
		assertFileExists(t, filepath.Join(destDir, ".git", "modules"))
	})

	t.Run("worktree is functional after migrate", func(t *testing.T) {
		destDir := filepath.Join(baretreeRoot, "github.com", "testuser", "submodule-test")
		worktreeDir := filepath.Join(destDir, "master")
		stdout := runGitSuccess(t, worktreeDir, "status")
		assertOutputContains(t, stdout, "On branch master")

		// Check submodule status
		stdout = runGitSuccess(t, worktreeDir, "submodule", "status")
		assertOutputContains(t, stdout, "libs/mylib")
	})
}

// TestMigrate_WithMultipleSubmodules tests migration with multiple submodules
func TestMigrate_WithMultipleSubmodules(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-multi-submodule")

	// Setup: create multiple submodule repositories
	submoduleRepo1 := filepath.Join(tempDir, "submodule-repo1")
	setupGitRepo(t, submoduleRepo1)
	writeFile(t, filepath.Join(submoduleRepo1, "lib1.txt"), "library 1 content")
	runGitSuccess(t, submoduleRepo1, "add", ".")
	runGitSuccess(t, submoduleRepo1, "commit", "-m", "Add lib1 file")

	submoduleRepo2 := filepath.Join(tempDir, "submodule-repo2")
	setupGitRepo(t, submoduleRepo2)
	writeFile(t, filepath.Join(submoduleRepo2, "lib2.txt"), "library 2 content")
	runGitSuccess(t, submoduleRepo2, "add", ".")
	runGitSuccess(t, submoduleRepo2, "commit", "-m", "Add lib2 file")

	submoduleRepo3 := filepath.Join(tempDir, "submodule-repo3")
	setupGitRepo(t, submoduleRepo3)
	writeFile(t, filepath.Join(submoduleRepo3, "lib3.txt"), "library 3 content")
	runGitSuccess(t, submoduleRepo3, "add", ".")
	runGitSuccess(t, submoduleRepo3, "commit", "-m", "Add lib3 file")

	// Setup: create main repository with multiple submodules
	repoDir := filepath.Join(tempDir, "main-repo")
	setupGitRepo(t, repoDir)

	// Add multiple submodules in different locations
	runGitSuccess(t, repoDir, "-c", "protocol.file.allow=always", "submodule", "add", submoduleRepo1, "libs/lib1")
	runGitSuccess(t, repoDir, "-c", "protocol.file.allow=always", "submodule", "add", submoduleRepo2, "libs/lib2")
	runGitSuccess(t, repoDir, "-c", "protocol.file.allow=always", "submodule", "add", submoduleRepo3, "vendor/lib3")
	runGitSuccess(t, repoDir, "commit", "-m", "Add multiple submodules")

	t.Run("migrate preserves multiple submodules with -d", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "migrated-repo")
		runBtSuccess(t, repoDir, "repo", "migrate", ".", "-d", destDir)

		worktreeDir := filepath.Join(destDir, "master")

		// Check .gitmodules exists
		assertFileExists(t, filepath.Join(worktreeDir, ".gitmodules"))

		// Check all submodules exist
		assertFileExists(t, filepath.Join(worktreeDir, "libs", "lib1", "lib1.txt"))
		assertFileContent(t, filepath.Join(worktreeDir, "libs", "lib1", "lib1.txt"), "library 1 content")

		assertFileExists(t, filepath.Join(worktreeDir, "libs", "lib2", "lib2.txt"))
		assertFileContent(t, filepath.Join(worktreeDir, "libs", "lib2", "lib2.txt"), "library 2 content")

		assertFileExists(t, filepath.Join(worktreeDir, "vendor", "lib3", "lib3.txt"))
		assertFileContent(t, filepath.Join(worktreeDir, "vendor", "lib3", "lib3.txt"), "library 3 content")

		// Check .git/modules has all submodules
		assertFileExists(t, filepath.Join(destDir, ".git", "modules"))
	})

	t.Run("all submodules are functional", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "migrated-repo")
		worktreeDir := filepath.Join(destDir, "master")

		// Check submodule status shows all submodules
		stdout := runGitSuccess(t, worktreeDir, "submodule", "status")
		assertOutputContains(t, stdout, "libs/lib1")
		assertOutputContains(t, stdout, "libs/lib2")
		assertOutputContains(t, stdout, "vendor/lib3")
	})
}

// TestMigrate_WithMultipleSubmodules_InPlace tests in-place migration with multiple submodules
func TestMigrate_WithMultipleSubmodules_InPlace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-multi-submodule-inplace")

	// Setup submodule repos
	submoduleRepo1 := filepath.Join(tempDir, "submodule-repo1")
	setupGitRepo(t, submoduleRepo1)
	writeFile(t, filepath.Join(submoduleRepo1, "lib1.txt"), "lib1 content")
	runGitSuccess(t, submoduleRepo1, "add", ".")
	runGitSuccess(t, submoduleRepo1, "commit", "-m", "Add lib1")

	submoduleRepo2 := filepath.Join(tempDir, "submodule-repo2")
	setupGitRepo(t, submoduleRepo2)
	writeFile(t, filepath.Join(submoduleRepo2, "lib2.txt"), "lib2 content")
	runGitSuccess(t, submoduleRepo2, "add", ".")
	runGitSuccess(t, submoduleRepo2, "commit", "-m", "Add lib2")

	// Setup main repo
	repoDir := filepath.Join(tempDir, "main-repo")
	setupGitRepo(t, repoDir)
	runGitSuccess(t, repoDir, "-c", "protocol.file.allow=always", "submodule", "add", submoduleRepo1, "deps/lib1")
	runGitSuccess(t, repoDir, "-c", "protocol.file.allow=always", "submodule", "add", submoduleRepo2, "deps/lib2")
	runGitSuccess(t, repoDir, "commit", "-m", "Add submodules")

	t.Run("in-place migrate preserves multiple submodules", func(t *testing.T) {
		runBtSuccess(t, repoDir, "repo", "migrate", ".", "-i")

		worktreeDir := filepath.Join(repoDir, "master")

		assertFileExists(t, filepath.Join(worktreeDir, ".gitmodules"))
		assertFileExists(t, filepath.Join(worktreeDir, "deps", "lib1", "lib1.txt"))
		assertFileContent(t, filepath.Join(worktreeDir, "deps", "lib1", "lib1.txt"), "lib1 content")
		assertFileExists(t, filepath.Join(worktreeDir, "deps", "lib2", "lib2.txt"))
		assertFileContent(t, filepath.Join(worktreeDir, "deps", "lib2", "lib2.txt"), "lib2 content")
		assertFileExists(t, filepath.Join(repoDir, ".git", "modules"))
	})
}

// TestMigrate_WithNestedSubmodule tests migration with nested submodules (submodule within submodule)
func TestMigrate_WithNestedSubmodule(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-nested-submodule")

	// Setup: create inner submodule repository
	innerSubmoduleRepo := filepath.Join(tempDir, "inner-submodule")
	setupGitRepo(t, innerSubmoduleRepo)
	writeFile(t, filepath.Join(innerSubmoduleRepo, "inner.txt"), "inner submodule content")
	runGitSuccess(t, innerSubmoduleRepo, "add", ".")
	runGitSuccess(t, innerSubmoduleRepo, "commit", "-m", "Add inner file")

	// Setup: create outer submodule repository with inner submodule
	outerSubmoduleRepo := filepath.Join(tempDir, "outer-submodule")
	setupGitRepo(t, outerSubmoduleRepo)
	writeFile(t, filepath.Join(outerSubmoduleRepo, "outer.txt"), "outer submodule content")
	runGitSuccess(t, outerSubmoduleRepo, "add", ".")
	runGitSuccess(t, outerSubmoduleRepo, "commit", "-m", "Add outer file")

	// Add inner submodule to outer submodule
	runGitSuccess(t, outerSubmoduleRepo, "-c", "protocol.file.allow=always", "submodule", "add", innerSubmoduleRepo, "nested/inner")
	runGitSuccess(t, outerSubmoduleRepo, "commit", "-m", "Add inner submodule")

	// Setup: create main repository with outer submodule
	repoDir := filepath.Join(tempDir, "main-repo")
	setupGitRepo(t, repoDir)
	runGitSuccess(t, repoDir, "-c", "protocol.file.allow=always", "submodule", "add", outerSubmoduleRepo, "libs/outer")
	runGitSuccess(t, repoDir, "commit", "-m", "Add outer submodule")

	// Initialize nested submodules
	runGitSuccess(t, repoDir, "-c", "protocol.file.allow=always", "submodule", "update", "--init", "--recursive")

	t.Run("migrate preserves nested submodules with -d", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "migrated-repo")
		runBtSuccess(t, repoDir, "repo", "migrate", ".", "-d", destDir)

		worktreeDir := filepath.Join(destDir, "master")

		// Check outer submodule
		assertFileExists(t, filepath.Join(worktreeDir, ".gitmodules"))
		assertFileExists(t, filepath.Join(worktreeDir, "libs", "outer", "outer.txt"))
		assertFileContent(t, filepath.Join(worktreeDir, "libs", "outer", "outer.txt"), "outer submodule content")

		// Check inner (nested) submodule
		assertFileExists(t, filepath.Join(worktreeDir, "libs", "outer", "nested", "inner", "inner.txt"))
		assertFileContent(t, filepath.Join(worktreeDir, "libs", "outer", "nested", "inner", "inner.txt"), "inner submodule content")
	})
}

// TestMigrate_WithNestedSubmodule_InPlace tests in-place migration with nested submodules
func TestMigrate_WithNestedSubmodule_InPlace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-nested-submodule-inplace")

	// Setup inner submodule
	innerSubmoduleRepo := filepath.Join(tempDir, "inner-submodule")
	setupGitRepo(t, innerSubmoduleRepo)
	writeFile(t, filepath.Join(innerSubmoduleRepo, "inner.txt"), "inner content")
	runGitSuccess(t, innerSubmoduleRepo, "add", ".")
	runGitSuccess(t, innerSubmoduleRepo, "commit", "-m", "Add inner")

	// Setup outer submodule with inner submodule
	outerSubmoduleRepo := filepath.Join(tempDir, "outer-submodule")
	setupGitRepo(t, outerSubmoduleRepo)
	writeFile(t, filepath.Join(outerSubmoduleRepo, "outer.txt"), "outer content")
	runGitSuccess(t, outerSubmoduleRepo, "add", ".")
	runGitSuccess(t, outerSubmoduleRepo, "commit", "-m", "Add outer")
	runGitSuccess(t, outerSubmoduleRepo, "-c", "protocol.file.allow=always", "submodule", "add", innerSubmoduleRepo, "nested/inner")
	runGitSuccess(t, outerSubmoduleRepo, "commit", "-m", "Add inner submodule")

	// Setup main repo
	repoDir := filepath.Join(tempDir, "main-repo")
	setupGitRepo(t, repoDir)
	runGitSuccess(t, repoDir, "-c", "protocol.file.allow=always", "submodule", "add", outerSubmoduleRepo, "vendor/outer")
	runGitSuccess(t, repoDir, "commit", "-m", "Add outer submodule")
	runGitSuccess(t, repoDir, "-c", "protocol.file.allow=always", "submodule", "update", "--init", "--recursive")

	t.Run("in-place migrate preserves nested submodules", func(t *testing.T) {
		runBtSuccess(t, repoDir, "repo", "migrate", ".", "-i")

		worktreeDir := filepath.Join(repoDir, "master")

		assertFileExists(t, filepath.Join(worktreeDir, "vendor", "outer", "outer.txt"))
		assertFileContent(t, filepath.Join(worktreeDir, "vendor", "outer", "outer.txt"), "outer content")
		assertFileExists(t, filepath.Join(worktreeDir, "vendor", "outer", "nested", "inner", "inner.txt"))
		assertFileContent(t, filepath.Join(worktreeDir, "vendor", "outer", "nested", "inner", "inner.txt"), "inner content")
	})
}

// TestMigrate_WithExternalWorktreesAndSubmodule tests migration with both external worktrees and submodules
func TestMigrate_WithExternalWorktreesAndSubmodule(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-worktree-submodule")

	// Setup: create a submodule repository
	submoduleRepo := filepath.Join(tempDir, "submodule-repo")
	setupGitRepo(t, submoduleRepo)
	writeFile(t, filepath.Join(submoduleRepo, "lib.txt"), "library content")
	runGitSuccess(t, submoduleRepo, "add", ".")
	runGitSuccess(t, submoduleRepo, "commit", "-m", "Add lib file")

	// Setup: create main repository with submodule
	repoDir := filepath.Join(tempDir, "main-repo")
	setupGitRepo(t, repoDir)
	runGitSuccess(t, repoDir, "-c", "protocol.file.allow=always", "submodule", "add", submoduleRepo, "libs/mylib")
	runGitSuccess(t, repoDir, "commit", "-m", "Add submodule")

	// Create external worktree
	runGitSuccess(t, repoDir, "branch", "feature-branch")
	worktreeDir := filepath.Join(tempDir, "external-worktree")
	runGitSuccess(t, repoDir, "worktree", "add", worktreeDir, "feature-branch")

	// Initialize submodule in the external worktree
	runGitSuccess(t, worktreeDir, "-c", "protocol.file.allow=always", "submodule", "update", "--init")

	// Add content to external worktree
	writeFile(t, filepath.Join(worktreeDir, "feature.txt"), "feature content")
	runGitSuccess(t, worktreeDir, "add", "feature.txt")
	runGitSuccess(t, worktreeDir, "commit", "-m", "Add feature file")

	t.Run("migrate with external worktrees and submodule with -d", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "migrated-repo")
		stdout := runBtSuccess(t, repoDir, "repo", "migrate", ".", "-d", destDir)

		assertOutputContains(t, stdout, "External worktrees to migrate: 1")

		// Check main worktree has submodule
		masterWorktreeDir := filepath.Join(destDir, "master")
		assertFileExists(t, filepath.Join(masterWorktreeDir, ".gitmodules"))
		assertFileExists(t, filepath.Join(masterWorktreeDir, "libs", "mylib", "lib.txt"))
		assertFileContent(t, filepath.Join(masterWorktreeDir, "libs", "mylib", "lib.txt"), "library content")

		// Check external worktree was migrated
		featureWorktreeDir := filepath.Join(destDir, "feature-branch")
		assertFileExists(t, filepath.Join(featureWorktreeDir, "feature.txt"))
		assertFileContent(t, filepath.Join(featureWorktreeDir, "feature.txt"), "feature content")

		// Check submodule exists in feature worktree
		assertFileExists(t, filepath.Join(featureWorktreeDir, ".gitmodules"))
		assertFileExists(t, filepath.Join(featureWorktreeDir, "libs", "mylib", "lib.txt"))
	})

	t.Run("master worktree is functional", func(t *testing.T) {
		destDir := filepath.Join(tempDir, "migrated-repo")

		// Check master worktree
		stdout := runGitSuccess(t, filepath.Join(destDir, "master"), "status")
		assertOutputContains(t, stdout, "On branch master")

		// Check submodule status in master
		stdout = runGitSuccess(t, filepath.Join(destDir, "master"), "submodule", "status")
		assertOutputContains(t, stdout, "libs/mylib")
	})

	// Note: Feature worktree submodule status is not fully tested here because
	// external worktrees with submodules require special handling of the
	// .git/worktrees/<worktree>/modules directory structure, which is complex.
	// The basic functionality (file existence) is verified above.
}

// TestMigrate_WithExternalWorktreesAndSubmodule_InPlace tests in-place migration with external worktrees and submodules
func TestMigrate_WithExternalWorktreesAndSubmodule_InPlace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "migrate-worktree-submodule-inplace")

	// Setup submodule repo
	submoduleRepo := filepath.Join(tempDir, "submodule-repo")
	setupGitRepo(t, submoduleRepo)
	writeFile(t, filepath.Join(submoduleRepo, "lib.txt"), "lib content")
	runGitSuccess(t, submoduleRepo, "add", ".")
	runGitSuccess(t, submoduleRepo, "commit", "-m", "Add lib")

	// Setup main repo with submodule
	repoDir := filepath.Join(tempDir, "main-repo")
	setupGitRepo(t, repoDir)
	runGitSuccess(t, repoDir, "-c", "protocol.file.allow=always", "submodule", "add", submoduleRepo, "vendor/lib")
	runGitSuccess(t, repoDir, "commit", "-m", "Add submodule")

	// Create external worktree
	runGitSuccess(t, repoDir, "branch", "develop")
	worktreeDir := filepath.Join(tempDir, "develop-worktree")
	runGitSuccess(t, repoDir, "worktree", "add", worktreeDir, "develop")
	runGitSuccess(t, worktreeDir, "-c", "protocol.file.allow=always", "submodule", "update", "--init")

	// Add content to worktree
	writeFile(t, filepath.Join(worktreeDir, "develop.txt"), "develop content")
	runGitSuccess(t, worktreeDir, "add", "develop.txt")
	runGitSuccess(t, worktreeDir, "commit", "-m", "Add develop file")

	t.Run("in-place migrate handles external worktrees with submodule", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "repo", "migrate", ".", "-i")

		assertOutputContains(t, stdout, "External worktrees to migrate: 1")

		// Check main worktree
		masterWorktreeDir := filepath.Join(repoDir, "master")
		assertFileExists(t, filepath.Join(masterWorktreeDir, ".gitmodules"))
		assertFileExists(t, filepath.Join(masterWorktreeDir, "vendor", "lib", "lib.txt"))

		// Check develop worktree was migrated
		developWorktreeDir := filepath.Join(repoDir, "develop")
		assertFileExists(t, filepath.Join(developWorktreeDir, "develop.txt"))
		assertFileContent(t, filepath.Join(developWorktreeDir, "develop.txt"), "develop content")
		assertFileExists(t, filepath.Join(developWorktreeDir, ".gitmodules"))

		// Original external worktree should be removed
		assertFileNotExists(t, worktreeDir)
	})
}
