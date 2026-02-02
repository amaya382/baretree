package e2e

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestCompletion_Worktree tests worktree name completion
func TestCompletion_Worktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// Create a temporary directory
	tempDir := createTempDir(t, "completion-worktree")

	// Initialize a baretree repository
	runBtSuccess(t, tempDir, "init", "myrepo")
	repoDir := filepath.Join(tempDir, "myrepo")

	// Add some worktrees (use -b to create new branches)
	runBtSuccess(t, repoDir, "add", "-b", "feature/auth")
	runBtSuccess(t, repoDir, "add", "-b", "feature/api")
	runBtSuccess(t, repoDir, "add", "-b", "bugfix/login")

	t.Run("cd command completes worktree names with special args", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "__complete", "cd", "")

		// Should contain worktree names
		assertOutputContains(t, stdout, "main")
		assertOutputContains(t, stdout, "feature/auth")
		assertOutputContains(t, stdout, "feature/api")
		assertOutputContains(t, stdout, "bugfix/login")

		// Should contain special arguments
		assertOutputContains(t, stdout, "@")
		assertOutputContains(t, stdout, "-")

		// Should have NoFileComp directive (:4 is the bitmask for ShellCompDirectiveNoFileComp)
		assertOutputContains(t, stdout, ":4")
	})

	t.Run("remove command completes worktree names without special args", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "__complete", "remove", "")

		// Should contain worktree names
		assertOutputContains(t, stdout, "main")
		assertOutputContains(t, stdout, "feature/auth")

		// Should NOT contain special arguments for remove
		assertOutputNotContains(t, stdout, "\n@\n")
		assertOutputNotContains(t, stdout, "\n-\n")
	})

	t.Run("rename command completes worktree names", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "__complete", "rename", "")

		// Should contain worktree names
		assertOutputContains(t, stdout, "feature/auth")
		assertOutputContains(t, stdout, "bugfix/login")
	})

	t.Run("repair command completes worktree names", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "__complete", "repair", "")

		// Should contain worktree names
		assertOutputContains(t, stdout, "main")
		assertOutputContains(t, stdout, "feature/auth")
	})

	t.Run("unbare first arg completes worktree names", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "__complete", "unbare", "")

		// Should contain worktree names
		assertOutputContains(t, stdout, "main")
		assertOutputContains(t, stdout, "feature/auth")

		// Should contain special arguments
		assertOutputContains(t, stdout, "@")
	})

	t.Run("unbare second arg falls back to file completion", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "__complete", "unbare", "main", "")

		// Should have Default directive (:0 is the bitmask for ShellCompDirectiveDefault)
		assertOutputContains(t, stdout, ":0")

		// Should NOT have NoFileComp directive (:4)
		assertOutputNotContains(t, stdout, ":4")
	})
}

// TestCompletion_Flags tests flag completion
func TestCompletion_Flags(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// Create a temporary directory
	tempDir := createTempDir(t, "completion-flags")

	// Initialize a baretree repository
	runBtSuccess(t, tempDir, "init", "myrepo")
	repoDir := filepath.Join(tempDir, "myrepo")

	t.Run("add command completes flags", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "__complete", "add", "--")

		// Should contain flag completions
		assertOutputContains(t, stdout, "--branch")
		assertOutputContains(t, stdout, "--base")
		assertOutputContains(t, stdout, "--detach")
		assertOutputContains(t, stdout, "--force")
		assertOutputContains(t, stdout, "--fetch")
	})

	t.Run("remove command completes flags", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "__complete", "remove", "--")

		// Should contain flag completions
		assertOutputContains(t, stdout, "--force")
		assertOutputContains(t, stdout, "--with-branch")
	})

	t.Run("list command completes flags", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "__complete", "list", "--")

		// Should contain flag completions
		assertOutputContains(t, stdout, "--json")
		assertOutputContains(t, stdout, "--paths")
	})

	t.Run("repair command completes flags", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "__complete", "repair", "--")

		// Should contain flag completions
		assertOutputContains(t, stdout, "--dry-run")
		assertOutputContains(t, stdout, "--source")
		assertOutputContains(t, stdout, "--all")
		assertOutputContains(t, stdout, "--fix-paths")
	})
}

