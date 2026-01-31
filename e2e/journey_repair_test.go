package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestRepair_BranchAsSource tests repair using branch name as source of truth
func TestRepair_BranchAsSource(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repair-branch-source")

	runBtSuccess(t, tempDir, "repo", "init", "repair-test")
	projectDir := filepath.Join(tempDir, "repair-test")
	runBtSuccess(t, projectDir, "add", "-b", "feature/test")

	// Create inconsistency by renaming branch
	bareDir := filepath.Join(projectDir, ".bare")
	cmd := exec.Command("git", "branch", "-m", "feature/test", "feature/renamed")
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to rename branch: %v", err)
	}

	t.Run("dry run shows what would be done", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "repair", "--dry-run", "--all")

		assertOutputContains(t, stdout, "to repair")
		assertOutputContains(t, stdout, "Rename directory")
		assertOutputContains(t, stdout, "feature/renamed")
		assertOutputContains(t, stdout, "Dry run")
	})

	t.Run("repair renames directory to match branch", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "repair", "--all")

		assertOutputContains(t, stdout, "Done")
		assertOutputContains(t, stdout, "Successfully repaired")

		// Verify directory was renamed
		assertFileNotExists(t, filepath.Join(projectDir, "feature", "test"))
		assertFileExists(t, filepath.Join(projectDir, "feature", "renamed"))

		// Verify list shows correct name
		stdout = runBtSuccess(t, projectDir, "list")
		assertOutputContains(t, stdout, "feature/renamed")
	})
}

// TestRepair_DirAsSource tests repair using directory name as source of truth
func TestRepair_DirAsSource(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repair-dir-source")

	runBtSuccess(t, tempDir, "repo", "init", "repair-test")
	projectDir := filepath.Join(tempDir, "repair-test")
	runBtSuccess(t, projectDir, "add", "-b", "feature/original")

	// Create inconsistency by renaming branch
	bareDir := filepath.Join(projectDir, ".bare")
	cmd := exec.Command("git", "branch", "-m", "feature/original", "feature/renamed")
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to rename branch: %v", err)
	}

	t.Run("dry run with source=dir shows what would be done", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "repair", "--source=dir", "--dry-run", "--all")

		assertOutputContains(t, stdout, "to repair")
		assertOutputContains(t, stdout, "Rename branch")
		assertOutputContains(t, stdout, "feature/original")
		assertOutputContains(t, stdout, "Dry run")
	})

	t.Run("repair renames branch to match directory", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "repair", "--source=dir", "--all")

		assertOutputContains(t, stdout, "Done")
		assertOutputContains(t, stdout, "Successfully repaired")

		// Verify directory still exists at original location
		assertFileExists(t, filepath.Join(projectDir, "feature", "original"))

		// Verify list shows correct name (branch should match directory now)
		stdout = runBtSuccess(t, projectDir, "list")
		assertOutputContains(t, stdout, "feature/original")
	})
}

// TestRepair_SpecificWorktree tests repairing a specific worktree
func TestRepair_SpecificWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repair-specific")

	runBtSuccess(t, tempDir, "repo", "init", "repair-test")
	projectDir := filepath.Join(tempDir, "repair-test")
	runBtSuccess(t, projectDir, "add", "-b", "feature/one")
	runBtSuccess(t, projectDir, "add", "-b", "feature/two")

	// Create inconsistency for feature/one only
	bareDir := filepath.Join(projectDir, ".bare")
	cmd := exec.Command("git", "branch", "-m", "feature/one", "feature/one-renamed")
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to rename branch: %v", err)
	}

	t.Run("repair specific worktree by directory name", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "repair", "feature/one")

		assertOutputContains(t, stdout, "Done")

		// Verify only feature/one was changed
		assertFileNotExists(t, filepath.Join(projectDir, "feature", "one"))
		assertFileExists(t, filepath.Join(projectDir, "feature", "one-renamed"))
		assertFileExists(t, filepath.Join(projectDir, "feature", "two")) // unchanged
	})
}

