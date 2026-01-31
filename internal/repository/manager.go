package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/amaya382/baretree/internal/config"
	"github.com/amaya382/baretree/internal/git"
)

// Manager handles repository-level operations
type Manager struct {
	Root     string
	BareDir  string
	Config   *config.Config
	Executor *git.Executor
}

// NewManager creates a new repository manager
func NewManager(repoRoot string) (*Manager, error) {
	cfg, err := config.LoadConfig(repoRoot)
	if err != nil {
		return nil, err
	}

	barePath := filepath.Join(repoRoot, config.BareDir)
	executor := git.NewExecutor(barePath)

	return &Manager{
		Root:     repoRoot,
		BareDir:  barePath,
		Config:   cfg,
		Executor: executor,
	}, nil
}

// InitializeBareRepo creates a baretree repository structure
// Configuration is stored in git-config instead of baretree.toml
func InitializeBareRepo(repoRoot, defaultBranch string) error {
	// Create repository root
	if err := os.MkdirAll(repoRoot, 0755); err != nil {
		return fmt.Errorf("failed to create repository root: %w", err)
	}

	// Initialize config in git-config
	if err := config.InitializeBaretreeConfig(repoRoot, defaultBranch); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	return nil
}

// GetMainWorktree returns the main (first) worktree
func (m *Manager) GetMainWorktree() (*git.Worktree, error) {
	output, err := m.Executor.Execute("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	worktrees := git.ParseWorktreeList(output)
	if len(worktrees) == 0 {
		return nil, fmt.Errorf("no worktrees found")
	}

	for i := range worktrees {
		if worktrees[i].IsMain {
			return &worktrees[i], nil
		}
	}

	// Fallback to first worktree
	return &worktrees[0], nil
}

// ExtractRepoName extracts repository name from URL
func ExtractRepoName(url string) string {
	// Remove trailing .git
	url = strings.TrimSuffix(url, ".git")

	// Extract last component
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return "repo"
}
