package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/worktree"
	"github.com/spf13/cobra"
)

// completeWorktreeNames returns worktree names for shell completion.
// If includeSpecial is true, it also includes @ (default worktree) and - (previous).
// Results are ordered with prefix matches first, then substring matches.
func completeWorktreeNames(includeSpecial bool) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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

		// Get bare repository path
		bareDir, err := repository.GetBareRepoPath(repoRoot)
		if err != nil {
			return completions, cobra.ShellCompDirectiveNoFileComp
		}

		// Load config and create manager
		mgr, err := repository.NewManager(repoRoot)
		if err != nil {
			return completions, cobra.ShellCompDirectiveNoFileComp
		}

		wtMgr := worktree.NewManager(repoRoot, bareDir, mgr.Config)

		// Get worktree list
		worktrees, err := wtMgr.List()
		if err != nil {
			return completions, cobra.ShellCompDirectiveNoFileComp
		}

		// Collect worktree names
		var names []string
		for _, wt := range worktrees {
			// Use relative path from repo root as completion
			relPath, err := filepath.Rel(repoRoot, wt.Path)
			if err != nil {
				continue
			}
			names = append(names, relPath)
		}

		// Filter and order by prefix match first, then substring match
		completions = filterWithPrefixPriority(names, toComplete)

		// Add special completions (only when no filter or matches special chars)
		if includeSpecial && (toComplete == "" || toComplete == "@" || toComplete == "-") {
			if toComplete == "" {
				completions = append(completions, "@", "-")
			} else if toComplete == "@" {
				completions = append(completions, "@")
			} else if toComplete == "-" {
				completions = append(completions, "-")
			}
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

// filterWithPrefixPriority filters strings by query with prefix matches first.
// If query is empty, returns all items unchanged.
func filterWithPrefixPriority(items []string, query string) []string {
	if query == "" {
		return items
	}

	queryLower := strings.ToLower(query)
	var prefixMatches []string
	var substringMatches []string

	for _, item := range items {
		itemLower := strings.ToLower(item)
		// Check for prefix match
		if strings.HasPrefix(itemLower, queryLower) {
			prefixMatches = append(prefixMatches, item)
		} else if strings.Contains(itemLower, queryLower) {
			// Substring match (not a prefix match)
			substringMatches = append(substringMatches, item)
		}
	}

	// Return prefix matches first, then substring matches
	return append(prefixMatches, substringMatches...)
}

// completeWorktreeThenPath returns worktree names for the first argument,
// then falls back to file path completion for subsequent arguments.
func completeWorktreeThenPath(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		// First argument: complete worktree names
		return completeWorktreeNames(true)(cmd, args, toComplete)
	}
	// Subsequent arguments: default file path completion
	return nil, cobra.ShellCompDirectiveDefault
}

// completeWorktreeForRename provides completion for the rename command.
// First argument: worktree name (optional if renaming current worktree)
// Second argument: new name (no completion)
func completeWorktreeForRename(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		// First argument: complete worktree names
		return completeWorktreeNames(false)(cmd, args, toComplete)
	}
	// Second argument: new name - no completion needed
	return nil, cobra.ShellCompDirectiveNoFileComp
}