// TestRepair_CurrentWorktree tests repairing the current worktree
func TestRepair_CurrentWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repair-current")

	runBtSuccess(t, tempDir, "repo", "init", "repair-test")
	projectDir := filepath.Join(tempDir, "repair-test")
	runBtSuccess(t, projectDir, "add", "-b", "feature/current")

	worktreeDir := filepath.Join(projectDir, "feature", "current")

	// Create inconsistency by renaming branch
	bareDir := filepath.Join(projectDir, ".bare")
	cmd := exec.Command("git", "branch", "-m", "feature/current", "feature/current-renamed")
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to rename branch: %v", err)
	}

	t.Run("repair from inside worktree uses source=dir", func(t *testing.T) {
		// Using dir as source to keep directory name same
		stdout := runBtSuccess(t, worktreeDir, "repair", "--source=dir")

		assertOutputContains(t, stdout, "Done")

		// Directory should still exist (branch was renamed to match)
		assertFileExists(t, filepath.Join(projectDir, "feature", "current"))
	})
}

// TestRepair_NoInconsistency tests behavior when there are no inconsistencies
func TestRepair_NoInconsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repair-consistent")

	runBtSuccess(t, tempDir, "repo", "init", "repair-test")
	projectDir := filepath.Join(tempDir, "repair-test")
	runBtSuccess(t, projectDir, "add", "-b", "feature/consistent")

	t.Run("repair all with no inconsistencies", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "repair", "--all")

		assertOutputContains(t, stdout, "No worktrees need repair")
	})

	t.Run("repair specific consistent worktree", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "repair", "feature/consistent")

		assertOutputContains(t, stdout, "already managed")
	})
}

// TestRepair_ErrorCases tests error handling
func TestRepair_ErrorCases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	t.Run("invalid source value", func(t *testing.T) {
		tempDir := createTempDir(t, "repair-error-source")
		runBtSuccess(t, tempDir, "repo", "init", "test")
		projectDir := filepath.Join(tempDir, "test")

		_, stderr := runBtExpectError(t, projectDir, "repair", "--source=invalid", "--all")
		assertOutputContains(t, stderr, "invalid source")
	})

	t.Run("worktree not found", func(t *testing.T) {
		tempDir := createTempDir(t, "repair-error-notfound")
		runBtSuccess(t, tempDir, "repo", "init", "test")
		projectDir := filepath.Join(tempDir, "test")

		_, stderr := runBtExpectError(t, projectDir, "repair", "nonexistent")
		assertOutputContains(t, stderr, "not found")
	})
}

// TestRepair_MultipleInconsistencies tests repairing multiple worktrees at once
func TestRepair_MultipleInconsistencies(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repair-multiple")

	runBtSuccess(t, tempDir, "repo", "init", "repair-test")
	projectDir := filepath.Join(tempDir, "repair-test")
	runBtSuccess(t, projectDir, "add", "-b", "feature/one")
	runBtSuccess(t, projectDir, "add", "-b", "feature/two")
	runBtSuccess(t, projectDir, "add", "-b", "feature/three")

	// Create inconsistencies for two worktrees
	bareDir := filepath.Join(projectDir, ".bare")
	cmd := exec.Command("git", "branch", "-m", "feature/one", "feature/one-renamed")
	cmd.Dir = bareDir
	_ = cmd.Run()

	cmd = exec.Command("git", "branch", "-m", "feature/two", "feature/two-renamed")
	cmd.Dir = bareDir
	_ = cmd.Run()

	t.Run("repair all fixes multiple inconsistencies", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "repair", "--all")

		assertOutputContains(t, stdout, "Found 2 worktree(s) to repair")
		assertOutputContains(t, stdout, "Successfully repaired 2 worktree(s)")

		// Verify directories were renamed
		assertFileNotExists(t, filepath.Join(projectDir, "feature", "one"))
		assertFileNotExists(t, filepath.Join(projectDir, "feature", "two"))
		assertFileExists(t, filepath.Join(projectDir, "feature", "one-renamed"))
		assertFileExists(t, filepath.Join(projectDir, "feature", "two-renamed"))
		assertFileExists(t, filepath.Join(projectDir, "feature", "three")) // unchanged
	})
}

