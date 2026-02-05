package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/amaya382/baretree/internal/git"
	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	repairDryRun   bool
	repairSource   string
	repairAll      bool
	repairFixPaths bool
)

var repairCmd = &cobra.Command{
	Use:   "repair [worktree]",
	Short: "Repair unmanaged worktrees into baretree structure",
	Long: `Repair unmanaged worktrees into baretree structure.

This command moves worktrees that are outside the repository root into the
proper baretree structure, and fixes naming inconsistencies between directory
names and branch names.

Use --source to specify which name to use as the source of truth:
  --source=branch  Use branch name as source of truth (default)
                   - Directory is renamed to match branch name
  --source=dir     Use directory name as source of truth
                   - Branch is renamed to match directory name

Use --fix-paths when the repository or worktrees have been moved:
  - Without arguments: auto-detect and fix paths for worktrees inside repo
  - With path arguments: fix paths for worktrees at specified external locations

Examples:
  bt repair test                      # Repair worktree 'test' (by name or branch)
  bt repair                           # Repair current worktree
  bt repair --all                     # Repair all unmanaged worktrees
  bt repair --source=dir              # Use directory name as source
  bt repair --dry-run                 # Show what would be done
  bt repair --fix-paths               # Fix paths after moving repository
  bt repair --fix-paths /path/to/ext  # Fix path for externally moved worktree`,
	Args:              cobra.ArbitraryArgs,
	RunE:              runRepair,
	ValidArgsFunction: completeWorktreeNames(false),
}

func init() {
	repairCmd.Flags().BoolVar(&repairDryRun, "dry-run", false, "Show what would happen without executing")
	repairCmd.Flags().StringVar(&repairSource, "source", "branch", "Source of truth for naming: branch or dir")
	repairCmd.Flags().BoolVar(&repairAll, "all", false, "Repair all unmanaged worktrees")
	repairCmd.Flags().BoolVar(&repairFixPaths, "fix-paths", false, "Fix worktree paths after moving repository")
}

