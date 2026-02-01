package worktree

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/amaya382/baretree/internal/config"
	"github.com/amaya382/baretree/internal/git"
)

const (
	// SharedDir is the directory name for managed shared files
	SharedDir = ".shared"
)

// PostCreateConflict represents a conflict when adding shared files
type PostCreateConflict struct {
	Source       string
	WorktreePath string
	WorktreeName string
}

// PostCreateApplyResult represents the result of applying post-create configuration
type PostCreateApplyResult struct {
	Source       string
	Type         string
	Managed      bool
	Applied      []string // worktree names where applied
	Skipped      []string // worktree names where skipped (already exists)
	SourceBranch string   // source branch name (for non-managed)
}

// PostCreateStatus represents the status of a post-create action in a worktree
type PostCreateStatus struct {
	WorktreeName string
	WorktreePath string
	Exists       bool
	IsSymlink    bool
	IsCorrect    bool // symlink points to correct location
}

// CommandResult represents the result of executing a command
type CommandResult struct {
	Command string
	Success bool
	Output  string
	Error   string
}

// GetPostCreateSourcePath returns the source path for a post-create file action
func (m *Manager) GetPostCreateSourcePath(action config.PostCreateAction) (string, error) {
	if action.Type == "command" {
		return "", fmt.Errorf("command type does not have a source path")
	}
	if action.Managed {
		return filepath.Join(m.RepoRoot, SharedDir, action.Source), nil
	}
	mainWorktree, err := m.getMainWorktreePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(mainWorktree, action.Source), nil
}

// GetSharedDir returns the path to the .shared directory
func (m *Manager) GetSharedDir() string {
	return filepath.Join(m.RepoRoot, SharedDir)
}

// CheckPostCreateConflicts checks if adding a post-create file would conflict with existing files
// For managed mode, this checks worktrees OTHER than the main worktree (since main worktree
// file will be moved to .shared/)
// For non-managed mode, this checks worktrees OTHER than the main worktree (since it's the source)
func (m *Manager) CheckPostCreateConflicts(source string, managed bool) ([]PostCreateConflict, error) {
	worktrees, err := m.listWorktrees()
	if err != nil {
		return nil, err
	}

	mainWorktree, err := m.getMainWorktreePath()
	if err != nil {
		return nil, err
	}

	var conflicts []PostCreateConflict

	for _, wt := range worktrees {
		if wt.IsBare {
			continue
		}

		// Skip the main worktree - for non-managed it's the source, for managed it will be moved
		// Use path comparison that handles symlinks and trailing slashes
		if pathsEqual(wt.Path, mainWorktree) {
			continue
		}

		targetPath := filepath.Join(wt.Path, source)
		if info, err := os.Lstat(targetPath); err == nil {
			// File exists - check if it's already a symlink to our source
			if info.Mode()&os.ModeSymlink != 0 {
				// It's a symlink, check if it points to our expected source
				linkTarget, err := os.Readlink(targetPath)
				if err == nil {
					expectedSource, _ := m.getExpectedSymlinkTarget(source, managed)
					if linkTarget == expectedSource {
						// Already correctly linked, not a conflict
						continue
					}
				}
			}
			// File exists and is not a correct symlink - it's a conflict
			conflicts = append(conflicts, PostCreateConflict{
				Source:       source,
				WorktreePath: targetPath,
				WorktreeName: filepath.Base(wt.Path),
			})
		}
	}

	return conflicts, nil
}

// getExpectedSymlinkTarget returns the expected symlink target for a post-create file
func (m *Manager) getExpectedSymlinkTarget(source string, managed bool) (string, error) {
	if managed {
		return filepath.Abs(filepath.Join(m.RepoRoot, SharedDir, source))
	}
	mainWorktree, err := m.getMainWorktreePath()
	if err != nil {
		return "", err
	}
	return filepath.Abs(filepath.Join(mainWorktree, source))
}