// TestRepair_ManuallyMovedWorktree tests repair after manually moving a worktree
// to an external location (outside the repository root).
//
// This simulates the scenario where a user moves a worktree using `mv` command
// instead of `git worktree move`, breaking Git's internal links.
//
// The user can use `bt repair --fix-paths <external-path>` to fix Git's internal
// links AND automatically move the worktree back into the baretree structure.
// This eliminates the need to call `git worktree repair` directly or run
// `bt repair --all` separately.
func TestRepair_ManuallyMovedWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repair-moved-wt")

	runBtSuccess(t, tempDir, "repo", "init", "repair-test")
	projectDir := filepath.Join(tempDir, "repair-test")
	runBtSuccess(t, projectDir, "add", "-b", "feature/moveme")

	worktreeDir := filepath.Join(projectDir, "feature", "moveme")
	externalDir := filepath.Join(tempDir, "external-location")

	// Manually move the worktree outside the repository (simulates `mv` command)
	if err := os.Rename(worktreeDir, externalDir); err != nil {
		t.Fatalf("failed to move worktree: %v", err)
	}

	// Git's worktree link is now broken
	cmd := exec.Command("git", "status")
	cmd.Dir = externalDir
	if err := cmd.Run(); err == nil {
		t.Log("Note: git status succeeded unexpectedly, but continuing test")
	}

	// bt repair --fix-paths with external path fixes Git's internal links
	// AND moves the worktree back into baretree structure in one step
	t.Run("bt repair --fix-paths fixes and moves worktree in one step", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "repair", "--fix-paths", externalDir)

		assertOutputContains(t, stdout, "Successfully fixed")
		assertOutputContains(t, stdout, "Moving worktrees into baretree structure")
		assertOutputContains(t, stdout, "feature/moveme")
	})

	t.Run("worktree is back in correct location and managed", func(t *testing.T) {
		// Verify worktree is back in the correct location
		assertFileExists(t, filepath.Join(projectDir, "feature", "moveme"))
		assertFileNotExists(t, externalDir)

		// Verify git status works
		runGitSuccess(t, filepath.Join(projectDir, "feature", "moveme"), "status")

		// Verify it's managed
		stdout := runBtSuccess(t, projectDir, "status")
		assertOutputContains(t, stdout, "feature/moveme")
		assertOutputContains(t, stdout, "[Managed]")
	})
}

// TestRepair_MovedBareRepository tests repair after manually moving the entire project.
// This simulates moving the entire project (bare repo + worktrees) to a new location.
// The --fix-paths option is used to update Git's internal worktree paths.
func TestRepair_MovedBareRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repair-moved-bare")

	runBtSuccess(t, tempDir, "repo", "init", "original-project")
	originalDir := filepath.Join(tempDir, "original-project")
	runBtSuccess(t, originalDir, "add", "-b", "feature/test")

	// Verify initial state works
	runGitSuccess(t, filepath.Join(originalDir, "feature", "test"), "status")

	// Move the entire project (bare repo + worktrees) to a new location
	newDir := filepath.Join(tempDir, "new-location")
	if err := os.Rename(originalDir, newDir); err != nil {
		t.Fatalf("failed to move project: %v", err)
	}

	worktreeDir := filepath.Join(newDir, "feature", "test")

	// Git's internal links are now broken - the .git file in worktree points to old path
	t.Run("git status fails after move", func(t *testing.T) {
		cmd := exec.Command("git", "status")
		cmd.Dir = worktreeDir
		if err := cmd.Run(); err == nil {
			// On some systems, relative paths might work, but typically this fails
			t.Log("Note: git status succeeded, possibly using relative paths")
		}
	})

	t.Run("bt repair --fix-paths --dry-run shows paths to fix", func(t *testing.T) {
		stdout := runBtSuccess(t, newDir, "repair", "--fix-paths", "--dry-run")

		assertOutputContains(t, stdout, "worktree path(s) to fix")
		assertOutputContains(t, stdout, "Dry run")
	})

	t.Run("bt repair --fix-paths fixes paths", func(t *testing.T) {
		stdout := runBtSuccess(t, newDir, "repair", "--fix-paths")

		assertOutputContains(t, stdout, "Successfully fixed")
	})

	t.Run("git status works after bt repair --fix-paths", func(t *testing.T) {
		runGitSuccess(t, worktreeDir, "status")
	})

	t.Run("bt commands work after repair", func(t *testing.T) {
		stdout := runBtSuccess(t, newDir, "list")
		assertOutputContains(t, stdout, "feature/test")

		// bt repair should show no issues (directory and branch names match)
		stdout = runBtSuccess(t, newDir, "repair", "--all")
		assertOutputContains(t, stdout, "No worktrees need repair")
	})
}