func runRepair(cmd *cobra.Command, args []string) error {
	// Validate source option
	if repairSource != "branch" && repairSource != "dir" {
		return fmt.Errorf("invalid source: %s (must be 'branch' or 'dir')", repairSource)
	}

	// Find repository root
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("not in a baretree repository: %w", err)
	}

	bareDir, err := repository.GetBareRepoPath(repoRoot)
	if err != nil {
		return err
	}

	executor := git.NewExecutor(bareDir)

	// Load config for post-create application
	mgr, err := repository.NewManager(repoRoot)
	if err != nil {
		return err
	}
	wtMgr := worktree.NewManager(repoRoot, bareDir, mgr.Config)

	// Handle --fix-paths mode
	if repairFixPaths {
		return runFixPaths(repoRoot, bareDir, executor, wtMgr, args, repairDryRun)
	}

	// Without --fix-paths, only 0 or 1 argument is allowed
	if len(args) > 1 {
		return fmt.Errorf("too many arguments; use --fix-paths to specify multiple paths")
	}

	// Get all worktrees
	output, err := executor.Execute("worktree", "list", "--porcelain")
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	worktrees := git.ParseWorktreeList(output)

	// Find worktrees that need repair
	var targets []repairTarget
	for _, wt := range worktrees {
		if wt.IsBare {
			continue
		}

		target := analyzeWorktreeForRepair(wt, repoRoot)
		if target != nil {
			targets = append(targets, *target)
		}
	}

	// Filter based on arguments
	if len(args) == 1 && !repairAll {
		// Specific worktree specified
		name := args[0]
		var found *repairTarget
		for _, t := range targets {
			if t.Branch == name || t.DirName == name {
				found = &t
				break
			}
		}
		if found == nil {
			// Check if it's a broken worktree (path doesn't exist)
			for _, wt := range worktrees {
				if wt.IsBare {
					continue
				}
				if wt.Branch == name {
					// Check if the path exists
					if _, err := os.Stat(wt.Path); os.IsNotExist(err) {
						fmt.Printf("Worktree '%s' has a broken path (moved or deleted)\n", name)
						fmt.Printf("Last known path: %s\n\n", wt.Path)
						fmt.Println("To fix this, specify the new location:")
						fmt.Printf("  bt repair --fix-paths /new/path/to/%s\n", name)
						return nil
					}
				}
			}
			// Check if it exists but is already managed
			for _, wt := range worktrees {
				if wt.IsBare {
					continue
				}
				relPath, _ := filepath.Rel(repoRoot, wt.Path)
				if (relPath == name || wt.Branch == name) && isWorktreeManaged(wt, repoRoot) {
					fmt.Printf("Worktree '%s' is already managed\n", name)
					return nil
				}
			}
			return fmt.Errorf("worktree not found: %s", name)
		}
		targets = []repairTarget{*found}
	} else if !repairAll {
		// No args: current worktree only
		currentWorktree, err := detectCurrentWorktree(cwd, repoRoot)
		if err != nil {
			return fmt.Errorf("failed to detect current worktree: %w", err)
		}

		var found *repairTarget
		for _, t := range targets {
			if t.DirName == currentWorktree || strings.HasPrefix(t.DirName, currentWorktree+string(filepath.Separator)) {
				found = &t
				break
			}
		}
		if found == nil {
			fmt.Printf("Current worktree '%s' is already managed\n", currentWorktree)
			return nil
		}
		targets = []repairTarget{*found}
	}

	if len(targets) == 0 {
		fmt.Println("No worktrees need repair")
		return nil
	}

	// Display what will be done
	fmt.Printf("Found %d worktree(s) to repair:\n\n", len(targets))
	for _, t := range targets {
		fmt.Printf("  Worktree: %s\n", t.Path)
		fmt.Printf("  Branch:   %s\n", t.Branch)
		fmt.Printf("  Issue:    %s\n", t.Issue)
		fmt.Printf("  Action:   %s\n", t.describeAction(repairSource))
		fmt.Println()
	}

	if repairDryRun {
		fmt.Println("Dry run - no changes made")
		return nil
	}

	// Perform repair
	for _, t := range targets {
		fmt.Printf("Repairing '%s'...\n", t.Branch)

		if err := t.repair(repoRoot, bareDir, executor, wtMgr, repairSource); err != nil {
			return fmt.Errorf("failed to repair %s: %w", t.Branch, err)
		}

		fmt.Printf("  Done\n")
	}

	fmt.Printf("\nSuccessfully repaired %d worktree(s)\n", len(targets))
	return nil
}

type repairTarget struct {
	Path     string // Current absolute path
	Branch   string // Branch name
	DirName  string // Current directory name (relative to repo root, may be empty for external)
	Issue    string // Description of the issue
	External bool   // Whether the worktree is outside repo root
}

type brokenWorktree struct {
	name    string // Worktree directory name in .git/worktrees/
	branch  string // Branch name
	oldPath string // Last known path
}

// analyzeWorktreeForRepair checks if a worktree needs repair and returns target info
func analyzeWorktreeForRepair(wt git.Worktree, repoRoot string) *repairTarget {
	relPath, err := filepath.Rel(repoRoot, wt.Path)

	// External worktree (outside repo root)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return &repairTarget{
			Path:     wt.Path,
			Branch:   wt.Branch,
			DirName:  "",
			Issue:    "Outside repository root",
			External: true,
		}
	}

	// Internal worktree with name mismatch
	if relPath != wt.Branch {
		return &repairTarget{
			Path:     wt.Path,
			Branch:   wt.Branch,
			DirName:  relPath,
			Issue:    fmt.Sprintf("Directory '%s' doesn't match branch '%s'", relPath, wt.Branch),
			External: false,
		}
	}

	return nil
}

// isWorktreeManaged checks if a worktree is properly managed
func isWorktreeManaged(wt git.Worktree, repoRoot string) bool {
	relPath, err := filepath.Rel(repoRoot, wt.Path)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return false
	}
	return relPath == wt.Branch
}

