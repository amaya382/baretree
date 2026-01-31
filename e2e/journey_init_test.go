package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestJourney_Init tests initializing a new baretree repository from scratch
func TestJourney_Init(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "journey-init")

	// Step 1: Initialize new repository
	t.Run("init new repository", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "repo", "init", "my-new-project")

		assertOutputContains(t, stdout, "Successfully initialized")
		assertOutputContains(t, stdout, "my-new-project")

		// Verify structure
		projectDir := filepath.Join(tempDir, "my-new-project")
		assertFileExists(t, projectDir)
		assertFileExists(t, filepath.Join(projectDir, ".git"))
		assertFileExists(t, filepath.Join(projectDir, "main"))
	})

	projectDir := filepath.Join(tempDir, "my-new-project")

	// Step 2: Verify list works
	t.Run("list worktrees", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "list")

		assertOutputContains(t, stdout, "main")
	})

	// Step 3: Add feature branch
	t.Run("add feature branch", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "add", "-b", "feature/init-test")

		assertOutputContains(t, stdout, "Worktree created")
		assertFileExists(t, filepath.Join(projectDir, "feature", "init-test"))
	})

	// Step 4: Verify both worktrees exist
	t.Run("list shows both worktrees", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "list")

		assertOutputContains(t, stdout, "main")
		assertOutputContains(t, stdout, "feature/init-test")
	})

	// Step 5: Create a file in main worktree and commit
	t.Run("commit in main worktree", func(t *testing.T) {
		mainDir := filepath.Join(projectDir, "main")

		// Create a file
		testFile := filepath.Join(mainDir, "hello.txt")
		err := os.WriteFile(testFile, []byte("Hello, World!"), 0644)
		if err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		// Add and commit using git directly
		runGitSuccess(t, mainDir, "add", "hello.txt")
		runGitSuccess(t, mainDir, "commit", "-m", "Add hello.txt")

		// Verify file exists
		assertFileExists(t, testFile)
	})

	// Step 6: Status command works
	t.Run("status command", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "status")

		assertOutputContains(t, stdout, "my-new-project")
		assertOutputContains(t, stdout, "Worktrees")
	})
}

// TestInit_CustomBranch tests init with a custom default branch
func TestInit_CustomBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "init-custom-branch")

	t.Run("init with custom branch", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "repo", "init", "custom-project", "-b", "develop")

		assertOutputContains(t, stdout, "Successfully initialized")

		projectDir := filepath.Join(tempDir, "custom-project")
		assertFileExists(t, filepath.Join(projectDir, "develop"))

		// Verify git-config has correct default branch
		bareDir := filepath.Join(projectDir, ".git")
		defaultBranch := getGitConfig(t, bareDir, "baretree.defaultbranch")
		if defaultBranch != "develop" {
			t.Errorf("expected baretree.defaultbranch to be 'develop', got %q", defaultBranch)
		}
	})
}

// TestInit_InCurrentDirectory tests init in current directory
func TestInit_InCurrentDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "init-current-dir")
	projectDir := filepath.Join(tempDir, "my-project")
	_ = os.MkdirAll(projectDir, 0755)

	t.Run("init in current directory", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "repo", "init")

		assertOutputContains(t, stdout, "Successfully initialized")
		assertFileExists(t, filepath.Join(projectDir, ".git"))
		assertFileExists(t, filepath.Join(projectDir, "main"))
	})
}

// TestInit_WithExistingFiles tests init moves existing files to worktree
func TestInit_WithExistingFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "init-existing-files")
	projectDir := filepath.Join(tempDir, "my-project")
	_ = os.MkdirAll(projectDir, 0755)

	// Create some existing files (including hidden files)
	_ = os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("# My Project"), 0644)
	_ = os.MkdirAll(filepath.Join(projectDir, "src"), 0755)
	_ = os.WriteFile(filepath.Join(projectDir, "src", "main.go"), []byte("package main"), 0644)
	_ = os.WriteFile(filepath.Join(projectDir, ".env"), []byte("SECRET=123"), 0644)
	_ = os.MkdirAll(filepath.Join(projectDir, ".vscode"), 0755)
	_ = os.WriteFile(filepath.Join(projectDir, ".vscode", "settings.json"), []byte("{}"), 0644)

	t.Run("init moves existing files to worktree", func(t *testing.T) {
		stdout := runBtSuccess(t, projectDir, "repo", "init")

		assertOutputContains(t, stdout, "Successfully initialized")
		assertOutputContains(t, stdout, "Moving")
		assertOutputContains(t, stdout, "Moved files:")

		// Verify files are in worktree, not root
		assertFileNotExists(t, filepath.Join(projectDir, "README.md"))
		assertFileNotExists(t, filepath.Join(projectDir, "src"))
		assertFileNotExists(t, filepath.Join(projectDir, ".env"))
		assertFileNotExists(t, filepath.Join(projectDir, ".vscode"))

		// Verify files are in worktree
		assertFileExists(t, filepath.Join(projectDir, "main", "README.md"))
		assertFileExists(t, filepath.Join(projectDir, "main", "src", "main.go"))
		assertFileExists(t, filepath.Join(projectDir, "main", ".env"))
		assertFileExists(t, filepath.Join(projectDir, "main", ".vscode", "settings.json"))

		// Verify content is preserved
		content, err := os.ReadFile(filepath.Join(projectDir, "main", "README.md"))
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(content) != "# My Project" {
			t.Errorf("unexpected content: %s", content)
		}

		// Verify hidden file content
		envContent, err := os.ReadFile(filepath.Join(projectDir, "main", ".env"))
		if err != nil {
			t.Fatalf("failed to read .env: %v", err)
		}
		if string(envContent) != "SECRET=123" {
			t.Errorf("unexpected .env content: %s", envContent)
		}
	})

	t.Run("can commit moved files", func(t *testing.T) {
		mainDir := filepath.Join(projectDir, "main")
		runGitSuccess(t, mainDir, "add", ".")
		runGitSuccess(t, mainDir, "commit", "-m", "Add existing files")

		// Verify git log shows commit
		output := runGitSuccess(t, mainDir, "log", "--oneline")
		assertOutputContains(t, output, "Add existing files")
	})
}

// TestInit_ErrorCases tests error handling for init command
func TestInit_ErrorCases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	t.Run("init fails if already baretree repo", func(t *testing.T) {
		tempDir := createTempDir(t, "init-error-baretree")

		// First init
		runBtSuccess(t, tempDir, "repo", "init", "existing")

		// Second init should fail
		projectDir := filepath.Join(tempDir, "existing")
		_, stderr := runBtExpectError(t, projectDir, "repo", "init")

		assertOutputContains(t, stderr, "already a baretree repository")
	})

	t.Run("init fails if already git repo", func(t *testing.T) {
		tempDir := createTempDir(t, "init-error-git")

		// Create a regular git repo
		projectDir := filepath.Join(tempDir, "git-repo")
		_ = os.MkdirAll(projectDir, 0755)
		runGitSuccess(t, projectDir, "init")

		// Init should fail
		_, stderr := runBtExpectError(t, projectDir, "repo", "init")

		assertOutputContains(t, stderr, "already a git repository")
		assertOutputContains(t, stderr, "bt repo migrate")
	})
}

// getGitConfig gets a git config value from the bare repository
func getGitConfig(t *testing.T, bareDir, key string) string {
	t.Helper()
	cmd := exec.Command("git", "config", "--file", filepath.Join(bareDir, "config"), "--get", key)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
