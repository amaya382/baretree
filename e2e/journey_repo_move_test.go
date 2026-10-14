package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupRepoMoveRoot(t *testing.T, prefix string) (root string, env map[string]string, repoDir string) {
	t.Helper()

	tempDir := createTempDir(t, prefix)
	root = filepath.Join(tempDir, "root")
	env = map[string]string{"BARETREE_ROOT": root}

	parent := filepath.Join(root, "github.com", "testuser")
	if err := os.MkdirAll(parent, 0755); err != nil {
		t.Fatalf("failed to create root: %v", err)
	}
	runBtSuccess(t, parent, "repo", "init", "old-name")
	repoDir = filepath.Join(parent, "old-name")
	runBtSuccess(t, repoDir, "add", "-b", "feature/test")

	return root, env, repoDir
}

func runBtWithEnvSuccess(t *testing.T, workDir string, env map[string]string, args ...string) string {
	t.Helper()

	stdout, stderr, err := runBtWithEnv(t, workDir, env, args...)
	if err != nil {
		t.Fatalf("bt %s failed: %v\nstdout: %s\nstderr: %s", strings.Join(args, " "), err, stdout, stderr)
	}
	return stdout
}

func runBtWithEnvFailure(t *testing.T, workDir string, env map[string]string, args ...string) string {
	t.Helper()

	stdout, stderr, err := runBtWithEnv(t, workDir, env, args...)
	if err == nil {
		t.Fatalf("bt %s should have failed\nstdout: %s", strings.Join(args, " "), stdout)
	}
	return stderr
}

// TestRepoMove_DestinationForm tests that a relative destination replaces the same part as the source query
func TestRepoMove_DestinationForm(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	tests := []struct {
		name   string
		query  string
		dest   string
		newRel string
	}{
		{"name only renames the repository", "old-name", "new-name", "github.com/testuser/new-name"},
		{"owner/name moves to another owner", "testuser/old-name", "otheruser/new-name", "github.com/otheruser/new-name"},
		{"full path is relative to the root", "github.com/testuser/old-name", "gitlab.com/group/new-name", "gitlab.com/group/new-name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, env, repoDir := setupRepoMoveRoot(t, "repo-move-form")
			newDir := filepath.Join(root, filepath.FromSlash(tt.newRel))

			stdout := runBtWithEnvSuccess(t, root, env, "repo", "move", tt.query, tt.dest)
			assertOutputContains(t, stdout, "Repository moved")
			assertFileNotExists(t, repoDir)

			runGitSuccess(t, filepath.Join(newDir, "main"), "status")
			stdout = runGitSuccess(t, filepath.Join(newDir, "feature", "test"), "status")
			assertOutputContains(t, stdout, "On branch feature/test")

			stdout = runBtSuccess(t, newDir, "repair", "--fix-paths", "--dry-run")
			assertOutputContains(t, stdout, "No worktree paths need fixing")

			stdout = runBtWithEnvSuccess(t, root, env, "repo", "list")
			assertOutputContains(t, stdout, tt.newRel)
		})
	}
}

// TestRepoMove_CleansUpEmptyParents tests that empty parent directories of the source are removed
func TestRepoMove_CleansUpEmptyParents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	root, env, repoDir := setupRepoMoveRoot(t, "repo-move-cleanup")

	runBtWithEnvSuccess(t, root, env, "repo", "move", "testuser/old-name", "otheruser/old-name")

	assertFileNotExists(t, filepath.Dir(repoDir))
	assertFileExists(t, filepath.Join(root, "github.com"))
}

// TestRepoMove_AbsolutePathAndAlias tests the mv alias with an absolute destination outside the root
func TestRepoMove_AbsolutePathAndAlias(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	root, env, _ := setupRepoMoveRoot(t, "repo-move-abs")
	newDir := filepath.Join(filepath.Dir(root), "outside", "moved")

	stdout := runBtWithEnvSuccess(t, root, env, "repo", "mv", "testuser/old-name", newDir)
	assertOutputContains(t, stdout, "outside the baretree root")

	runGitSuccess(t, filepath.Join(newDir, "feature", "test"), "status")
}

// TestRepoMove_ExternalWorktree tests that a worktree outside the repository keeps working after a move
func TestRepoMove_ExternalWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	root, env, repoDir := setupRepoMoveRoot(t, "repo-move-external")
	externalDir := filepath.Join(filepath.Dir(root), "external")
	cmd := exec.Command("git", "worktree", "add", "-b", "ext", externalDir)
	cmd.Dir = filepath.Join(repoDir, ".git")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to add external worktree: %v\n%s", err, out)
	}

	// A different depth makes the external worktree unreachable via the old relative position
	runBtWithEnvSuccess(t, root, env, "repo", "move", "github.com/testuser/old-name", "new-name")

	stdout := runGitSuccess(t, externalDir, "status")
	assertOutputContains(t, stdout, "On branch ext")
}

// TestRepoMove_ErrorCases tests that invalid moves are rejected without touching the repository
func TestRepoMove_ErrorCases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	root, env, repoDir := setupRepoMoveRoot(t, "repo-move-errors")
	parent := filepath.Join(root, "github.com", "testuser")
	runBtSuccess(t, parent, "repo", "init", "other")

	const fullPath = "github.com/testuser/old-name"

	tests := []struct {
		name    string
		workDir string
		query   string
		dest    string
		wantErr string
	}{
		{"destination exists", root, "old-name", "other", "destination already exists"},
		{"destination inside source", root, fullPath, "github.com/testuser/old-name/sub", "overlap"},
		{"destination inside another repo", root, fullPath, "github.com/testuser/other/nested", "inside another baretree repository"},
		{"inside the source repository", filepath.Join(repoDir, "main"), "old-name", "new-name", "while inside it"},
		{"name source with owner/name destination", root, "old-name", "otheruser/new-name", "must be a repository name"},
		{"owner/name source with name destination", root, "testuser/old-name", "new-name", "must be an organization/repository"},
		{"parent directory component", root, "old-name", "..", "invalid destination"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stderr := runBtWithEnvFailure(t, tt.workDir, env, "repo", "move", tt.query, tt.dest)
			assertOutputContains(t, stderr, tt.wantErr)
			runGitSuccess(t, filepath.Join(repoDir, "feature", "test"), "status")
		})
	}
}