func (t *repairTarget) describeAction(source string) string {
	if t.External {
		return fmt.Sprintf("Move to %s/", t.Branch)
	}

	if source == "branch" {
		return fmt.Sprintf("Rename directory '%s' -> '%s'", t.DirName, t.Branch)
	}
	return fmt.Sprintf("Rename branch '%s' -> '%s'", t.Branch, t.DirName)
}

func (t *repairTarget) repair(repoRoot, bareDir string, executor *git.Executor, wtMgr *worktree.Manager, source string) error {
	if t.External {
		return t.repairExternal(repoRoot, bareDir, executor, wtMgr)
	}
	return t.repairInternal(repoRoot, bareDir, executor, source)
}

func (t *repairTarget) repairExternal(repoRoot, bareDir string, executor *git.Executor, wtMgr *worktree.Manager) error {
	targetPath := filepath.Join(repoRoot, t.Branch)

	// Check if target already exists
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("target path already exists: %s", targetPath)
	}

	// Create parent directories if needed
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Move worktree
	if err := os.Rename(t.Path, targetPath); err != nil {
		return fmt.Errorf("failed to move worktree: %w", err)
	}

	// Repair Git worktree metadata
	if _, err := executor.Execute("worktree", "repair", targetPath); err != nil {
		// Try to rollback
		if rollbackErr := os.Rename(targetPath, t.Path); rollbackErr != nil {
			return fmt.Errorf("failed to repair worktree and also failed to roll back: %w / %w", err, rollbackErr)
		}
		return fmt.Errorf("failed to repair worktree: %w", err)
	}

	// Apply post-create file configuration (pass nil to discard command output in repair context)
	if _, err := wtMgr.ApplyPostCreateConfig(targetPath, nil); err != nil {
		fmt.Printf("  Warning: failed to apply post-create config: %v\n", err)
	}

	return nil
}

func (t *repairTarget) repairInternal(repoRoot, bareDir string, executor *git.Executor, source string) error {
	if source == "branch" {
		return t.renameDirToBranch(repoRoot, executor)
	}
	return t.renameBranchToDir(executor)
}

func (t *repairTarget) renameDirToBranch(repoRoot string, executor *git.Executor) error {
	newPath := filepath.Join(repoRoot, t.Branch)

	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("target directory already exists: %s", newPath)
	}

	if err := os.MkdirAll(filepath.Dir(newPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	if err := os.Rename(t.Path, newPath); err != nil {
		return fmt.Errorf("failed to move directory: %w", err)
	}

	if _, err := executor.Execute("worktree", "repair", newPath); err != nil {
		if rollbackErr := os.Rename(newPath, t.Path); rollbackErr != nil {
			return fmt.Errorf("failed to update worktree registration and also failed to roll back: %w / %w", err, rollbackErr)
		}
		return fmt.Errorf("failed to update worktree registration: %w", err)
	}

	return cleanupEmptyDirs(filepath.Dir(t.Path), repoRoot)
}

func (t *repairTarget) renameBranchToDir(executor *git.Executor) error {
	if _, err := executor.Execute("show-ref", "--verify", "--quiet", "refs/heads/"+t.DirName); err == nil {
		return fmt.Errorf("target branch already exists: %s", t.DirName)
	}

	if _, err := executor.Execute("branch", "-m", t.Branch, t.DirName); err != nil {
		return fmt.Errorf("failed to rename branch: %w", err)
	}

	return nil
}

// cleanupEmptyDirs removes empty parent directories up to repoRoot
func cleanupEmptyDirs(dir, repoRoot string) error {
	for dir != repoRoot && dir != filepath.Dir(dir) {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		if err := os.Remove(dir); err != nil {
			return err
		}
		dir = filepath.Dir(dir)
	}
	return nil
}

// detectCurrentWorktree detects the current worktree name from cwd
func detectCurrentWorktree(cwd, repoRoot string) (string, error) {
	// Check if cwd is within repoRoot
	relPath, err := filepath.Rel(repoRoot, cwd)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("not inside a worktree")
	}

	// Handle hierarchical worktree names (feature/auth -> feature/auth)
	// We need to find the worktree path by walking up until we find the .git file
	worktreePath := cwd

	for {
		gitFile := filepath.Join(worktreePath, ".git")
		info, err := os.Stat(gitFile)
		if err == nil && !info.IsDir() {
			// Found the worktree root (has .git file, not directory)
			break
		}

		parent := filepath.Dir(worktreePath)
		if parent == worktreePath || parent == repoRoot || !strings.HasPrefix(parent, repoRoot) {
			// Reached repo root or can't go further - use first path component as fallback
			parts := strings.Split(relPath, string(filepath.Separator))
			if len(parts) > 0 && parts[0] != "." && parts[0] != ".git" {
				return parts[0], nil
			}
			return "", fmt.Errorf("could not find worktree root")
		}
		worktreePath = parent
	}

	// Get worktree name relative to repo root
	worktreeName, err := filepath.Rel(repoRoot, worktreePath)
	if err != nil {
		return "", err
	}

	return worktreeName, nil
}