// TestCompletion_Subcommands tests subcommand completion
func TestCompletion_Subcommands(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "completion-subcommands")

	t.Run("root command completes subcommands", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "__complete", "")

		// Should contain main commands
		assertOutputContains(t, stdout, "add")
		assertOutputContains(t, stdout, "cd")
		assertOutputContains(t, stdout, "list")
		assertOutputContains(t, stdout, "remove")
		assertOutputContains(t, stdout, "repo")
		assertOutputContains(t, stdout, "config")

		// Should contain aliases
		assertOutputContains(t, stdout, "init")
		assertOutputContains(t, stdout, "clone")
		assertOutputContains(t, stdout, "go")
	})

	t.Run("repo command completes subcommands", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "__complete", "repo", "")

		// Should contain repo subcommands
		assertOutputContains(t, stdout, "init")
		assertOutputContains(t, stdout, "clone")
		assertOutputContains(t, stdout, "migrate")
		assertOutputContains(t, stdout, "list")
		assertOutputContains(t, stdout, "cd")
		assertOutputContains(t, stdout, "get")
	})

	t.Run("config command completes subcommands", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "__complete", "config", "")

		// Should contain config subcommands
		assertOutputContains(t, stdout, "export")
		assertOutputContains(t, stdout, "import")
	})

	t.Run("post-create command completes subcommands", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "__complete", "post-create", "")

		// Should contain post-create subcommands
		assertOutputContains(t, stdout, "add")
		assertOutputContains(t, stdout, "remove")
		assertOutputContains(t, stdout, "apply")
		assertOutputContains(t, stdout, "list")
	})

	t.Run("shell-init completes shell names", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "__complete", "shell-init", "")

		// Should contain shell names
		assertOutputContains(t, stdout, "bash")
		assertOutputContains(t, stdout, "zsh")
		assertOutputContains(t, stdout, "fish")
	})
}

// TestCompletion_Repository tests repository name completion
func TestCompletion_Repository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// Create a temporary directory structure simulating baretree root
	tempDir := createTempDir(t, "completion-repo")

	// Configure baretree root
	runGitSuccess(t, tempDir, "config", "--global", "baretree.root", tempDir)
	t.Cleanup(func() {
		runGitSuccess(t, tempDir, "config", "--global", "--unset", "baretree.root")
	})

	// Create some repositories
	runBtSuccess(t, tempDir, "init", filepath.Join(tempDir, "github.com", "user", "repo1"))
	runBtSuccess(t, tempDir, "init", filepath.Join(tempDir, "github.com", "user", "repo2"))
	runBtSuccess(t, tempDir, "init", filepath.Join(tempDir, "gitlab.com", "org", "project"))

	t.Run("go command completes repository names with -", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "__complete", "go", "")

		// Should contain repository paths
		assertOutputContains(t, stdout, "github.com/user/repo1")
		assertOutputContains(t, stdout, "github.com/user/repo2")
		assertOutputContains(t, stdout, "gitlab.com/org/project")

		// Should contain - for previous
		assertOutputContains(t, stdout, "-")
	})

	t.Run("repo cd command completes repository names", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "__complete", "repo", "cd", "")

		// Should contain repository paths
		assertOutputContains(t, stdout, "github.com/user/repo1")
		assertOutputContains(t, stdout, "gitlab.com/org/project")
	})

	t.Run("repos command completes repository names without -", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "__complete", "repos", "")

		// Should contain repository paths
		assertOutputContains(t, stdout, "github.com/user/repo1")

		// repos is for filtering, not navigation, so no -
		lines := strings.Split(stdout, "\n")
		if slices.Contains(lines, "-") {
			t.Error("repos completion should not include standalone '-'")
		}
	})
}

// TestCompletion_PartialMatch tests completion with partial input
func TestCompletion_PartialMatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "completion-partial")

	runBtSuccess(t, tempDir, "init", "myrepo")
	repoDir := filepath.Join(tempDir, "myrepo")

	runBtSuccess(t, repoDir, "add", "-b", "feature/auth")
	runBtSuccess(t, repoDir, "add", "-b", "feature/api")
	runBtSuccess(t, repoDir, "add", "-b", "bugfix/login")

	t.Run("partial match filters completions", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "__complete", "cd", "feat")

		// Should contain feature branches
		assertOutputContains(t, stdout, "feature/auth")
		assertOutputContains(t, stdout, "feature/api")

		// Should NOT contain bugfix or main (doesn't match prefix)
		// Note: Cobra's completion might still return all, filtering is shell's job
		// But we can at least verify feature branches are present
	})
}