// AddPostCreate adds a new post-create action configuration and applies it
func (m *Manager) AddPostCreate(source string, actionType string, managed bool) (*PostCreateApplyResult, error) {
	// Check if already exists in config
	for _, a := range m.Config.PostCreate {
		if a.Source == source {
			return nil, fmt.Errorf("post-create action %s is already configured", source)
		}
	}

	// For command type, just add to config (no file operations needed)
	if actionType == "command" {
		newAction := config.PostCreateAction{
			Source: source,
			Type:   actionType,
		}
		m.Config.PostCreate = append(m.Config.PostCreate, newAction)

		if err := config.SaveConfig(m.RepoRoot, m.Config); err != nil {
			return nil, fmt.Errorf("failed to save config: %w", err)
		}

		return &PostCreateApplyResult{
			Source: source,
			Type:   actionType,
		}, nil
	}

	// Check for conflicts (for symlink/copy types)
	conflicts, err := m.CheckPostCreateConflicts(source, managed)
	if err != nil {
		return nil, err
	}
	if len(conflicts) > 0 {
		return nil, &PostCreateConflictError{Conflicts: conflicts}
	}

	// Get source file path
	mainWorktree, err := m.getMainWorktreePath()
	if err != nil {
		return nil, err
	}
	sourcePath := filepath.Join(mainWorktree, source)

	// Check source exists in main worktree
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("source file does not exist: %s", sourcePath)
	}

	// For managed: move source to .shared directory
	if managed {
		sharedDir := m.GetSharedDir()
		targetPath := filepath.Join(sharedDir, source)

		// Create .shared directory structure
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create .shared directory: %w", err)
		}

		// Move file from main worktree to .shared
		if err := os.Rename(sourcePath, targetPath); err != nil {
			return nil, fmt.Errorf("failed to move %s to .shared: %w", source, err)
		}
	}

	// Add to config
	newAction := config.PostCreateAction{
		Source:  source,
		Type:    actionType,
		Managed: managed,
	}
	m.Config.PostCreate = append(m.Config.PostCreate, newAction)

	// Save config
	if err := config.SaveConfig(m.RepoRoot, m.Config); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	// Apply to all worktrees
	result, err := m.applyPostCreateToAllWorktrees(newAction)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// applyPostCreateToAllWorktrees applies a post-create configuration to all worktrees
func (m *Manager) applyPostCreateToAllWorktrees(action config.PostCreateAction) (*PostCreateApplyResult, error) {
	// Commands are not applied to existing worktrees
	if action.Type == "command" {
		return &PostCreateApplyResult{
			Source: action.Source,
			Type:   action.Type,
		}, nil
	}

	worktrees, err := m.listWorktrees()
	if err != nil {
		return nil, err
	}

	mainWorktree, err := m.getMainWorktreePath()
	if err != nil {
		return nil, err
	}

	sourcePath, err := m.GetPostCreateSourcePath(action)
	if err != nil {
		return nil, err
	}

	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return nil, err
	}

	result := &PostCreateApplyResult{
		Source:       action.Source,
		Type:         action.Type,
		Managed:      action.Managed,
		SourceBranch: m.Config.Repository.DefaultBranch,
	}

	for _, wt := range worktrees {
		if wt.IsBare {
			continue
		}

		wtName := filepath.Base(wt.Path)
		targetPath := filepath.Join(wt.Path, action.Source)

		// For non-managed, skip the main worktree (it's the source)
		if !action.Managed && pathsEqual(wt.Path, mainWorktree) {
			continue
		}

		// Check if target already exists
		if _, err := os.Lstat(targetPath); err == nil {
			result.Skipped = append(result.Skipped, wtName)
			continue
		}

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory for %s: %w", targetPath, err)
		}

		switch action.Type {
		case "symlink":
			if err := os.Symlink(absSource, targetPath); err != nil {
				return nil, fmt.Errorf("failed to create symlink %s: %w", targetPath, err)
			}
		case "copy":
			if err := copyFile(sourcePath, targetPath); err != nil {
				return nil, fmt.Errorf("failed to copy to %s: %w", targetPath, err)
			}
		default:
			return nil, fmt.Errorf("unknown post-create type: %s", action.Type)
		}

		result.Applied = append(result.Applied, wtName)
	}

	return result, nil
}

