package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/worktree"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show baretree repository status (worktrees, config, shared files)",
	Long: `Display detailed information about the baretree repository.

Shows:
  - Repository root and bare repository location
  - Configuration file location
  - All worktrees with management status
  - Warnings for unmanaged worktrees
  - Configured shared files

Example:
  bt status`,
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Find repository root
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("not in a baretree repository: %w", err)
	}

	// Get bare repository path
	bareDir, err := repository.GetBareRepoPath(repoRoot)
	if err != nil {
		return err
	}

	// Load config and create manager
	mgr, err := repository.NewManager(repoRoot)
	if err != nil {
		return err
	}

	wtMgr := worktree.NewManager(repoRoot, bareDir, mgr.Config)

	// Get all worktrees
	worktrees, err := wtMgr.List()
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Print repository information
	fmt.Println("Repository Information:")
	fmt.Printf("  Root:          %s\n", repoRoot)
	fmt.Printf("  Bare repo:     %s\n", bareDir)
	fmt.Printf("  Default branch: %s\n", mgr.Config.Repository.DefaultBranch)
	fmt.Println()

	// Check if default branch worktree exists
	defaultBranchPath := filepath.Join(repoRoot, mgr.Config.Repository.DefaultBranch)
	defaultBranchMissing := false
	if _, err := os.Stat(defaultBranchPath); os.IsNotExist(err) {
		defaultBranchMissing = true
	}

	// Check for broken worktrees (moved to unknown location) before listing
	brokenWorktrees := detectBrokenWorktrees(bareDir)
	brokenBranches := make(map[string]brokenWorktree)
	for _, bw := range brokenWorktrees {
		brokenBranches[bw.branch] = bw
	}

	// Print worktrees
	fmt.Println("Worktrees:")
	var unmanagedWorktrees []string

	// Determine which worktree we're currently in
	cwdAbs, _ := filepath.Abs(cwd)

	// Collect all worktree paths for nested check
	var allWorktreePaths []string
	for _, wt := range worktrees {
		if !wt.IsBare {
			allWorktreePaths = append(allWorktreePaths, wt.Path)
		}
	}

	for _, wt := range worktrees {
		prefix := " "
		// Check if current directory is within this worktree
		wtPathAbs, _ := filepath.Abs(wt.Path)
		if isPathWithin(cwdAbs, wtPathAbs) {
			prefix = "@"
		}

		branchName := wt.Branch
		if branchName == "" {
			branchName = "(detached)"
		}

		// Check if this worktree is broken (path doesn't exist)
		if _, isBroken := brokenBranches[branchName]; isBroken {
			relPath, _ := filepath.Rel(repoRoot, wt.Path)
			fmt.Printf("  %s %-20s  %-30s  %s%s\n",
				prefix,
				branchName,
				relPath,
				"[Broken]",
				" ⚠️",
			)
			continue
		}

		// Check if managed and not nested inside another worktree
		managed := wtMgr.IsManaged(wt.Path) && !wtMgr.IsNestedInWorktree(wt.Path, allWorktreePaths)
		status := "[Managed]"
		statusSymbol := ""

		if !managed {
			status = "[Unmanaged]"
			statusSymbol = " ⚠️"
			unmanagedWorktrees = append(unmanagedWorktrees, wt.Path)
		}

		relPath, _ := filepath.Rel(repoRoot, wt.Path)

		fmt.Printf("  %s %-20s  %-30s  %s%s\n",
			prefix,
			branchName,
			relPath,
			status,
			statusSymbol,
		)
	}

	fmt.Println()

	// Print warnings
	if len(unmanagedWorktrees) > 0 || len(brokenWorktrees) > 0 || defaultBranchMissing {
		fmt.Println("Warnings:")
		if defaultBranchMissing {
			fmt.Printf("  - Default branch worktree '%s' does not exist\n", mgr.Config.Repository.DefaultBranch)
			fmt.Printf("    Expected path: %s\n", defaultBranchPath)
			fmt.Printf("    Fix with: git config --file %s/config baretree.defaultbranch <branch>\n", bareDir)
		}
		for _, path := range unmanagedWorktrees {
			fmt.Printf("  - Worktree at %s is outside managed directory\n", path)
			fmt.Printf("    Run 'bt repair %s' to fix it\n", path)
		}
		for _, bw := range brokenWorktrees {
			fmt.Printf("  - Worktree '%s' has broken path (moved or deleted)\n", bw.branch)
			fmt.Printf("    Last known: %s\n", bw.oldPath)
			fmt.Printf("    Run 'bt repair --fix-paths /new/path' to fix it\n")
		}
		fmt.Println()
	}

	// Print shared files configuration with status
	if len(mgr.Config.Shared) > 0 {
		fmt.Println("Shared files:")

		// Get shared status
		statuses, err := wtMgr.GetSharedStatus()
		if err != nil {
			// Fallback to simple listing
			for _, shared := range mgr.Config.Shared {
				modeStr := ""
				if shared.Managed {
					modeStr = ", managed"
				}
				fmt.Printf("  %s (%s%s)\n", shared.Source, shared.Type, modeStr)
			}
		} else {
			for _, status := range statuses {
				modeStr := ""
				if status.Managed {
					modeStr = ", managed"
				}
				fmt.Printf("  %s (%s%s)\n", status.Source, status.Type, modeStr)

				// Show source info for non-managed
				if !status.Managed && status.SourceWorktree != "" {
					fmt.Printf("    - source: %s\n", status.SourceWorktree)
				}

				// Show applied worktrees
				if len(status.Applied) > 0 {
					fmt.Printf("    + applied: %s\n", joinWorktrees(status.Applied))
				}

				// Show missing worktrees
				if len(status.Missing) > 0 {
					fmt.Printf("    x missing: %s\n", joinWorktrees(status.Missing))
				}
			}
		}
	} else {
		fmt.Println("No shared files configured.")
		fmt.Println("  Use 'bt shared add <file> --type symlink' to configure shared files.")
	}

	return nil
}

