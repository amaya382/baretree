package repo

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/amaya382/baretree/internal/global"
	"github.com/spf13/cobra"
)

var (
	repoRemoveForce bool
)

var removeCmd = &cobra.Command{
	Use:     "remove <repository>",
	Aliases: []string{"rm"},
	Short:   "Remove a baretree repository",
	Long: `Remove a baretree repository and all its worktrees.

The repository can be specified as:
  - Repository name only: baretree
  - Organization/repository: amaya382/baretree
  - Full path: github.com/amaya382/baretree

This will permanently delete the entire repository directory including:
  - All worktrees
  - The bare repository (.bare)
  - All local branches and history

Examples:
  bt repo remove baretree
  bt repo rm amaya382/baretree
  bt repo rm github.com/amaya382/baretree --force`,
	Args: cobra.ExactArgs(1),
	RunE: runRepoRemove,
}

func init() {
	removeCmd.Flags().BoolVarP(&repoRemoveForce, "force", "f", false, "Skip confirmation prompt")
	removeCmd.GroupID = groupCross
	Cmd.AddCommand(removeCmd)
}

func runRepoRemove(cmd *cobra.Command, args []string) error {
	query := args[0]

	// Load global config
	cfg, err := global.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get root directories
	roots := cfg.Roots
	if len(roots) == 0 {
		return fmt.Errorf("no root directories configured")
	}

	// Scan for repositories
	repos, err := global.ScanRepositories(roots)
	if err != nil {
		return fmt.Errorf("failed to scan repositories: %w", err)
	}

	if len(repos) == 0 {
		return fmt.Errorf("no repositories found")
	}

	// Find matching repository
	match, ambiguousMatches, err := resolveRepository(repos, query)
	if err != nil {
		if len(ambiguousMatches) > 0 {
			fmt.Fprintf(os.Stderr, "Ambiguous repository name '%s'. Did you mean one of these?\n\n", query)
			for _, repo := range ambiguousMatches {
				fmt.Fprintf(os.Stderr, "  bt repo remove %s\n", repo.RelativePath)
			}
			fmt.Fprintln(os.Stderr)
			return fmt.Errorf("ambiguous repository name")
		}
		return err
	}

	// Check if we're currently inside the repository
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if strings.HasPrefix(cwd, match.Path) {
		return fmt.Errorf("cannot remove repository while inside it: %s", match.Path)
	}

	// Confirm deletion unless --force is specified
	if !repoRemoveForce {
		fmt.Printf("This will permanently delete the repository:\n")
		fmt.Printf("  Path: %s\n", match.Path)
		fmt.Printf("  Name: %s\n\n", match.RelativePath)
		fmt.Printf("Are you sure? [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	fmt.Printf("Removing repository %s...\n", match.RelativePath)

	// Remove the repository directory
	if err := os.RemoveAll(match.Path); err != nil {
		return fmt.Errorf("failed to remove repository: %w", err)
	}

	// Try to clean up empty parent directories
	cleanupEmptyParents(match.Path, roots)

	fmt.Printf("✓ Repository removed: %s\n", match.RelativePath)

	return nil
}

// cleanupEmptyParents removes empty parent directories up to (but not including) the root
func cleanupEmptyParents(path string, roots []string) {
	parent := filepath.Dir(path)

	for {
		// Check if we've reached a root directory
		isRoot := false
		for _, root := range roots {
			if parent == root {
				isRoot = true
				break
			}
		}
		if isRoot {
			break
		}

		// Try to remove the directory (will fail if not empty)
		if err := os.Remove(parent); err != nil {
			break
		}

		parent = filepath.Dir(parent)
	}
}
