package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRepo is the repository URL used for e2e tests
const TestRepo = "https://github.com/amaya382/baretree"

// btBinary is the path to the compiled bt binary
var btBinary string

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Build the binary before running tests
	if err := buildBinary(); err != nil {
		panic("failed to build bt binary: " + err.Error())
	}

	// Run tests
	code := m.Run()

	// Cleanup
	os.Remove(btBinary)

	os.Exit(code)
}

// buildBinary compiles the bt binary for testing
func buildBinary() error {
	// Get the project root (parent of e2e directory)
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	projectRoot := filepath.Dir(wd)

	// Build binary in temp location
	btBinary = filepath.Join(os.TempDir(), "bt-e2e-test")

	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", btBinary, "./cmd/bt")
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// runBt executes the bt command with the given arguments
func runBt(t *testing.T, workDir string, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	cmd := exec.Command(btBinary, args...)
	cmd.Dir = workDir

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// runBtSuccess runs bt and expects it to succeed
func runBtSuccess(t *testing.T, workDir string, args ...string) string {
	t.Helper()

	stdout, stderr, err := runBt(t, workDir, args...)
	if err != nil {
		t.Fatalf("bt %s failed: %v\nstdout: %s\nstderr: %s",
			strings.Join(args, " "), err, stdout, stderr)
	}
	return stdout
}

// runBtFailure runs bt and expects it to fail
func runBtFailure(t *testing.T, workDir string, args ...string) (stdout, stderr string) {
	t.Helper()

	stdout, stderr, err := runBt(t, workDir, args...)
	if err == nil {
		t.Fatalf("bt %s expected to fail but succeeded\nstdout: %s",
			strings.Join(args, " "), stdout)
	}
	return stdout, stderr
}

// createTempDir creates a temporary directory for testing
func createTempDir(t *testing.T, prefix string) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "baretree-e2e-"+prefix+"-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	return dir
}

// assertFileExists checks that a file or directory exists
func assertFileExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected %s to exist, but it doesn't", path)
	}
}

// assertFileNotExists checks that a file or directory does not exist
func assertFileNotExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err == nil {
		t.Errorf("expected %s to not exist, but it does", path)
	}
}

// assertIsSymlink checks that a path is a symlink
func assertIsSymlink(t *testing.T, path string) {
	t.Helper()

	info, err := os.Lstat(path)
	if err != nil {
		t.Errorf("failed to stat %s: %v", path, err)
		return
	}

	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected %s to be a symlink, but it's not", path)
	}
}

// assertOutputContains checks that output contains expected string
func assertOutputContains(t *testing.T, output, expected string) {
	t.Helper()

	if !strings.Contains(output, expected) {
		t.Errorf("expected output to contain %q, but got:\n%s", expected, output)
	}
}

// assertOutputNotContains checks that output does not contain a string
func assertOutputNotContains(t *testing.T, output, notExpected string) {
	t.Helper()

	if strings.Contains(output, notExpected) {
		t.Errorf("expected output to NOT contain %q, but got:\n%s", notExpected, output)
	}
}

// isDirectory checks if a path is a directory
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// runGitSuccess runs git command and expects it to succeed
func runGitSuccess(t *testing.T, workDir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = workDir

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s failed: %v\nstdout: %s\nstderr: %s",
			strings.Join(args, " "), err, outBuf.String(), errBuf.String())
	}
	return outBuf.String()
}

// runBtExpectError runs bt and expects it to fail, returns stdout and stderr
func runBtExpectError(t *testing.T, workDir string, args ...string) (stdout, stderr string) {
	t.Helper()

	stdout, stderr, err := runBt(t, workDir, args...)
	if err == nil {
		t.Fatalf("bt %s expected to fail but succeeded\nstdout: %s",
			strings.Join(args, " "), stdout)
	}
	return stdout, stderr
}

// assertSymlinkIsRelative checks that a symlink target is a relative path
func assertSymlinkIsRelative(t *testing.T, path string) {
	t.Helper()

	target, err := os.Readlink(path)
	if err != nil {
		t.Errorf("failed to read symlink %s: %v", path, err)
		return
	}

	if filepath.IsAbs(target) {
		t.Errorf("expected symlink %s to have relative target, but got absolute path: %s", path, target)
	}
}