// joinWorktrees joins worktree names with commas
func joinWorktrees(names []string) string {
	if len(names) == 0 {
		return "(none)"
	}
	result := names[0]
	for i := 1; i < len(names); i++ {
		result += ", " + names[i]
	}
	return result
}

// isPathWithin checks if childPath is within parentPath
func isPathWithin(childPath, parentPath string) bool {
	// Clean and ensure trailing separator for accurate prefix matching
	parent := filepath.Clean(parentPath) + string(filepath.Separator)
	child := filepath.Clean(childPath) + string(filepath.Separator)

	return strings.HasPrefix(child, parent)
}

// detectBrokenWorktrees finds worktrees that have been moved to unknown locations
func detectBrokenWorktrees(bareDir string) []brokenWorktree {
	var broken []brokenWorktree

	worktreesDir := filepath.Join(bareDir, "worktrees")

	// Check if worktrees directory exists
	if _, err := os.Stat(worktreesDir); os.IsNotExist(err) {
		return broken
	}

	// Read all worktree entries
	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		return broken
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		gitdirFile := filepath.Join(worktreesDir, entry.Name(), "gitdir")
		content, err := os.ReadFile(gitdirFile)
		if err != nil {
			continue
		}

		oldGitPath := strings.TrimSpace(string(content))
		oldWorktreePath := strings.TrimSuffix(oldGitPath, "/.git")
		if oldWorktreePath == oldGitPath {
			oldWorktreePath = strings.TrimSuffix(oldGitPath, "\\.git")
		}

		// Check if the worktree path still exists
		if _, err := os.Stat(oldWorktreePath); os.IsNotExist(err) {
			branchName := getBranchNameFromWorktree(bareDir, entry.Name())
			broken = append(broken, brokenWorktree{
				name:    entry.Name(),
				branch:  branchName,
				oldPath: oldWorktreePath,
			})
		}
	}

	return broken
}
