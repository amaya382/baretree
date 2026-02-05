package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRepoConfigRoot_Get tests getting the root directory
func TestRepoConfigRoot_Get(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repo-config-root-get")

	// Unset any existing BARETREE_ROOT to test git-config behavior
	origEnv := os.Getenv("BARETREE_ROOT")
	os.Unsetenv("BARETREE_ROOT")
	defer func() {
		if origEnv != "" {
			os.Setenv("BARETREE_ROOT", origEnv)
		}
	}()

	t.Run("get shows default when not configured", func(t *testing.T) {
		// First unset any git-config setting
		runGitConfigUnset(t, "baretree.root")

		stdout := runBtSuccess(t, tempDir, "repo", "config", "root")

		// Should show default path
		assertOutputContains(t, stdout, "baretree")
	})
}

// TestRepoConfigRoot_Set tests setting the root directory
func TestRepoConfigRoot_Set(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repo-config-root-set")
	newRoot := filepath.Join(tempDir, "my-repos")

	// Unset any existing BARETREE_ROOT
	origEnv := os.Getenv("BARETREE_ROOT")
	os.Unsetenv("BARETREE_ROOT")
	defer func() {
		if origEnv != "" {
			os.Setenv("BARETREE_ROOT", origEnv)
		}
		// Clean up git-config
		runGitConfigUnset(t, "baretree.root")
	}()

	t.Run("set root directory", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "repo", "config", "root", newRoot)

		assertOutputContains(t, stdout, "Root set to")
		assertOutputContains(t, stdout, newRoot)
	})

	t.Run("verify root was set", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "repo", "config", "root")

		assertOutputContains(t, stdout, newRoot)
	})

	t.Run("verify git config was updated", func(t *testing.T) {
		output := runGitConfigGet(t, "baretree.root")
		if !strings.Contains(output, newRoot) {
			t.Errorf("expected git config to contain %q, got %q", newRoot, output)
		}
	})
}

// TestRepoConfigRoot_SetSamePath tests setting the same path doesn't show change
func TestRepoConfigRoot_SetSamePath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repo-config-root-same")
	newRoot := filepath.Join(tempDir, "my-repos")

	// Unset any existing BARETREE_ROOT
	origEnv := os.Getenv("BARETREE_ROOT")
	os.Unsetenv("BARETREE_ROOT")
	defer func() {
		if origEnv != "" {
			os.Setenv("BARETREE_ROOT", origEnv)
		}
		runGitConfigUnset(t, "baretree.root")
	}()

	// Set root first
	runBtSuccess(t, tempDir, "repo", "config", "root", newRoot)

	t.Run("setting same path shows already set message", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "repo", "config", "root", newRoot)
		assertOutputContains(t, stdout, "Root is already")
	})
}

// TestRepoConfigRoot_Unset tests unsetting the root directory
func TestRepoConfigRoot_Unset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repo-config-root-unset")
	newRoot := filepath.Join(tempDir, "my-repos")

	// Unset any existing BARETREE_ROOT
	origEnv := os.Getenv("BARETREE_ROOT")
	os.Unsetenv("BARETREE_ROOT")
	defer func() {
		if origEnv != "" {
			os.Setenv("BARETREE_ROOT", origEnv)
		}
		runGitConfigUnset(t, "baretree.root")
	}()

	// Set root first
	runBtSuccess(t, tempDir, "repo", "config", "root", newRoot)

	t.Run("unset root directory", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "repo", "config", "root", "--unset")

		assertOutputContains(t, stdout, "Root setting removed")
		assertOutputContains(t, stdout, "~/baretree")
	})

	t.Run("verify root reverts to default", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "repo", "config", "root")

		assertOutputContains(t, stdout, "baretree")
	})
}

// TestRepoConfigRoot_UnsetWithArg tests error when using --unset with path argument
func TestRepoConfigRoot_UnsetWithArg(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repo-config-root-unset-arg")

	t.Run("unset with path argument fails", func(t *testing.T) {
		_, stderr := runBtExpectError(t, tempDir, "repo", "config", "root", "--unset", "/some/path")
		assertOutputContains(t, stderr, "cannot specify path with --unset flag")
	})
}

// TestRepoConfigRoot_EnvVarWarning tests warning when BARETREE_ROOT is set
func TestRepoConfigRoot_EnvVarWarning(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repo-config-root-env")
	envRoot := filepath.Join(tempDir, "env-root")

	// Set BARETREE_ROOT
	origEnv := os.Getenv("BARETREE_ROOT")
	os.Setenv("BARETREE_ROOT", envRoot)
	defer func() {
		if origEnv != "" {
			os.Setenv("BARETREE_ROOT", origEnv)
		} else {
			os.Unsetenv("BARETREE_ROOT")
		}
	}()

	t.Run("get shows env root", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "repo", "config", "root")

		assertOutputContains(t, stdout, envRoot)
	})

	t.Run("set shows warning about env var", func(t *testing.T) {
		newRoot := filepath.Join(tempDir, "new-root")
		stdout := runBtSuccess(t, tempDir, "repo", "config", "root", newRoot)

		assertOutputContains(t, stdout, "Warning: BARETREE_ROOT environment variable is set")
		assertOutputContains(t, stdout, "Root set to")
	})
}

// TestRepoConfigRoot_Help tests help output
func TestRepoConfigRoot_Help(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repo-config-root-help")

	t.Run("help shows usage", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "repo", "config", "root", "--help")

		assertOutputContains(t, stdout, "Get or set the root directory")
		assertOutputContains(t, stdout, "bt repo config root")
		assertOutputContains(t, stdout, "--unset")
		assertOutputContains(t, stdout, "--add")
		assertOutputContains(t, stdout, "--all")
	})
}

