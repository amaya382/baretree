package worktree

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/amaya382/baretree/internal/config"
)

// SyncToRootApplyResult represents the result of applying a sync-to-root configuration
type SyncToRootApplyResult struct {
	Source  string
	Target  string
	Applied bool   // true if symlink was created
	Skipped bool   // true if symlink already exists correctly
	Error   string // non-empty if there was an error
}

// SyncToRootStatusInfo represents the status of a sync-to-root configuration
type SyncToRootStatusInfo struct {
	Source       string
	Target       string
	SourceExists bool   // source file/dir exists in default branch worktree
	TargetExists bool   // target symlink exists in repo root
	IsCorrect    bool   // symlink points to correct location
	LinkTarget   string // actual symlink target (if symlink)
	ExpectedLink string // expected symlink target
}

// AddSyncToRoot adds a new sync-to-root action configuration and creates the symlink
func (m *Manager) AddSyncToRoot(source, target string, force bool) (*SyncToRootApplyResult, error) {
	// Default target to source if not specified
	if target == "" {
		target = source
	}

	// Check if already exists in config
	for _, a := range m.Config.SyncToRoot {
		if a.Source == source {
			return nil, fmt.Errorf("sync-to-root action for %s is already configured", source)
		}
	}

	// Get main worktree path
	mainWorktree, err := m.getMainWorktreePath()
	if err != nil {
		return nil, err
	}

	// Check source exists in main worktree
	sourcePath := filepath.Join(mainWorktree, source)
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("source does not exist: %s", sourcePath)
	} else if err != nil {
		return nil, fmt.Errorf("failed to stat source: %w", err)
	}

	// Calculate target path in repo root
	targetPath := filepath.Join(m.RepoRoot, target)

	// Calculate relative path from target to source
	// e.g., from /repo/CLAUDE.md to /repo/main/CLAUDE.md -> main/CLAUDE.md
	targetDir := filepath.Dir(targetPath)
	relSource, err := filepath.Rel(targetDir, sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate relative path: %w", err)
	}

	// Check if target already exists
	targetInfo, err := os.Lstat(targetPath)
	if err == nil {
		// Target exists
		if targetInfo.Mode()&os.ModeSymlink != 0 {
			// It's a symlink - check if it points to the correct location
			linkTarget, err := os.Readlink(targetPath)
			if err == nil {
				if linkTarget == relSource {
					// Already correctly linked, just add to config
					newAction := config.SyncToRootAction{
						Source: source,
						Target: target,
					}
					m.Config.SyncToRoot = append(m.Config.SyncToRoot, newAction)
					if err := config.SaveConfig(m.RepoRoot, m.Config); err != nil {
						return nil, fmt.Errorf("failed to save config: %w", err)
					}
					return &SyncToRootApplyResult{
						Source:  source,
						Target:  target,
						Skipped: true,
					}, nil
				}
			}
			// Wrong symlink - remove if force
			if force {
				if err := os.Remove(targetPath); err != nil {
					return nil, fmt.Errorf("failed to remove existing symlink: %w", err)
				}
			} else {
				return nil, fmt.Errorf("target exists and is a symlink to wrong location: %s (use --force to overwrite)", targetPath)
			}
		} else {
			// It's a regular file/directory
			return nil, fmt.Errorf("target already exists and is not a symlink: %s (please remove manually)", targetPath)
		}
	}

	// Create parent directories if needed
	if targetDir != m.RepoRoot {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create parent directory: %w", err)
		}
	}

	// Create symlink with relative path
	if err := os.Symlink(relSource, targetPath); err != nil {
		return nil, fmt.Errorf("failed to create symlink: %w", err)
	}

	// Add to config
	newAction := config.SyncToRootAction{
		Source: source,
		Target: target,
	}
	m.Config.SyncToRoot = append(m.Config.SyncToRoot, newAction)

	// Save config
	if err := config.SaveConfig(m.RepoRoot, m.Config); err != nil {
		// Try to clean up the symlink we just created
		_ = os.Remove(targetPath)
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return &SyncToRootApplyResult{
		Source:  source,
		Target:  target,
		Applied: true,
	}, nil
}