// runFixPaths fixes worktree paths after the repository has been moved.
// If externalPaths is provided, those paths are used directly.
// Otherwise, it reads the gitdir files in .git/worktrees/*/gitdir to find the old paths,
// calculates the new paths based on the current repository root, and runs
// git worktree repair to update Git's internal links.
// For external paths, it also moves them back into the baretree structure.
func runFixPaths(repoRoot, bareDir string, executor *git.Executor, wtMgr *worktree.Manager, externalPaths []string, dryRun bool) error {
	var pathsToRepair []string
	var brokenWorktrees []brokenWorktree

	// If external paths are provided, use them directly
	if len(externalPaths) > 0 {
		for _, p := range externalPaths {
			absPath, err := filepath.Abs(p)
			if err != nil {
				return fmt.Errorf("failed to get absolute path for %s: %w", p, err)
			}

			// Check if the path exists
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				return fmt.Errorf("path does not exist: %s", absPath)
			}

			// Check if it looks like a worktree (has .git file)
			gitFile := filepath.Join(absPath, ".git")
			if _, err := os.Stat(gitFile); os.IsNotExist(err) {
				return fmt.Errorf("not a worktree (no .git file): %s", absPath)
			}

			pathsToRepair = append(pathsToRepair, absPath)
		}
	} else {
		// Auto-detect paths from worktrees directory
		worktreesDir := filepath.Join(bareDir, "worktrees")

		// Check if worktrees directory exists
		if _, err := os.Stat(worktreesDir); os.IsNotExist(err) {
			fmt.Println("No worktrees found")
			return nil
		}

		// Read all worktree entries
		entries, err := os.ReadDir(worktreesDir)
		if err != nil {
			return fmt.Errorf("failed to read worktrees directory: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			gitdirFile := filepath.Join(worktreesDir, entry.Name(), "gitdir")
			content, err := os.ReadFile(gitdirFile)
			if err != nil {
				continue // Skip if gitdir file doesn't exist
			}

			oldGitPath := strings.TrimSpace(string(content))
			// oldGitPath is like "/old/path/project/main/.git"
			// We need to extract the worktree relative path

			// Remove the "/.git" suffix to get the worktree path
			oldWorktreePath := strings.TrimSuffix(oldGitPath, "/.git")
			if oldWorktreePath == oldGitPath {
				// No .git suffix found, try without leading slash for Windows compatibility
				oldWorktreePath = strings.TrimSuffix(oldGitPath, "\\.git")
			}

			// Check if the old path still exists (worktree wasn't moved)
			if _, err := os.Stat(oldWorktreePath); err == nil {
				// Old path exists, check if it's still valid
				gitFile := filepath.Join(oldWorktreePath, ".git")
				if _, err := os.Stat(gitFile); err == nil {
					// Worktree exists at old path, no repair needed
					continue
				}
			}

			// Try to find the relative path by looking for common patterns
			// The worktree name is stored in the HEAD file or can be inferred from gitdir
			newPath := inferNewWorktreePath(repoRoot, bareDir, entry.Name(), oldWorktreePath)
			if newPath == "" {
				// Could not find the worktree - it was likely moved to an unknown location
				branchName := getBranchNameFromWorktree(bareDir, entry.Name())
				brokenWorktrees = append(brokenWorktrees, brokenWorktree{
					name:    entry.Name(),
					branch:  branchName,
					oldPath: oldWorktreePath,
				})
				continue
			}

			// Check if the new path exists
			if _, err := os.Stat(newPath); os.IsNotExist(err) {
				branchName := getBranchNameFromWorktree(bareDir, entry.Name())
				brokenWorktrees = append(brokenWorktrees, brokenWorktree{
					name:    entry.Name(),
					branch:  branchName,
					oldPath: oldWorktreePath,
				})
				continue
			}

			// Check if repair is needed (old path != new path)
			newGitPath := filepath.Join(newPath, ".git")
			if oldGitPath == newGitPath {
				continue // Already correct
			}

			pathsToRepair = append(pathsToRepair, newPath)
		}

		// Report broken worktrees that need manual intervention
		if len(brokenWorktrees) > 0 {
			fmt.Printf("Found %d worktree(s) with broken paths:\n\n", len(brokenWorktrees))
			for _, bw := range brokenWorktrees {
				fmt.Printf("  Branch: %s\n", bw.branch)
				fmt.Printf("  Last known path: %s\n", bw.oldPath)
				fmt.Println()
			}
			fmt.Println("To fix these, specify the new location:")
			fmt.Println("  bt repair --fix-paths /new/path/to/worktree")
			fmt.Println()
		}
	}

	if len(pathsToRepair) == 0 && len(brokenWorktrees) == 0 {
		fmt.Println("No worktree paths need fixing")
		return nil
	}

	if len(pathsToRepair) == 0 {
		// Only broken worktrees exist, nothing to auto-fix
		return nil
	}

	fmt.Printf("Found %d worktree path(s) to fix:\n\n", len(pathsToRepair))
	for _, p := range pathsToRepair {
		fmt.Printf("  %s\n", p)
	}
	fmt.Println()

	if dryRun {
		fmt.Println("Dry run - no changes made")
		return nil
	}

	// Run git worktree repair with all paths
	args := append([]string{"worktree", "repair"}, pathsToRepair...)
	if _, err := executor.Execute(args...); err != nil {
		return fmt.Errorf("failed to repair worktree paths: %w", err)
	}

	fmt.Printf("Successfully fixed %d worktree path(s)\n", len(pathsToRepair))

	// For external paths, move them back into baretree structure
	if len(externalPaths) > 0 {
		fmt.Println()
		fmt.Println("Moving worktrees into baretree structure...")

		for _, extPath := range pathsToRepair {
			// Check if this path is outside repoRoot
			relPath, err := filepath.Rel(repoRoot, extPath)
			if err != nil || strings.HasPrefix(relPath, "..") {
				// This is an external worktree, move it back
				branchName := getBranchNameFromPath(bareDir, extPath)
				if branchName == "" {
					fmt.Printf("  Warning: could not determine branch name for %s\n", extPath)
					continue
				}

				targetPath := filepath.Join(repoRoot, branchName)

				if dryRun {
					fmt.Printf("  Would move %s -> %s\n", extPath, targetPath)
					continue
				}

				// Check if target already exists
				if _, err := os.Stat(targetPath); err == nil {
					fmt.Printf("  Warning: target path already exists: %s\n", targetPath)
					continue
				}

				// Create parent directories if needed
				if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
					fmt.Printf("  Warning: failed to create parent directory: %v\n", err)
					continue
				}

				// Move worktree
				if err := os.Rename(extPath, targetPath); err != nil {
					fmt.Printf("  Warning: failed to move worktree: %v\n", err)
					continue
				}

				// Repair Git worktree metadata after move
				if _, err := executor.Execute("worktree", "repair", targetPath); err != nil {
					// Try to rollback
					if rollbackErr := os.Rename(targetPath, extPath); rollbackErr != nil {
						fmt.Printf("  Error: failed to repair and rollback: %v / %v\n", err, rollbackErr)
						continue
					}
					fmt.Printf("  Warning: failed to repair after move, rolled back: %v\n", err)
					continue
				}

				// Apply post-create file configuration (pass nil to discard command output in repair context)
				if _, err := wtMgr.ApplyPostCreateConfig(targetPath, nil); err != nil {
					fmt.Printf("  Warning: failed to apply post-create config: %v\n", err)
				}

				fmt.Printf("  Moved %s -> %s\n", extPath, branchName)
			}
		}

		fmt.Println("Done")
	}

	return nil
}

