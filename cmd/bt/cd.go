package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/worktree"
	"github.com/spf13/cobra"
)

const historyFile = ".baretree_history"

var cdCmd = &cobra.Command{
	Use:   "cd [worktree-name]",
	Short: "Change to a worktree directory",
	Long: `Output the absolute path to a worktree directory for use with shell integration.

This command should be used with the shell function installed by 'bt shell-init'.

The worktree name can be:
  - Branch name (e.g., feature/auth)
  - Directory name relative to repo root
  - @ for default worktree
  - (empty) for current worktree root
  - - (dash) to go to previous worktree

Setup:
  Add to your shell configuration (~/.bashrc or ~/.zshrc):
    eval "$(bt shell-init bash)"   # for bash
    eval "$(bt shell-init zsh)"    # for zsh

Usage (after shell setup):
  bt cd feature/auth    # Change to feature/auth worktree
  bt cd @               # Change to default worktree
  bt cd                 # Change to current worktree root
  bt cd -               # Change to previous worktree

Examples:
  bt cd feature/test
  bt cd @
  bt cd -`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCd,
}

func runCd(cmd *cobra.Command, args []string) error {
	var targetName string
	if len(args) > 0 {
		targetName = args[0]
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

	var targetPath string

	// Handle special case: previous directory
	if targetName == "-" {
		targetPath, err = getPreviousDirectory()
		if err != nil {
			return fmt.Errorf("no previous directory: %w", err)
		}
	} else {
		// Resolve worktree (pass cwd for empty name resolution)
		targetPath, err = wtMgr.ResolveFromCwd(targetName, cwd)
		if err != nil {
			var ambiguousErr *worktree.AmbiguousMatchError
			if errors.As(err, &ambiguousErr) {
				fmt.Fprintf(os.Stderr, "Ambiguous worktree name '%s'. Did you mean one of these?\n\n", ambiguousErr.Name)
				for _, wt := range ambiguousErr.Matches {
					relPath, _ := filepath.Rel(ambiguousErr.RepoRoot, wt.Path)
					fmt.Fprintf(os.Stderr, "  bt cd %s\n", relPath)
				}
				fmt.Fprintln(os.Stderr)
				return fmt.Errorf("ambiguous worktree name")
			}
			return err
		}
	}

	// Save current directory to history
	if err := savePreviousDirectory(cwd); err != nil {
		// Non-fatal, just warn
		fmt.Fprintf(os.Stderr, "Warning: failed to save directory history: %v\n", err)
	}

	// Output the path (shell function will use this)
	fmt.Println(targetPath)

	return nil
}

func getPreviousDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	historyPath := filepath.Join(homeDir, historyFile)
	data, err := os.ReadFile(historyPath)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func savePreviousDirectory(dir string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	historyPath := filepath.Join(homeDir, historyFile)
	return os.WriteFile(historyPath, []byte(dir), 0644)
}
