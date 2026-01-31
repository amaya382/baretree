package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Executor executes git commands
type Executor struct {
	workDir string
}

// NewExecutor creates a new git command executor
func NewExecutor(workDir string) *Executor {
	return &Executor{workDir: workDir}
}

// Execute runs a git command and returns the output
func (e *Executor) Execute(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if e.workDir != "" {
		cmd.Dir = e.workDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w\nstderr: %s", strings.Join(args, " "), err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// ExecuteWithStderr runs a git command and returns both stdout and stderr
func (e *Executor) ExecuteWithStderr(args ...string) (stdout, stderr string, err error) {
	cmd := exec.Command("git", args...)
	if e.workDir != "" {
		cmd.Dir = e.workDir
	}

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	return strings.TrimSpace(outBuf.String()), strings.TrimSpace(errBuf.String()), err
}

// IsBareRepo checks if the given directory is a bare git repository
func IsBareRepo(dir string) bool {
	executor := NewExecutor(dir)
	output, err := executor.Execute("rev-parse", "--is-bare-repository")
	if err != nil {
		return false
	}
	return output == "true"
}

// Clone executes git clone with the given arguments
func Clone(args ...string) error {
	cmd := exec.Command("git", append([]string{"clone"}, args...)...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stderr // Show clone progress

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w\n%s", err, stderr.String())
	}

	return nil
}

// ErrGitUserNotConfigured is returned when git user.name or user.email is not set
type ErrGitUserNotConfigured struct {
	MissingName  bool
	MissingEmail bool
}

func (e *ErrGitUserNotConfigured) Error() string {
	var missing []string
	if e.MissingName {
		missing = append(missing, "user.name")
	}
	if e.MissingEmail {
		missing = append(missing, "user.email")
	}
	return fmt.Sprintf("git %s not configured. Please run:\n  git config --global user.name \"Your Name\"\n  git config --global user.email \"your@email.com\"",
		strings.Join(missing, " and "))
}

// CheckUserConfig checks if git user.name and user.email are configured
func CheckUserConfig() error {
	executor := NewExecutor("")

	var missingName, missingEmail bool

	name, err := executor.Execute("config", "user.name")
	if err != nil || name == "" {
		missingName = true
	}

	email, err := executor.Execute("config", "user.email")
	if err != nil || email == "" {
		missingEmail = true
	}

	if missingName || missingEmail {
		return &ErrGitUserNotConfigured{
			MissingName:  missingName,
			MissingEmail: missingEmail,
		}
	}

	return nil
}
