package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Git config keys for baretree
const (
	GitConfigSection          = "baretree"
	GitConfigKeyBareDir       = "baretree.baredir"
	GitConfigKeyDefaultBranch = "baretree.defaultbranch"
	GitConfigKeyShared        = "baretree.shared"
)

// LoadConfigFromGit loads configuration from git-config in the bare repository
func LoadConfigFromGit(repoRoot string) (*Config, error) {
	// First, find the bare directory by checking common locations
	bareDir := findBareDir(repoRoot)
	if bareDir == "" {
		return nil, fmt.Errorf("bare repository not found in %s", repoRoot)
	}

	cfg := &Config{
		Repository: Repository{
			BareDir: filepath.Base(bareDir),
		},
		Shared: []Shared{},
	}

	// Read config values
	if defaultBranch, err := gitConfigGet(bareDir, GitConfigKeyDefaultBranch); err == nil && defaultBranch != "" {
		cfg.Repository.DefaultBranch = defaultBranch
	} else {
		cfg.Repository.DefaultBranch = "main"
	}

	// Read shared entries
	sharedEntries, err := gitConfigGetAll(bareDir, GitConfigKeyShared)
	if err == nil {
		for _, entry := range sharedEntries {
			if shared, err := parseSharedEntry(entry); err == nil {
				cfg.Shared = append(cfg.Shared, shared)
			}
		}
	}

	return cfg, nil
}

// SaveConfigToGit saves configuration to git-config in the bare repository
func SaveConfigToGit(repoRoot string, cfg *Config) error {
	bareDir := filepath.Join(repoRoot, cfg.Repository.BareDir)

	// Save basic config
	if err := gitConfigSet(bareDir, GitConfigKeyBareDir, cfg.Repository.BareDir); err != nil {
		return fmt.Errorf("failed to set baredir: %w", err)
	}

	if err := gitConfigSet(bareDir, GitConfigKeyDefaultBranch, cfg.Repository.DefaultBranch); err != nil {
		return fmt.Errorf("failed to set defaultbranch: %w", err)
	}

	// Clear existing shared entries and add new ones
	_ = gitConfigUnsetAll(bareDir, GitConfigKeyShared)
	for _, shared := range cfg.Shared {
		entry := formatSharedEntry(shared)
		if err := gitConfigAdd(bareDir, GitConfigKeyShared, entry); err != nil {
			return fmt.Errorf("failed to add shared entry: %w", err)
		}
	}

	return nil
}

// IsBaretreeRepoGit checks if the given path is a baretree repository
// A baretree repository is identified by:
// 1. Having a bare git repository (e.g., .bare directory) under the project root
// 2. The bare repository being a valid git bare repository
func IsBaretreeRepoGit(path string) bool {
	bareDir := findBareDir(path)
	if bareDir == "" {
		return false
	}

	// Verify that the bare repo is directly under the given path
	// (not somewhere else via worktree resolution)
	bareDirParent := filepath.Dir(bareDir)
	return bareDirParent == path
}

// FindRepoRootGit finds the repository root by looking for a bare repo with baretree config
func FindRepoRootGit(startPath string) (string, error) {
	currentPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", err
	}

	// First, check if we're in a worktree by looking for a .git file
	repoRoot, found := findRepoRootFromWorktree(currentPath)
	if found {
		return repoRoot, nil
	}

	// Fall back to walking up the directory tree
	for {
		if IsBaretreeRepoGit(currentPath) {
			return currentPath, nil
		}

		parent := filepath.Dir(currentPath)
		if parent == currentPath {
			return "", fmt.Errorf("not in a baretree repository")
		}
		currentPath = parent
	}
}

// findRepoRootFromWorktree attempts to find the baretree repository root
// by reading the .git file in a worktree and resolving to the bare repo's parent
func findRepoRootFromWorktree(startPath string) (string, bool) {
	currentPath := startPath

	for {
		gitPath := filepath.Join(currentPath, ".git")
		info, err := os.Stat(gitPath)
		if err == nil && !info.IsDir() {
			// Found a .git file - this is likely a worktree
			content, err := os.ReadFile(gitPath)
			if err != nil {
				return "", false
			}

			gitdirLine := strings.TrimSpace(string(content))
			if strings.HasPrefix(gitdirLine, "gitdir: ") {
				worktreeGitDir := strings.TrimPrefix(gitdirLine, "gitdir: ")
				bareRepoPath := resolveWorktreeToBareRepo(worktreeGitDir)
				if bareRepoPath != "" {
					// The repo root is the parent of the bare repo directory
					repoRoot := filepath.Dir(bareRepoPath)
					if IsBaretreeRepoGit(repoRoot) {
						return repoRoot, true
					}
				}
			}
		}

		parent := filepath.Dir(currentPath)
		if parent == currentPath {
			return "", false
		}
		currentPath = parent
	}
}