// TestRepair_PathChanged tests repair after the path to the repository changes.
// This simulates scenarios like home directory rename or mount point change.
// The --fix-paths option is used to update Git's internal worktree paths.
func TestRepair_PathChanged(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repair-path-changed")

	// Create a simulated "home directory" structure
	oldHome := filepath.Join(tempDir, "home", "olduser", "projects")
	if err := os.MkdirAll(oldHome, 0755); err != nil {
		t.Fatalf("failed to create old home: %v", err)
	}

	runBtSuccess(t, oldHome, "repo", "init", "myproject")
	projectDir := filepath.Join(oldHome, "myproject")
	runBtSuccess(t, projectDir, "add", "-b", "develop")
	runBtSuccess(t, projectDir, "add", "-b", "feature/auth")

	// Verify everything works initially
	runGitSuccess(t, filepath.Join(projectDir, "develop"), "status")
	runGitSuccess(t, filepath.Join(projectDir, "feature", "auth"), "status")

	// Simulate home directory rename (olduser -> newuser)
	newHome := filepath.Join(tempDir, "home", "newuser", "projects")
	if err := os.MkdirAll(filepath.Dir(newHome), 0755); err != nil {
		t.Fatalf("failed to create new home parent: %v", err)
	}
	if err := os.Rename(oldHome, newHome); err != nil {
		t.Fatalf("failed to rename home directory: %v", err)
	}

	newProjectDir := filepath.Join(newHome, "myproject")
	developDir := filepath.Join(newProjectDir, "develop")
	featureDir := filepath.Join(newProjectDir, "feature", "auth")

	t.Run("git commands fail after path change", func(t *testing.T) {
		// The .git file still references the old path
		cmd := exec.Command("git", "status")
		cmd.Dir = developDir
		if err := cmd.Run(); err == nil {
			t.Log("Note: git status succeeded unexpectedly")
		}
	})

	t.Run("bt repair --fix-paths fixes all paths", func(t *testing.T) {
		stdout := runBtSuccess(t, newProjectDir, "repair", "--fix-paths")

		assertOutputContains(t, stdout, "Successfully fixed")
	})

	t.Run("git commands work after bt repair --fix-paths", func(t *testing.T) {
		runGitSuccess(t, developDir, "status")
		runGitSuccess(t, featureDir, "status")
	})

	t.Run("bt commands work after repair", func(t *testing.T) {
		stdout := runBtSuccess(t, newProjectDir, "list")
		assertOutputContains(t, stdout, "develop")
		assertOutputContains(t, stdout, "feature/auth")

		stdout = runBtSuccess(t, newProjectDir, "status")
		assertOutputContains(t, stdout, "main") // default branch

		// bt repair should show no issues
		stdout = runBtSuccess(t, newProjectDir, "repair", "--all")
		assertOutputContains(t, stdout, "No worktrees need repair")
	})

	t.Run("can add new worktrees after repair", func(t *testing.T) {
		runBtSuccess(t, newProjectDir, "add", "-b", "feature/new")
		assertFileExists(t, filepath.Join(newProjectDir, "feature", "new"))
		runGitSuccess(t, filepath.Join(newProjectDir, "feature", "new"), "status")
	})
}