// RemovePostCreate removes a post-create action configuration and cleans up
func (m *Manager) RemovePostCreate(source string, removeAll bool) (*PostCreateRemoveResult, error) {
	// Find action config
	var found *config.PostCreateAction
	var foundIndex int
	for i, a := range m.Config.PostCreate {
		if a.Source == source {
			found = &m.Config.PostCreate[i]
			foundIndex = i
			break
		}
	}

	if found == nil {
		return nil, fmt.Errorf("post-create action %s is not configured", source)
	}

	result := &PostCreateRemoveResult{
		Source:  source,
		Type:    found.Type,
		Managed: found.Managed,
	}

	// For command type, just remove from config
	if found.Type == "command" {
		m.Config.PostCreate = append(m.Config.PostCreate[:foundIndex], m.Config.PostCreate[foundIndex+1:]...)
		if err := config.SaveConfig(m.RepoRoot, m.Config); err != nil {
			return nil, fmt.Errorf("failed to save config: %w", err)
		}
		return result, nil
	}

	worktrees, err := m.listWorktrees()
	if err != nil {
		return nil, err
	}

	mainWorktree, err := m.getMainWorktreePath()
	if err != nil {
		return nil, err
	}

	// Remove from worktrees
	for _, wt := range worktrees {
		if wt.IsBare {
			continue
		}

		// Skip main worktree for non-managed (it's the source)
		if !found.Managed && pathsEqual(wt.Path, mainWorktree) {
			continue
		}

		wtName := filepath.Base(wt.Path)
		targetPath := filepath.Join(wt.Path, source)

		info, err := os.Lstat(targetPath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to stat %s: %w", targetPath, err)
		}

		isSymlink := info.Mode()&os.ModeSymlink != 0

		if isSymlink {
			// Always remove symlinks
			if err := os.Remove(targetPath); err != nil {
				return nil, fmt.Errorf("failed to remove symlink %s: %w", targetPath, err)
			}
			result.RemovedSymlinks = append(result.RemovedSymlinks, wtName)
		} else if removeAll {
			// Remove copies only if --all is specified
			if err := os.RemoveAll(targetPath); err != nil {
				return nil, fmt.Errorf("failed to remove %s: %w", targetPath, err)
			}
			result.RemovedCopies = append(result.RemovedCopies, wtName)
		} else {
			// Skip copies
			result.SkippedCopies = append(result.SkippedCopies, wtName)
		}
	}

	// Remove from config
	m.Config.PostCreate = append(m.Config.PostCreate[:foundIndex], m.Config.PostCreate[foundIndex+1:]...)

	// Save config
	if err := config.SaveConfig(m.RepoRoot, m.Config); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return result, nil
}

// PostCreateRemoveResult represents the result of removing post-create configuration
type PostCreateRemoveResult struct {
	Source          string
	Type            string
	Managed         bool
	RemovedSymlinks []string
	RemovedCopies   []string
	SkippedCopies   []string
}

// ApplyAllPostCreate applies all post-create configurations (for manual config edits)
func (m *Manager) ApplyAllPostCreate() ([]PostCreateApplyResult, error) {
	if len(m.Config.PostCreate) == 0 {
		return nil, nil
	}

	// First, check for conflicts across all file-based post-create configs
	var allConflicts []PostCreateConflict
	for _, action := range m.Config.PostCreate {
		if action.Type == "command" {
			continue
		}
		conflicts, err := m.CheckPostCreateConflicts(action.Source, action.Managed)
		if err != nil {
			return nil, err
		}
		allConflicts = append(allConflicts, conflicts...)
	}

	if len(allConflicts) > 0 {
		return nil, &PostCreateConflictError{Conflicts: allConflicts}
	}

	// Apply all post-create configs
	var results []PostCreateApplyResult
	for _, action := range m.Config.PostCreate {
		if action.Type == "command" {
			// Commands are not applied to existing worktrees
			results = append(results, PostCreateApplyResult{
				Source: action.Source,
				Type:   action.Type,
			})
			continue
		}

		// For managed, ensure source exists in .shared
		sourcePath, err := m.GetPostCreateSourcePath(action)
		if err != nil {
			return nil, err
		}

		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			// For managed, try to move from main worktree
			if action.Managed {
				mainWorktree, err := m.getMainWorktreePath()
				if err != nil {
					return nil, err
				}
				mainSourcePath := filepath.Join(mainWorktree, action.Source)
				if _, err := os.Stat(mainSourcePath); err == nil {
					// Create .shared directory structure
					sharedDir := m.GetSharedDir()
					targetPath := filepath.Join(sharedDir, action.Source)
					if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
						return nil, fmt.Errorf("failed to create .shared directory: %w", err)
					}
					// Move file
					if err := os.Rename(mainSourcePath, targetPath); err != nil {
						return nil, fmt.Errorf("failed to move %s to .shared: %w", action.Source, err)
					}
				}
			} else {
				// Skip if source doesn't exist
				continue
			}
		}

		result, err := m.applyPostCreateToAllWorktrees(action)
		if err != nil {
			return nil, err
		}
		results = append(results, *result)
	}

	return results, nil
}

