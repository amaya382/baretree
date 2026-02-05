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
	GitConfigKeyDefaultBranch = "baretree.defaultbranch"
	GitConfigKeyPostCreate    = "baretree.postcreate"
	GitConfigKeySyncToRoot    = "baretree.synctoroot"
)

// LoadConfigFromGit loads configuration from git-config in the bare repository
func LoadConfigFromGit(repoRoot string) (*Config, error) {
	// First, find the bare directory by checking common locations
	bareDir := findBareDir(repoRoot)
	if bareDir == "" {
		return nil, fmt.Errorf("bare repository not found in %s", repoRoot)
	}

	cfg := &Config{
		Repository: Repository{},
		PostCreate: []PostCreateAction{},
		SyncToRoot: []SyncToRootAction{},
	}

	// Read config values
	if defaultBranch, err := gitConfigGet(bareDir, GitConfigKeyDefaultBranch); err == nil && defaultBranch != "" {
		cfg.Repository.DefaultBranch = defaultBranch
	} else {
		cfg.Repository.DefaultBranch = "main"
	}

	// Read post-create entries
	postCreateEntries, err := gitConfigGetAll(bareDir, GitConfigKeyPostCreate)
	if err == nil {
		for _, entry := range postCreateEntries {
			if action, err := parsePostCreateEntry(entry); err == nil {
				cfg.PostCreate = append(cfg.PostCreate, action)
			}
		}
	}

	// Read sync-to-root entries
	syncToRootEntries, err := gitConfigGetAll(bareDir, GitConfigKeySyncToRoot)
	if err == nil {
		for _, entry := range syncToRootEntries {
			if action, err := parseSyncToRootEntry(entry); err == nil {
				cfg.SyncToRoot = append(cfg.SyncToRoot, action)
			}
		}
	}

	return cfg, nil
}

// SaveConfigToGit saves configuration to git-config in the bare repository
func SaveConfigToGit(repoRoot string, cfg *Config) error {
	bareDir := filepath.Join(repoRoot, BareDir)

	// Save basic config
	if err := gitConfigSet(bareDir, GitConfigKeyDefaultBranch, cfg.Repository.DefaultBranch); err != nil {
		return fmt.Errorf("failed to set defaultbranch: %w", err)
	}

	// Clear existing post-create entries and add new ones
	_ = gitConfigUnsetAll(bareDir, GitConfigKeyPostCreate)
	for _, action := range cfg.PostCreate {
		entry := formatPostCreateEntry(action)
		if err := gitConfigAdd(bareDir, GitConfigKeyPostCreate, entry); err != nil {
			return fmt.Errorf("failed to add post-create entry: %w", err)
		}
	}

	// Clear existing sync-to-root entries and add new ones
	_ = gitConfigUnsetAll(bareDir, GitConfigKeySyncToRoot)
	for _, action := range cfg.SyncToRoot {
		entry := formatSyncToRootEntry(action)
		if err := gitConfigAdd(bareDir, GitConfigKeySyncToRoot, entry); err != nil {
			return fmt.Errorf("failed to add sync-to-root entry: %w", err)
		}
	}

	return nil
}

// IsBaretreeRepoGit checks if the given path is a baretree repository
// A baretree repository is identified by:
// 1. Having a bare git repository (.git directory) under the project root
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

