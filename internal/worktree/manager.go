package worktree

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/amaya382/baretree/internal/config"
	"github.com/amaya382/baretree/internal/git"
)

// Manager handles worktree operations
type Manager struct {
	RepoRoot string
	BareDir  string
	Config   *config.Config
	Executor *git.Executor
}

// NewManager creates a new worktree manager
func NewManager(repoRoot, bareDir string, cfg *config.Config) *Manager {
	return &Manager{
		RepoRoot: repoRoot,
		BareDir:  bareDir,
		Config:   cfg,
		Executor: git.NewExecutor(bareDir),
	}
}

// ErrWorktreeAlreadyExists is returned when trying to add a worktree for a branch that already has one
type ErrWorktreeAlreadyExists struct {
	BranchName   string
	WorktreePath string
}

func (e *ErrWorktreeAlreadyExists) Error() string {
	return fmt.Sprintf("branch '%s' is already checked out at '%s'", e.BranchName, e.WorktreePath)
}

// ErrBranchNotFound is returned when the specified branch doesn't exist
type ErrBranchNotFound struct {
	BranchName string
}

func (e *ErrBranchNotFound) Error() string {
	return fmt.Sprintf("branch '%s' not found locally or on any remote", e.BranchName)
}

// AddOptions contains options for adding a worktree
type AddOptions struct {
	NewBranch  bool   // Create a new branch
	BaseBranch string // Base branch for new branch
	TrackRef   string // Remote ref to track (e.g., "origin/feature/x")
}

// Add creates a new worktree
func (m *Manager) Add(branchName string, newBranch bool, baseBranch string) (string, error) {
	return m.AddWithOptions(branchName, AddOptions{
		NewBranch:  newBranch,
		BaseBranch: baseBranch,
	})
}

// AddWithOptions creates a new worktree with extended options
func (m *Manager) AddWithOptions(branchName string, opts AddOptions) (string, error) {
	// Construct worktree path from branch name
	// feature/auth -> {repoRoot}/feature/auth
	worktreePath := filepath.Join(m.RepoRoot, branchName)

	// Check if the branch already has a worktree
	if !opts.NewBranch {
		worktrees, err := m.List()
		if err == nil {
			for _, wt := range worktrees {
				if wt.Branch == branchName {
					return "", &ErrWorktreeAlreadyExists{
						BranchName:   branchName,
						WorktreePath: wt.Path,
					}
				}
			}
		}
	}

	// Build git worktree add command
	args := []string{"worktree", "add"}

	if opts.NewBranch {
		args = append(args, "-b", branchName)
	} else if opts.TrackRef != "" {
		// Create tracking branch: git worktree add -b <branch> <path> <remote-ref>
		args = append(args, "-b", branchName)
	}

	args = append(args, worktreePath)

	if opts.NewBranch && opts.BaseBranch != "" {
		args = append(args, opts.BaseBranch)
	} else if opts.TrackRef != "" {
		args = append(args, opts.TrackRef)
	} else if !opts.NewBranch {
		args = append(args, branchName)
	}

	// Execute git worktree add
	if _, err := m.Executor.Execute(args...); err != nil {
		return "", fmt.Errorf("failed to add worktree: %w", err)
	}

	// Apply shared file configuration
	if err := m.ApplySharedConfig(worktreePath); err != nil {
		return "", fmt.Errorf("failed to apply shared config: %w", err)
	}

	return worktreePath, nil
}

// Remove removes a worktree
func (m *Manager) Remove(worktreePath string, force bool) error {
	args := []string{"worktree", "remove"}

	if force {
		args = append(args, "--force")
	}

	args = append(args, worktreePath)

	if _, err := m.Executor.Execute(args...); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	return nil
}

// List returns all worktrees (excluding the bare repository itself)
func (m *Manager) List() ([]git.Worktree, error) {
	output, err := m.Executor.Execute("worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	allWorktrees := git.ParseWorktreeList(output)

	// Get the default branch from config
	defaultBranch := m.Config.Repository.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	// Filter out the bare repository and mark the default branch worktree
	var worktrees []git.Worktree
	for _, wt := range allWorktrees {
		// Skip bare repository
		if wt.IsBare {
			continue
		}

		// Mark as main/default if the branch matches the default branch
		wt.IsMain = (wt.Branch == defaultBranch)

		worktrees = append(worktrees, wt)
	}

	return worktrees, nil
}

// Fetch fetches from remotes
func (m *Manager) Fetch(remote string) error {
	return m.Executor.Fetch(remote)
}

// ResolveBranch resolves a branch specification
func (m *Manager) ResolveBranch(branchSpec string) (*git.BranchInfo, error) {
	return m.Executor.ResolveBranch(branchSpec)
}

// IsManaged checks if a worktree is managed by baretree
// A managed worktree must be within the repository root and not inside another worktree
func (m *Manager) IsManaged(worktreePath string) bool {
	// Check if worktree is within repository root
	relPath, err := filepath.Rel(m.RepoRoot, worktreePath)
	if err != nil {
		return false
	}

	// If path goes up (..), it's outside repo root
	if strings.HasPrefix(relPath, "..") {
		return false
	}

	// Check if it's not the bare directory
	if worktreePath == m.BareDir || strings.HasPrefix(relPath, config.BareDir) {
		return false
	}

	return true
}

// IsNestedInWorktree checks if a worktree path is nested inside another worktree
func (m *Manager) IsNestedInWorktree(worktreePath string, allWorktrees []string) bool {
	for _, otherPath := range allWorktrees {
		if otherPath == worktreePath {
			continue
		}
		// Check if worktreePath is inside otherPath
		rel, err := filepath.Rel(otherPath, worktreePath)
		if err != nil {
			continue
		}
		// If rel doesn't start with "..", worktreePath is inside otherPath
		if !strings.HasPrefix(rel, "..") {
			return true
		}
	}
	return false
}