// GetPostCreateStatus returns the status of all post-create actions
func (m *Manager) GetPostCreateStatus() ([]PostCreateStatusInfo, error) {
	worktrees, err := m.listWorktrees()
	if err != nil {
		return nil, err
	}

	mainWorktree, err := m.getMainWorktreePath()
	if err != nil {
		return nil, err
	}

	var statuses []PostCreateStatusInfo

	for _, action := range m.Config.PostCreate {
		info := PostCreateStatusInfo{
			Source:  action.Source,
			Type:    action.Type,
			Managed: action.Managed,
		}

		// Command type doesn't have file status
		if action.Type == "command" {
			statuses = append(statuses, info)
			continue
		}

		sourcePath, err := m.GetPostCreateSourcePath(action)
		if err != nil {
			return nil, err
		}

		absSource, _ := filepath.Abs(sourcePath)
		info.SourceExists = fileExists(sourcePath)

		for _, wt := range worktrees {
			if wt.IsBare {
				continue
			}

			wtName := filepath.Base(wt.Path)
			targetPath := filepath.Join(wt.Path, action.Source)

			// For non-managed, main worktree is the source
			if !action.Managed && pathsEqual(wt.Path, mainWorktree) {
				info.SourceWorktree = wtName
				continue
			}

			status := PostCreateStatus{
				WorktreeName: wtName,
				WorktreePath: targetPath,
			}

			linfo, err := os.Lstat(targetPath)
			if os.IsNotExist(err) {
				status.Exists = false
				info.Missing = append(info.Missing, wtName)
			} else if err != nil {
				continue
			} else {
				status.Exists = true
				status.IsSymlink = linfo.Mode()&os.ModeSymlink != 0

				if status.IsSymlink {
					linkTarget, err := os.Readlink(targetPath)
					if err == nil && linkTarget == absSource {
						status.IsCorrect = true
						info.Applied = append(info.Applied, wtName)
					} else {
						info.Applied = append(info.Applied, wtName) // exists but may point elsewhere
					}
				} else {
					info.Applied = append(info.Applied, wtName)
				}
			}
		}

		statuses = append(statuses, info)
	}

	return statuses, nil
}

// PostCreateStatusInfo represents the status of a post-create configuration
type PostCreateStatusInfo struct {
	Source         string
	Type           string
	Managed        bool
	SourceExists   bool
	SourceWorktree string   // for non-managed
	Applied        []string // worktrees where applied
	Missing        []string // worktrees where missing
}

