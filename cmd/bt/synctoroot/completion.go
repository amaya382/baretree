package synctoroot

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/amaya382/baretree/internal/repository"
	"github.com/spf13/cobra"
)

// completeSourceFiles completes files/directories in the default branch worktree
func completeSourceFiles(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		// Second argument (target) - no completion
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	completions := []string{}

	// Find repository root
	cwd, err := os.Getwd()
	if err != nil {
		return completions, cobra.ShellCompDirectiveNoFileComp
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return completions, cobra.ShellCompDirectiveNoFileComp
	}

	// Load config to get default branch
	mgr, err := repository.NewManager(repoRoot)
	if err != nil {
		return completions, cobra.ShellCompDirectiveNoFileComp
	}

	defaultBranch := mgr.Config.Repository.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	// Get default branch worktree path
	mainWorktree := filepath.Join(repoRoot, defaultBranch)
	if _, err := os.Stat(mainWorktree); os.IsNotExist(err) {
		return completions, cobra.ShellCompDirectiveNoFileComp
	}

	// List files/directories in the main worktree
	completions = listFilesForCompletion(mainWorktree, toComplete)

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// listFilesForCompletion lists files/directories for completion
func listFilesForCompletion(basePath, prefix string) []string {
	var results []string

	// Determine the directory to search
	searchDir := basePath
	searchPrefix := prefix

	if prefix != "" {
		// If prefix contains a directory separator, search in that directory
		if strings.Contains(prefix, "/") {
			dir := filepath.Dir(prefix)
			searchDir = filepath.Join(basePath, dir)
			searchPrefix = filepath.Base(prefix)
		}
	}

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return results
	}

	for _, entry := range entries {
		name := entry.Name()

		// Skip .git directory
		if name == ".git" {
			continue
		}

		// Build full relative path
		var fullPath string
		if prefix != "" && strings.Contains(prefix, "/") {
			fullPath = filepath.Join(filepath.Dir(prefix), name)
		} else {
			fullPath = name
		}

		// Filter by prefix
		if searchPrefix != "" && !strings.HasPrefix(strings.ToLower(name), strings.ToLower(searchPrefix)) {
			continue
		}

		// Add trailing slash for directories
		if entry.IsDir() {
			fullPath += "/"
		}

		results = append(results, fullPath)
	}

	return results
}

// completeConfiguredSources completes configured sync-to-root sources
func completeConfiguredSources(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	completions := []string{}

	// Find repository root
	cwd, err := os.Getwd()
	if err != nil {
		return completions, cobra.ShellCompDirectiveNoFileComp
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return completions, cobra.ShellCompDirectiveNoFileComp
	}

	// Load config
	mgr, err := repository.NewManager(repoRoot)
	if err != nil {
		return completions, cobra.ShellCompDirectiveNoFileComp
	}

	// Filter by prefix
	for _, action := range mgr.Config.SyncToRoot {
		if toComplete == "" || strings.HasPrefix(strings.ToLower(action.Source), strings.ToLower(toComplete)) {
			completions = append(completions, action.Source)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