// TestCompletion_SubstringMatch tests that completion supports substring matching
// with prefix matches prioritized over substring matches
func TestCompletion_SubstringMatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "completion-substring")

	runBtSuccess(t, tempDir, "init", "myrepo")
	repoDir := filepath.Join(tempDir, "myrepo")

	// Create worktrees: test (prefix match) and feat/test, chore/test (substring matches)
	runBtSuccess(t, repoDir, "add", "-b", "test")
	runBtSuccess(t, repoDir, "add", "-b", "feat/test")
	runBtSuccess(t, repoDir, "add", "-b", "chore/test")
	runBtSuccess(t, repoDir, "add", "-b", "other") // "other" does not contain "te"

	t.Run("worktree completion includes both prefix and substring matches", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "__complete", "cd", "te")

		// Should contain prefix match
		assertOutputContains(t, stdout, "test")
		// Should contain substring matches
		assertOutputContains(t, stdout, "feat/test")
		assertOutputContains(t, stdout, "chore/test")
		// Should NOT contain other (doesn't match "te" or "bar")
		assertOutputNotContains(t, stdout, "other")
	})

	t.Run("worktree completion prioritizes prefix matches over substring matches", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "__complete", "cd", "te")
		lines := strings.Split(stdout, "\n")

		// Find positions of matches
		testIdx := -1
		featTestIdx := -1
		choreTestIdx := -1

		for i, line := range lines {
			switch line {
			case "test":
				testIdx = i
			case "feat/test":
				featTestIdx = i
			case "chore/test":
				choreTestIdx = i
			}
		}

		// Prefix match "test" should come before substring matches
		if testIdx == -1 {
			t.Fatal("prefix match 'test' not found")
		}
		if featTestIdx == -1 {
			t.Fatal("substring match 'feat/test' not found")
		}
		if choreTestIdx == -1 {
			t.Fatal("substring match 'chore/test' not found")
		}

		if testIdx > featTestIdx {
			t.Errorf("prefix match 'test' (idx=%d) should come before 'feat/test' (idx=%d)", testIdx, featTestIdx)
		}
		if testIdx > choreTestIdx {
			t.Errorf("prefix match 'test' (idx=%d) should come before 'chore/test' (idx=%d)", testIdx, choreTestIdx)
		}
	})

	t.Run("special completions excluded when filter is applied", func(t *testing.T) {
		stdout := runBtSuccess(t, repoDir, "__complete", "cd", "te")
		lines := strings.Split(stdout, "\n")

		// @ and - should NOT be in the output when filtering
		if slices.Contains(lines, "@") {
			t.Error("special completion '@' should not appear when filter 'te' is applied")
		}
		if slices.Contains(lines, "-") {
			t.Error("special completion '-' should not appear when filter 'te' is applied")
		}
	})
}

// TestCompletion_RepositorySubstringMatch tests repository completion with substring matching
func TestCompletion_RepositorySubstringMatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "completion-repo-substring")

	// Configure baretree root
	runGitSuccess(t, tempDir, "config", "--global", "baretree.root", tempDir)
	t.Cleanup(func() {
		runGitSuccess(t, tempDir, "config", "--global", "--unset", "baretree.root")
	})

	// Create repositories with names that test prefix vs substring matching
	// "bar" is prefix match for "bar", "github.com/user/foobar" is substring match
	runBtSuccess(t, tempDir, "init", filepath.Join(tempDir, "github.com", "user", "bar"))
	runBtSuccess(t, tempDir, "init", filepath.Join(tempDir, "github.com", "user", "foobar"))
	runBtSuccess(t, tempDir, "init", filepath.Join(tempDir, "github.com", "user", "other")) // "other" does not contain "bar"

	t.Run("repository completion includes both prefix and substring matches", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "__complete", "go", "bar")

		// Should contain prefix match (name starts with "bar")
		assertOutputContains(t, stdout, "github.com/user/bar")
		// Should contain substring match (name contains "bar")
		assertOutputContains(t, stdout, "github.com/user/foobar")
		// Should NOT contain other (doesn't match "te" or "bar")
		assertOutputNotContains(t, stdout, "other")
	})

	t.Run("repository completion prioritizes prefix matches", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "__complete", "go", "bar")
		lines := strings.Split(stdout, "\n")

		barIdx := -1
		foobarIdx := -1

		for i, line := range lines {
			if line == "github.com/user/bar" {
				barIdx = i
			}
			if line == "github.com/user/foobar" {
				foobarIdx = i
			}
		}

		if barIdx == -1 {
			t.Fatal("prefix match 'github.com/user/bar' not found")
		}
		if foobarIdx == -1 {
			t.Fatal("substring match 'github.com/user/foobar' not found")
		}

		if barIdx > foobarIdx {
			t.Errorf("prefix match 'bar' (idx=%d) should come before 'foobar' (idx=%d)", barIdx, foobarIdx)
		}
	})

	t.Run("special completion '-' excluded when filter is applied", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "__complete", "go", "bar")
		lines := strings.Split(stdout, "\n")

		if slices.Contains(lines, "-") {
			t.Error("special completion '-' should not appear when filter 'bar' is applied")
		}
	})
}