// listWorktrees returns all worktrees
func (m *Manager) listWorktrees() ([]WorktreeInfo, error) {
	output, err := m.Executor.Execute("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return ParseWorktreeList(output), nil
}

// PostCreateConflictError is returned when conflicts are detected
type PostCreateConflictError struct {
	Conflicts []PostCreateConflict
}

func (e *PostCreateConflictError) Error() string {
	return fmt.Sprintf("conflicts detected in %d location(s)", len(e.Conflicts))
}

// ExecutePostCreateCommands executes all command-type post-create actions in a worktree
func (m *Manager) ExecutePostCreateCommands(worktreePath string) []CommandResult {
	var results []CommandResult

	for _, action := range m.Config.PostCreate {
		if action.Type != "command" {
			continue
		}

		result := CommandResult{
			Command: action.Source,
		}

		cmd := exec.Command("sh", "-c", action.Source)
		cmd.Dir = worktreePath

		output, err := cmd.CombinedOutput()
		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Output = string(output)
		} else {
			result.Success = true
			result.Output = string(output)
		}

		results = append(results, result)
	}

	return results
}

// ApplyPostCreateConfig applies post-create file/directory configuration to a worktree
// and executes any configured commands
func (m *Manager) ApplyPostCreateConfig(worktreePath string) ([]CommandResult, error) {
	// Apply file-based actions first
	for _, action := range m.Config.PostCreate {
		if action.Type == "command" {
			continue
		}

		sourcePath, err := m.GetPostCreateSourcePath(action)
		if err != nil {
			return nil, err
		}

		targetPath := filepath.Join(worktreePath, action.Source)

		// Check if source exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			// Source doesn't exist yet, skip (not an error)
			continue
		}

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create parent directory for %s: %w", targetPath, err)
		}

		// Check if target already exists
		if _, err := os.Lstat(targetPath); err == nil {
			// Target exists, skip to avoid overwriting
			continue
		}

		switch action.Type {
		case "symlink":
			// Create symlink with absolute path
			absSource, err := filepath.Abs(sourcePath)
			if err != nil {
				return nil, fmt.Errorf("failed to get absolute path for %s: %w", sourcePath, err)
			}

			if err := os.Symlink(absSource, targetPath); err != nil {
				return nil, fmt.Errorf("failed to create symlink %s -> %s: %w", targetPath, absSource, err)
			}

		case "copy":
			if err := copyFile(sourcePath, targetPath); err != nil {
				return nil, fmt.Errorf("failed to copy %s to %s: %w", sourcePath, targetPath, err)
			}

		default:
			return nil, fmt.Errorf("unknown post-create type: %s", action.Type)
		}
	}

	// Execute commands after file operations
	commandResults := m.ExecutePostCreateCommands(worktreePath)

	return commandResults, nil
}

// getMainWorktreePath returns the path to the main worktree (default branch worktree)
func (m *Manager) getMainWorktreePath() (string, error) {
	defaultBranch := m.Config.Repository.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	return filepath.Join(m.RepoRoot, defaultBranch), nil
}

// GetDefaultBranch returns the default branch name
func (m *Manager) GetDefaultBranch() string {
	if m.Config.Repository.DefaultBranch == "" {
		return "main"
	}
	return m.Config.Repository.DefaultBranch
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// pathsEqual compares two file paths, handling symlinks and normalizing
func pathsEqual(path1, path2 string) bool {
	// First try direct comparison
	if path1 == path2 {
		return true
	}

	// Try with cleaned paths
	clean1 := filepath.Clean(path1)
	clean2 := filepath.Clean(path2)
	if clean1 == clean2 {
		return true
	}

	// Try with symlink resolution
	real1, err1 := filepath.EvalSymlinks(path1)
	real2, err2 := filepath.EvalSymlinks(path2)
	if err1 == nil && err2 == nil && real1 == real2 {
		return true
	}

	return false
}

// ParseWorktreeList is a wrapper for git.ParseWorktreeList
func ParseWorktreeList(output string) []WorktreeInfo {
	gitWorktrees := git.ParseWorktreeList(output)
	worktrees := make([]WorktreeInfo, len(gitWorktrees))

	for i, wt := range gitWorktrees {
		worktrees[i] = WorktreeInfo{
			Path:   wt.Path,
			Head:   wt.Head,
			Branch: wt.Branch,
			IsMain: wt.IsMain,
			IsBare: wt.IsBare,
		}
	}

	return worktrees
}

// WorktreeInfo represents worktree information
type WorktreeInfo struct {
	Path   string
	Head   string
	Branch string
	IsMain bool
	IsBare bool
}
