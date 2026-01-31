package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/amaya382/baretree/internal/git"
	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	listJSON  bool
	listPaths bool
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all worktrees in the current repository",
	Long: `List all worktrees in the current repository.

Shows worktree path, branch name, HEAD commit, and management status.

Indicators (first two columns):
  * = Current worktree (where you are now)
  @ = Default worktree (configured default branch)

Examples:
  bt list
  bt ls
  bt list --json
  bt list --paths`,
	RunE: runList,
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
	listCmd.Flags().BoolVar(&listPaths, "paths", false, "Output only paths (for scripting)")
}

func runList(cmd *cobra.Command, args []string) error {
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

	if len(worktrees) == 0 {
		fmt.Println("No worktrees found")
		return nil
	}

	// Output based on format
	if listJSON {
		return outputJSON(worktrees)
	}

	if listPaths {
		return outputPaths(worktrees)
	}

	return outputTable(worktrees, wtMgr)
}

func outputJSON(worktrees []git.Worktree) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(worktrees)
}

func outputPaths(worktrees []git.Worktree) error {
	for _, wt := range worktrees {
		fmt.Println(wt.Path)
	}
	return nil
}

func outputTable(worktrees []git.Worktree, wtMgr *worktree.Manager) error {
	// Get current working directory to detect current worktree
	cwd, _ := os.Getwd()

	// Calculate column widths
	maxBranchLen := 10
	for _, wt := range worktrees {
		if len(wt.Branch) > maxBranchLen {
			maxBranchLen = len(wt.Branch)
		}
	}

	// Print worktrees
	for _, wt := range worktrees {
		// Determine indicators: * for current, @ for default (separate columns)
		currentMark := " "
		defaultMark := " "
		isCurrent := strings.HasPrefix(cwd, wt.Path+string(os.PathSeparator)) || cwd == wt.Path
		if isCurrent {
			currentMark = "*"
		}
		if wt.IsMain {
			defaultMark = "@"
		}

		// Determine if managed
		managed := wtMgr.IsManaged(wt.Path)
		status := "[M]"
		if !managed {
			status = "[U]"
		}

		// Format branch name
		branchName := wt.Branch
		if branchName == "" {
			branchName = "(detached)"
		}

		// Format HEAD (short hash)
		headShort := wt.Head
		if len(headShort) > 7 {
			headShort = headShort[:7]
		}

		// Print line with two indicator columns
		fmt.Printf("%s%s %-*s  %s %s\n",
			currentMark,
			defaultMark,
			maxBranchLen, branchName,
			headShort,
			status,
		)
	}

	return nil
}
