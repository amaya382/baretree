package e2e

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// TestJourney1_BasicWorkflow tests the basic clone and worktree workflow
// Scenario: Clone a new project, create feature branch, navigate between worktrees
func TestJourney1_BasicWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "journey1")

	// Step 1: Clone repository
	t.Run("clone repository", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "my-project")

		assertOutputContains(t, stdout, "Successfully cloned")

		// Verify structure
		projectDir := filepath.Join(tempDir, "my-project")
		assertFileExists(t, projectDir)
		assertFileExists(t, filepath.Join(projectDir, ".bare"))
		assertFileExists(t, filepath.Join(projectDir, ".bare"))

		// Should have a default branch worktree (main or master)
		mainExists := isDirectory(filepath.Join(projectDir, "main"))
		masterExists := isDirectory(filepath.Join(projectDir, "master"))
		if !mainExists && !masterExists {
			t.Error("expected main or master worktree to exist")
		}
	})

	projectDir := filepath.Join(tempDir, "my-project")

	// Step 2: List worktrees (should show only default branch)
	t.Run("list worktrees - initial", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "list")

		// Should contain at least one worktree marker
		assertOutputContains(t, stdout, "@")
	})

	// Step 3: Add feature branch worktree
	t.Run("add feature branch", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "add", "-b", "feature/auth")

		assertOutputContains(t, stdout, "Worktree created")
		assertFileExists(t, filepath.Join(projectDir, "feature", "auth"))
	})

	// Step 4: List worktrees (should show 2 worktrees)
	t.Run("list worktrees - after add", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "list")

		assertOutputContains(t, stdout, "feature/auth")
	})

	// Step 5: Test cd command (outputs path)
	t.Run("cd to feature branch", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "cd", "feature/auth")

		expectedPath := filepath.Join(projectDir, "feature", "auth")
		assertOutputContains(t, stdout, expectedPath)
	})

	// Step 6: Test cd to main (@)
	t.Run("cd to main with @", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "cd", "@")

		// Should output a path (either main or master)
		if stdout == "" {
			t.Error("expected cd @ to output a path")
		}
	})

	// Step 7: Test status command
	t.Run("status command", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "status")

		assertOutputContains(t, stdout, "Repository Information")
		assertOutputContains(t, stdout, "Worktrees")
		assertOutputContains(t, stdout, "feature/auth")
	})
}

// TestAddExistingWorktree tests adding a worktree that already exists
func TestAddExistingWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "add-existing")

	runBtSuccess(t, tempDir, "repo", "init", "test-repo")
	projectDir := filepath.Join(tempDir, "test-repo")
	runBtSuccess(t, projectDir, "add", "-b", "feature/existing")

	t.Run("add existing worktree shows guidance", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "add", "feature/existing")

		assertOutputContains(t, stdout, "already exists")
		assertOutputContains(t, stdout, "bt cd feature/existing")
	})
}

// TestJourney2_MultipleFeaturesAndCleanup tests working with multiple features
func TestJourney2_MultipleFeaturesAndCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "journey2")

	// Clone
	runBtSuccess(t, tempDir, "repo", "clone", TestRepo, "multi-feature")
	projectDir := filepath.Join(tempDir, "multi-feature")

	// Add multiple feature branches
	t.Run("add multiple features", func(t *testing.T) {
		runBtSuccess(t, projectDir, "add", "-b", "feature/login")
		runBtSuccess(t, projectDir, "add", "-b", "feature/signup")
		runBtSuccess(t, projectDir, "add", "-b", "bugfix/cors")

		assertFileExists(t, filepath.Join(projectDir, "feature", "login"))
		assertFileExists(t, filepath.Join(projectDir, "feature", "signup"))
		assertFileExists(t, filepath.Join(projectDir, "bugfix", "cors"))
	})

	// List should show all worktrees
	t.Run("list all worktrees", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "list")

		assertOutputContains(t, stdout, "feature/login")
		assertOutputContains(t, stdout, "feature/signup")
		assertOutputContains(t, stdout, "bugfix/cors")
	})

	// Remove a worktree
	t.Run("remove worktree", func(t *testing.T) {
		runBtSuccess(t, projectDir, "remove", "bugfix/cors", "--force")

		assertFileNotExists(t, filepath.Join(projectDir, "bugfix", "cors"))
	})

	// List should not show removed worktree
	t.Run("list after remove", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "list")

		assertOutputContains(t, stdout, "feature/login")
		assertOutputContains(t, stdout, "feature/signup")
		assertOutputNotContains(t, stdout, "bugfix/cors")
	})
}

// TestJourney3_MigrateExistingRepo tests migrating an existing repository
func TestJourney3_MigrateExistingRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "journey3")

	// First, clone a regular repository using git
	t.Run("setup existing repo", func(t *testing.T) {
		cmd := exec.Command("git", "clone", TestRepo, "existing-repo")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to clone test repo: %v", err)
		}

		// Create a branch
		repoDir := filepath.Join(tempDir, "existing-repo")
		cmd = exec.Command("git", "checkout", "-b", "develop")
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create branch: %v", err)
		}
	})

	// Migrate the repository
	t.Run("migrate repository", func(t *testing.T) {
		repoDir := filepath.Join(tempDir, "existing-repo")
		newDir := filepath.Join(tempDir, "existing-repo-baretree")
		stdout := runBtSuccess(t, repoDir, "repo", "migrate", ".", "-d", newDir)

		assertOutputContains(t, stdout, "Migration successful")

		// Check new structure
		assertFileExists(t, newDir)
		assertFileExists(t, filepath.Join(newDir, ".bare"))
		assertFileExists(t, filepath.Join(newDir, ".bare"))
		assertFileExists(t, filepath.Join(newDir, "develop"))
	})

	// Verify migrated repo works
	t.Run("use migrated repo", func(t *testing.T) {
		newDir := filepath.Join(tempDir, "existing-repo-baretree")

		stdout := runBtSuccess(t, newDir, "status")
		assertOutputContains(t, stdout, "develop")

		// Can add new worktrees
		runBtSuccess(t, newDir, "add", "-b", "feature/new")
		assertFileExists(t, filepath.Join(newDir, "feature", "new"))
	})
}