// TestRepoConfigRoot_Add tests adding root directories with --add flag
func TestRepoConfigRoot_Add(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repo-config-root-add")
	root1 := filepath.Join(tempDir, "root1")
	root2 := filepath.Join(tempDir, "root2")

	// Unset any existing BARETREE_ROOT
	origEnv := os.Getenv("BARETREE_ROOT")
	os.Unsetenv("BARETREE_ROOT")
	defer func() {
		if origEnv != "" {
			os.Setenv("BARETREE_ROOT", origEnv)
		}
		runGitConfigUnset(t, "baretree.root")
	}()

	t.Run("set first root", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "repo", "config", "root", root1)
		assertOutputContains(t, stdout, "Root set to")
		assertOutputContains(t, stdout, root1)
	})

	t.Run("add second root", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "repo", "config", "root", "--add", root2)
		assertOutputContains(t, stdout, "Root")
		assertOutputContains(t, stdout, "added")
		assertOutputContains(t, stdout, root2)
	})

	t.Run("verify both roots exist in git config", func(t *testing.T) {
		output := runGitConfigGetAll(t, "baretree.root")
		if !strings.Contains(output, root1) {
			t.Errorf("expected git config to contain %q, got %q", root1, output)
		}
		if !strings.Contains(output, root2) {
			t.Errorf("expected git config to contain %q, got %q", root2, output)
		}
	})

	t.Run("primary root is the last added", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "repo", "config", "root")
		assertOutputContains(t, stdout, root2)
	})

	t.Run("bt repo root --all shows all roots", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "repo", "root", "--all")
		assertOutputContains(t, stdout, root1)
		assertOutputContains(t, stdout, root2)
	})

	t.Run("bt repo config root --all shows all roots", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "repo", "config", "root", "--all")
		assertOutputContains(t, stdout, root1)
		assertOutputContains(t, stdout, root2)
	})
}

// TestRepoConfigRoot_AddSamePath tests that adding same path shows already exists
func TestRepoConfigRoot_AddSamePath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repo-config-root-add-same")
	root1 := filepath.Join(tempDir, "root1")

	// Unset any existing BARETREE_ROOT
	origEnv := os.Getenv("BARETREE_ROOT")
	os.Unsetenv("BARETREE_ROOT")
	defer func() {
		if origEnv != "" {
			os.Setenv("BARETREE_ROOT", origEnv)
		}
		runGitConfigUnset(t, "baretree.root")
	}()

	// Set first root
	runBtSuccess(t, tempDir, "repo", "config", "root", root1)

	t.Run("adding same path shows already exists", func(t *testing.T) {
		stdout := runBtSuccess(t, tempDir, "repo", "config", "root", "--add", root1)
		assertOutputContains(t, stdout, "already exists")
	})
}

// TestRepoConfigRoot_AddWithoutPath tests error when using --add without path
func TestRepoConfigRoot_AddWithoutPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repo-config-root-add-nopath")

	t.Run("add without path argument fails", func(t *testing.T) {
		_, stderr := runBtExpectError(t, tempDir, "repo", "config", "root", "--add")
		assertOutputContains(t, stderr, "--add requires a path argument")
	})
}

// TestRepoConfigRoot_AddUnsetConflict tests error when using --add and --unset together
func TestRepoConfigRoot_AddUnsetConflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repo-config-root-add-unset")

	t.Run("add with unset flag fails", func(t *testing.T) {
		_, stderr := runBtExpectError(t, tempDir, "repo", "config", "root", "--add", "--unset")
		assertOutputContains(t, stderr, "cannot use --unset and --add together")
	})
}

// TestRepoConfigRoot_AllConflict tests error when using --all with other flags or arguments
func TestRepoConfigRoot_AllConflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tempDir := createTempDir(t, "repo-config-root-all-conflict")

	t.Run("all with path argument fails", func(t *testing.T) {
		_, stderr := runBtExpectError(t, tempDir, "repo", "config", "root", "--all", "/some/path")
		assertOutputContains(t, stderr, "--all cannot be used with other flags or arguments")
	})

	t.Run("all with add flag fails", func(t *testing.T) {
		_, stderr := runBtExpectError(t, tempDir, "repo", "config", "root", "--all", "--add", "/some/path")
		assertOutputContains(t, stderr, "--all cannot be used with other flags or arguments")
	})

	t.Run("all with unset flag fails", func(t *testing.T) {
		_, stderr := runBtExpectError(t, tempDir, "repo", "config", "root", "--all", "--unset")
		assertOutputContains(t, stderr, "--all cannot be used with other flags or arguments")
	})
}

// Helper functions for git config operations
func runGitConfigUnset(t *testing.T, key string) {
	t.Helper()
	// Ignore error if key doesn't exist
	cmd := exec.Command("git", "config", "--global", "--unset-all", key)
	_ = cmd.Run()
}

func runGitConfigGet(t *testing.T, key string) string {
	t.Helper()
	cmd := exec.Command("git", "config", "--global", "--get", key)
	output, _ := cmd.Output()
	return strings.TrimSpace(string(output))
}

func runGitConfigGetAll(t *testing.T, key string) string {
	t.Helper()
	cmd := exec.Command("git", "config", "--global", "--get-all", key)
	output, _ := cmd.Output()
	return strings.TrimSpace(string(output))
}