// getBranchNameFromPath gets the branch name for a worktree at the given path
func getBranchNameFromPath(bareDir, worktreePath string) string {
	// Read the .git file to find the worktree name
	gitFile := filepath.Join(worktreePath, ".git")
	content, err := os.ReadFile(gitFile)
	if err != nil {
		return ""
	}

	// Content is like "gitdir: /path/to/.git/worktrees/test"
	gitdirLine := strings.TrimSpace(string(content))
	if !strings.HasPrefix(gitdirLine, "gitdir: ") {
		return ""
	}

	gitdirPath := strings.TrimPrefix(gitdirLine, "gitdir: ")
	worktreeName := filepath.Base(gitdirPath)

	return getBranchNameFromWorktree(bareDir, worktreeName)
}

// inferNewWorktreePath tries to determine the new worktree path based on the
// current repository root and available information.
func inferNewWorktreePath(repoRoot, bareDir, worktreeName, oldWorktreePath string) string {
	// First, try to read the HEAD file to get the branch name
	headFile := filepath.Join(bareDir, "worktrees", worktreeName, "HEAD")
	content, err := os.ReadFile(headFile)
	if err == nil {
		headContent := strings.TrimSpace(string(content))
		// HEAD content is like "ref: refs/heads/feature/test" or a commit hash
		if strings.HasPrefix(headContent, "ref: refs/heads/") {
			branchName := strings.TrimPrefix(headContent, "ref: refs/heads/")
			newPath := filepath.Join(repoRoot, branchName)
			if _, err := os.Stat(newPath); err == nil {
				return newPath
			}
		}
	}

	// Fallback: try using the worktree directory name directly
	// This works for simple branch names like "main" or "develop"
	newPath := filepath.Join(repoRoot, worktreeName)
	if _, err := os.Stat(newPath); err == nil {
		return newPath
	}

	// Try to extract relative path from old path
	// Look for common directory names that might be the repo root
	parts := strings.Split(oldWorktreePath, string(filepath.Separator))
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == filepath.Base(repoRoot) {
			// Found a matching directory name, try to construct the relative path
			relParts := parts[i+1:]
			if len(relParts) > 0 {
				newPath := filepath.Join(repoRoot, filepath.Join(relParts...))
				if _, err := os.Stat(newPath); err == nil {
					return newPath
				}
			}
			break
		}
	}

	return ""
}

// getBranchNameFromWorktree extracts the branch name from a worktree's HEAD file
func getBranchNameFromWorktree(bareDir, worktreeName string) string {
	headFile := filepath.Join(bareDir, "worktrees", worktreeName, "HEAD")
	content, err := os.ReadFile(headFile)
	if err != nil {
		return worktreeName // Fallback to directory name
	}

	headContent := strings.TrimSpace(string(content))
	if strings.HasPrefix(headContent, "ref: refs/heads/") {
		return strings.TrimPrefix(headContent, "ref: refs/heads/")
	}

	// Detached HEAD, return directory name
	return worktreeName
}