// findBareDir looks for the bare git repository (.git directory)
// It also handles the case where we're in a worktree (where .git is a file pointing to the bare repo)
func findBareDir(repoRoot string) string {
	gitPath := filepath.Join(repoRoot, BareDir)
	info, err := os.Stat(gitPath)
	if err != nil {
		return ""
	}

	// Check if it's a directory (bare repo at root)
	if info.IsDir() && isBareRepo(gitPath) {
		return gitPath
	}

	// Check if it's a file (worktree pointer)
	if !info.IsDir() {
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
// worktreeGitDir is typically something like "/path/to/repo/.git/worktrees/branch-name"
func resolveWorktreeToBareRepo(worktreeGitDir string) string {
	// Check if this looks like a worktree path (contains /worktrees/)
	if !strings.Contains(worktreeGitDir, string(filepath.Separator)+"worktrees"+string(filepath.Separator)) {
		return ""
	}

	// Find the worktrees directory and go up to get the bare repo
	// /path/to/repo/.git/worktrees/branch-name -> /path/to/repo/.git
	parts := strings.Split(worktreeGitDir, string(filepath.Separator)+"worktrees"+string(filepath.Separator))
	if len(parts) >= 1 {
		return parts[0]
	}

	return ""
}

// isBareRepo checks if a directory is a bare git repository
func isBareRepo(dir string) bool {
	// Check if HEAD file exists
	headPath := filepath.Join(dir, "HEAD")
	if _, err := os.Stat(headPath); err != nil {
		return false
	}

	// Check if core.bare = true in the config
	// Run git config to check (more reliable than parsing config file)
	cmd := exec.Command("git", "config", "--file", filepath.Join(dir, "config"), "--get", "core.bare")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
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

// gitConfigUnset removes a single-valued key from git config
func gitConfigUnset(bareDir, key string) error {
	cmd := exec.Command("git", "config", "--file", filepath.Join(bareDir, "config"), "--unset", key)
	return cmd.Run()
}

// gitConfigUnsetAll removes all values for a key
func gitConfigUnsetAll(bareDir, key string) error {
	cmd := exec.Command("git", "config", "--file", filepath.Join(bareDir, "config"), "--unset-all", key)
	return cmd.Run()
}

// parsePostCreateEntry parses a post-create entry from git config format
// Format for symlink/copy: "source:type" or "source:type:managed"
// Format for command: "command_string:command"
func parsePostCreateEntry(entry string) (PostCreateAction, error) {
	// Find the last colon to determine the type
	// This handles commands that may contain colons
	lastColonIdx := strings.LastIndex(entry, ":")
	if lastColonIdx == -1 {
		return PostCreateAction{}, fmt.Errorf("invalid post-create entry format: %s", entry)
	}

	suffix := entry[lastColonIdx+1:]

	// Check if it's a command type
	if suffix == "command" {
		return PostCreateAction{
			Source: entry[:lastColonIdx],
			Type:   "command",
		}, nil
	}

	// Parse as symlink/copy format: "source:type" or "source:type:managed"
	parts := strings.Split(entry, ":")
	if len(parts) < 2 {
		return PostCreateAction{}, fmt.Errorf("invalid post-create entry format: %s", entry)
	}

	action := PostCreateAction{
		Source: parts[0],
		Type:   parts[1],
	}

	if len(parts) >= 3 && parts[2] == "managed" {
		action.Managed = true
	}

	return action, nil
}

// formatPostCreateEntry formats a PostCreateAction for git config storage
func formatPostCreateEntry(action PostCreateAction) string {
	if action.Type == "command" {
		return fmt.Sprintf("%s:command", action.Source)
	}
	if action.Managed {
		return fmt.Sprintf("%s:%s:managed", action.Source, action.Type)
	}
	return fmt.Sprintf("%s:%s", action.Source, action.Type)
}

// parseSyncToRootEntry parses a sync-to-root entry from git config format
// Format: "source" or "source:target"
func parseSyncToRootEntry(entry string) (SyncToRootAction, error) {
	parts := strings.SplitN(entry, ":", 2)
	action := SyncToRootAction{
		Source: parts[0],
	}
	if len(parts) >= 2 {
		action.Target = parts[1]
	}
	return action, nil
}

// formatSyncToRootEntry formats a SyncToRootAction for git config storage
func formatSyncToRootEntry(action SyncToRootAction) string {
	if action.Target != "" && action.Target != action.Source {
		return fmt.Sprintf("%s:%s", action.Source, action.Target)
	}
	return action.Source
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
func InitializeBaretreeConfig(repoRoot, defaultBranch string) error {
	barePath := filepath.Join(repoRoot, BareDir)

	// Set config values in git config
	if err := gitConfigSet(barePath, GitConfigKeyDefaultBranch, defaultBranch); err != nil {
		return fmt.Errorf("failed to set defaultbranch: %w", err)
	}

	return nil
}

// UnsetDefaultBranch removes the default branch configuration from the repository
func UnsetDefaultBranch(repoRoot string) error {
	bareDir := findBareDir(repoRoot)
	if bareDir == "" {
		return fmt.Errorf("bare repository not found in %s", repoRoot)
	}

	if err := gitConfigUnset(bareDir, GitConfigKeyDefaultBranch); err != nil {
		return fmt.Errorf("failed to unset defaultbranch: %w", err)
	}

	return nil
}
