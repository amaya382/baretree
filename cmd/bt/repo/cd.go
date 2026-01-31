package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/amaya382/baretree/internal/global"
	"github.com/amaya382/baretree/internal/repository"
	"github.com/spf13/cobra"
)

const repoHistoryFile = ".baretree_repo_history"

var cdCmd = &cobra.Command{
	Use:   "cd <repository>",
	Short: "Output repository path [bt go]",
	Long: `Output the absolute path to a repository for use with shell integration.

This command should be used with the shell function installed by 'bt shell-init'.

The repository can be specified as:
  - Repository name only: baretree
  - Organization/repository: amaya382/baretree
  - Full path: github.com/amaya382/baretree
  - - (dash) to go to previous repository

Resolution order (for partial matches):
  1. Exact match on full relative path
  2. Exact match on org/repo
  3. Exact match on repo name
  4. Partial match (contains query)

Examples:
  bt repo cd baretree                    # Match by repo name
  bt repo cd amaya382/baretree           # Match by org/repo
  bt repo cd github.com/amaya382/baretree # Match by full path
  bt repo cd -                           # Go to previous repository`,
	Args: cobra.ExactArgs(1),
	RunE: runRepoCd,
}

func init() {
	cdCmd.GroupID = groupCross
	Cmd.AddCommand(cdCmd)
}

func runRepoCd(cmd *cobra.Command, args []string) error {
	query := args[0]

	// Handle special case: previous directory
	if query == "-" {
		prevDir, err := getRepoPreviousDirectory()
		if err != nil {
			return fmt.Errorf("no previous repository: %w", err)
		}

		// Save current directory before changing
		cwd, _ := os.Getwd()
		if err := saveRepoPreviousDirectory(cwd); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save directory history: %v\n", err)
		}

		fmt.Println(prevDir)
		return nil
	}

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
				fmt.Fprintf(os.Stderr, "  bt repo cd %s\n", repo.RelativePath)
			}
			fmt.Fprintln(os.Stderr)
			return fmt.Errorf("ambiguous repository name")
		}
		return err
	}

	// Save current directory before changing
	cwd, _ := os.Getwd()
	if err := saveRepoPreviousDirectory(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save directory history: %v\n", err)
	}

	// Determine the target path (default worktree if available)
	targetPath := match.Path

	// Try to get the default worktree directory
	mgr, err := repository.NewManager(match.Path)
	if err == nil && mgr.Config.Repository.DefaultBranch != "" {
		defaultWorktreePath := filepath.Join(match.Path, mgr.Config.Repository.DefaultBranch)
		if _, err := os.Stat(defaultWorktreePath); err == nil {
			targetPath = defaultWorktreePath
		}
	}

	// Output the path (shell function will use this)
	fmt.Println(targetPath)
	return nil
}

// resolveRepository finds a repository matching the query.
// Resolution order:
// 1. Exact match on full relative path (github.com/org/repo)
// 2. Exact match on org/repo
// 3. Exact match on repo name (returns error if multiple matches)
// 4. Partial match (returns error if multiple matches)
// Returns the matching repository, list of ambiguous matches (if any), and error.
func resolveRepository(repos []global.RepoInfo, query string) (*global.RepoInfo, []global.RepoInfo, error) {
	query = strings.ToLower(query)
	queryParts := strings.Split(query, "/")

	// 1. Exact match on full relative path
	for i := range repos {
		if strings.ToLower(repos[i].RelativePath) == query {
			return &repos[i], nil, nil
		}
	}

	// 2. Exact match on org/repo (last two components)
	if len(queryParts) == 2 {
		for i := range repos {
			relParts := strings.Split(repos[i].RelativePath, string(filepath.Separator))
			if len(relParts) >= 2 {
				orgRepo := strings.ToLower(relParts[len(relParts)-2] + "/" + relParts[len(relParts)-1])
				if orgRepo == query {
					return &repos[i], nil, nil
				}
			}
		}
	}

	// 3. Exact match on repo name only - collect all matches
	if len(queryParts) == 1 {
		var matches []global.RepoInfo
		for i := range repos {
			if strings.ToLower(repos[i].Name) == query {
				matches = append(matches, repos[i])
			}
		}
		if len(matches) == 1 {
			return &matches[0], nil, nil
		}
		if len(matches) > 1 {
			return nil, matches, fmt.Errorf("ambiguous repository name '%s': %d matches found", query, len(matches))
		}
	}

	// 4. Partial match (contains query) - collect all matches
	var matches []global.RepoInfo
	for i := range repos {
		if strings.Contains(strings.ToLower(repos[i].RelativePath), query) ||
			strings.Contains(strings.ToLower(repos[i].Name), query) {
			matches = append(matches, repos[i])
		}
	}
	if len(matches) == 1 {
		return &matches[0], nil, nil
	}
	if len(matches) > 1 {
		return nil, matches, fmt.Errorf("ambiguous repository name '%s': %d matches found", query, len(matches))
	}

	return nil, nil, fmt.Errorf("repository not found: %s", query)
}

func getRepoPreviousDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	historyPath := filepath.Join(homeDir, repoHistoryFile)
	data, err := os.ReadFile(historyPath)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func saveRepoPreviousDirectory(dir string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	historyPath := filepath.Join(homeDir, repoHistoryFile)
	return os.WriteFile(historyPath, []byte(dir), 0644)
}