// findBareDir looks for a bare git repository in common locations
// It also handles the case where we're in a worktree (where .git is a file pointing to the bare repo)
func findBareDir(repoRoot string) string {
	// Check for .bare directory first (baretree standard)
	bareDir := filepath.Join(repoRoot, ".bare")
	info, err := os.Stat(bareDir)
	if err == nil && info.IsDir() && isBareRepo(bareDir) {
		return bareDir
	}

	// Check for .git file (worktree pointer) - NOT .git directory (normal git repo)
	gitPath := filepath.Join(repoRoot, ".git")
	info, err = os.Stat(gitPath)
	if err == nil && !info.IsDir() {
		// It's a file - this is a worktree, read to resolve to bare repo
		content, err := os.ReadFile(gitPath)
		if err != nil {
			return ""
		}
		gitdirLine := strings.TrimSpace(string(content))
		if strings.HasPrefix(gitdirLine, "gitdir: ") {
			// This is a worktree, resolve to the main bare repo
			worktreeGitDir := strings.TrimPrefix(gitdirLine, "gitdir: ")
			bareRepoPath := resolveWorktreeToBareRepo(worktreeGitDir)
			if bareRepoPath != "" && isBareRepo(bareRepoPath) {
				return bareRepoPath
			}
		}
	}

	return ""
}

// resolveWorktreeToBareRepo resolves a worktree git directory to the main bare repository
// worktreeGitDir is typically something like "/path/to/repo/.bare/worktrees/branch-name"
func resolveWorktreeToBareRepo(worktreeGitDir string) string {
	// Check if this looks like a worktree path (contains /worktrees/)
	if !strings.Contains(worktreeGitDir, string(filepath.Separator)+"worktrees"+string(filepath.Separator)) {
		return ""
	}

	// Find the worktrees directory and go up to get the bare repo
	// /path/to/repo/.bare/worktrees/branch-name -> /path/to/repo/.bare
	parts := strings.Split(worktreeGitDir, string(filepath.Separator)+"worktrees"+string(filepath.Separator))
	if len(parts) >= 1 {
		return parts[0]
	}

	return ""
}

// isBareRepo checks if a directory is a bare git repository
func isBareRepo(dir string) bool {
	headPath := filepath.Join(dir, "HEAD")
	info, err := os.Stat(headPath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// gitConfigGet gets a single value from git config
func gitConfigGet(bareDir, key string) (string, error) {
	cmd := exec.Command("git", "config", "--file", filepath.Join(bareDir, "config"), "--get", key)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// gitConfigGetAll gets all values for a key from git config
func gitConfigGetAll(bareDir, key string) ([]string, error) {
	cmd := exec.Command("git", "config", "--file", filepath.Join(bareDir, "config"), "--get-all", key)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string
	for _, line := range lines {
		if line != "" {
			result = append(result, line)
		}
	}
	return result, nil
}

// gitConfigSet sets a value in git config
func gitConfigSet(bareDir, key, value string) error {
	cmd := exec.Command("git", "config", "--file", filepath.Join(bareDir, "config"), key, value)
	return cmd.Run()
}

// gitConfigAdd adds a value to git config (for multi-valued keys)
func gitConfigAdd(bareDir, key, value string) error {
	cmd := exec.Command("git", "config", "--file", filepath.Join(bareDir, "config"), "--add", key, value)
	return cmd.Run()
}

// gitConfigUnsetAll removes all values for a key
func gitConfigUnsetAll(bareDir, key string) error {
	cmd := exec.Command("git", "config", "--file", filepath.Join(bareDir, "config"), "--unset-all", key)
	return cmd.Run()
}

// parseSharedEntry parses a shared entry from git config format
// Format: "source:type" or "source:type:managed"
func parseSharedEntry(entry string) (Shared, error) {
	parts := strings.Split(entry, ":")
	if len(parts) < 2 {
		return Shared{}, fmt.Errorf("invalid shared entry format: %s", entry)
	}

	shared := Shared{
		Source: parts[0],
		Type:   parts[1],
	}

	if len(parts) >= 3 && parts[2] == "managed" {
		shared.Managed = true
	}

	return shared, nil
}

// formatSharedEntry formats a Shared struct for git config storage
func formatSharedEntry(shared Shared) string {
	if shared.Managed {
		return fmt.Sprintf("%s:%s:managed", shared.Source, shared.Type)
	}
	return fmt.Sprintf("%s:%s", shared.Source, shared.Type)
}

// GetBareDir returns the bare directory path for a repository root
func GetBareDir(repoRoot string) (string, error) {
	bareDir := findBareDir(repoRoot)
	if bareDir == "" {
		return "", fmt.Errorf("bare repository not found in %s", repoRoot)
	}
	return bareDir, nil
}

// InitializeBaretreeConfig initializes baretree configuration in the bare repository
func InitializeBaretreeConfig(repoRoot, bareDir, defaultBranch string) error {
	barePath := filepath.Join(repoRoot, bareDir)

	// Set config values in git config
	if err := gitConfigSet(barePath, GitConfigKeyBareDir, bareDir); err != nil {
		return fmt.Errorf("failed to set baredir: %w", err)
	}

	if err := gitConfigSet(barePath, GitConfigKeyDefaultBranch, defaultBranch); err != nil {
		return fmt.Errorf("failed to set defaultbranch: %w", err)
	}

	return nil
}