// RemoveSyncToRoot removes a sync-to-root action configuration and deletes the symlink
func (m *Manager) RemoveSyncToRoot(source string) error {
	// Find action config
	var found *config.SyncToRootAction
	var foundIndex int
	for i, a := range m.Config.SyncToRoot {
		if a.Source == source {
			found = &m.Config.SyncToRoot[i]
			foundIndex = i
			break
		}
	}

	if found == nil {
		return fmt.Errorf("sync-to-root action for %s is not configured", source)
	}

	// Calculate target path
	target := found.Target
	if target == "" {
		target = found.Source
	}
	targetPath := filepath.Join(m.RepoRoot, target)

	// Remove symlink if it exists and is a symlink
	info, err := os.Lstat(targetPath)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(targetPath); err != nil {
				return fmt.Errorf("failed to remove symlink: %w", err)
			}
		}
		// Don't remove non-symlinks automatically
	}

	// Remove from config
	m.Config.SyncToRoot = append(m.Config.SyncToRoot[:foundIndex], m.Config.SyncToRoot[foundIndex+1:]...)

	// Save config
	if err := config.SaveConfig(m.RepoRoot, m.Config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// ApplyAllSyncToRoot applies all sync-to-root configurations
func (m *Manager) ApplyAllSyncToRoot(force bool) ([]SyncToRootApplyResult, error) {
	if len(m.Config.SyncToRoot) == 0 {
		return nil, nil
	}

	mainWorktree, err := m.getMainWorktreePath()
	if err != nil {
		return nil, err
	}

	var results []SyncToRootApplyResult

	for _, action := range m.Config.SyncToRoot {
		result := SyncToRootApplyResult{
			Source: action.Source,
			Target: action.Target,
		}

		if result.Target == "" {
			result.Target = result.Source
		}

		// Check source exists
		sourcePath := filepath.Join(mainWorktree, action.Source)
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			result.Error = fmt.Sprintf("source does not exist: %s", sourcePath)
			results = append(results, result)
			continue
		} else if err != nil {
			result.Error = fmt.Sprintf("failed to stat source: %v", err)
			results = append(results, result)
			continue
		}

		// Calculate target path
		targetPath := filepath.Join(m.RepoRoot, result.Target)
		targetDir := filepath.Dir(targetPath)

		// Calculate relative path from target to source
		relSource, err := filepath.Rel(targetDir, sourcePath)
		if err != nil {
			result.Error = fmt.Sprintf("failed to calculate relative path: %v", err)
			results = append(results, result)
			continue
		}

		// Check if target already exists
		targetInfo, err := os.Lstat(targetPath)
		if err == nil {
			if targetInfo.Mode()&os.ModeSymlink != 0 {
				// Check if it points to correct location
				linkTarget, err := os.Readlink(targetPath)
				if err == nil {
					if linkTarget == relSource {
						result.Skipped = true
						results = append(results, result)
						continue
					}
				}
				// Wrong symlink
				if force {
					if err := os.Remove(targetPath); err != nil {
						result.Error = fmt.Sprintf("failed to remove existing symlink: %v", err)
						results = append(results, result)
						continue
					}
				} else {
					result.Error = "target exists and is a symlink to wrong location (use --force to overwrite)"
					results = append(results, result)
					continue
				}
			} else {
				// Regular file/directory
				result.Error = "target already exists and is not a symlink (please remove manually)"
				results = append(results, result)
				continue
			}
		}

		// Create parent directories if needed
		if targetDir != m.RepoRoot {
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				result.Error = fmt.Sprintf("failed to create parent directory: %v", err)
				results = append(results, result)
				continue
			}
		}

		// Create symlink with relative path
		if err := os.Symlink(relSource, targetPath); err != nil {
			result.Error = fmt.Sprintf("failed to create symlink: %v", err)
			results = append(results, result)
			continue
		}

		result.Applied = true
		results = append(results, result)
	}

	return results, nil
}

// GetSyncToRootStatus returns the status of all sync-to-root configurations
func (m *Manager) GetSyncToRootStatus() ([]SyncToRootStatusInfo, error) {
	mainWorktree, err := m.getMainWorktreePath()
	if err != nil {
		return nil, err
	}

	var statuses []SyncToRootStatusInfo

	for _, action := range m.Config.SyncToRoot {
		target := action.Target
		if target == "" {
			target = action.Source
		}

		info := SyncToRootStatusInfo{
			Source: action.Source,
			Target: target,
		}

		// Check source exists
		sourcePath := filepath.Join(mainWorktree, action.Source)
		targetPath := filepath.Join(m.RepoRoot, target)
		targetDir := filepath.Dir(targetPath)

		// Calculate expected relative path
		relSource, _ := filepath.Rel(targetDir, sourcePath)
		info.ExpectedLink = relSource

		if _, err := os.Stat(sourcePath); err == nil {
			info.SourceExists = true
		}

		// Check target symlink
		targetInfo, err := os.Lstat(targetPath)
		if err == nil {
			info.TargetExists = true

			if targetInfo.Mode()&os.ModeSymlink != 0 {
				linkTarget, err := os.Readlink(targetPath)
				if err == nil {
					info.LinkTarget = linkTarget
					info.IsCorrect = (linkTarget == relSource)
				}
			}
		}

		statuses = append(statuses, info)
	}

	return statuses, nil
}
