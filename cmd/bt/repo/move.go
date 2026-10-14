package repo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/amaya382/baretree/internal/global"
	"github.com/amaya382/baretree/internal/repository"
	"github.com/spf13/cobra"
)

var moveCmd = &cobra.Command{
	Use:     "move <repository> <destination>",
	Aliases: []string{"mv"},
	Short:   "Move or rename a baretree repository",
	Long: `Move or rename a baretree repository and fix its worktree paths.

The repository can be specified as:
  - Repository name only: baretree
  - Organization/repository: amaya382/baretree
  - Full path: github.com/amaya382/baretree

The destination can be specified as:
  - Same form as the repository: only that part of the path is replaced
      name only               -> renames the repository
      organization/repository -> moves to another organization
      full path               -> path relative to the baretree root
  - Absolute path: /path/to/new-name

The destination must not exist, and moving across filesystems is not supported.

Examples:
  bt repo move baretree new-name                           # -> github.com/amaya382/new-name
  bt repo mv amaya382/baretree other-org/baretree          # -> github.com/other-org/baretree
  bt repo mv github.com/amaya382/baretree gitlab.com/a/b   # -> gitlab.com/a/b
  bt repo mv baretree /path/to/baretree`,
	Args:              cobra.ExactArgs(2),
	RunE:              runRepoMove,
	ValidArgsFunction: completeMoveArgs,
}

func init() {
	moveCmd.GroupID = groupCross
	Cmd.AddCommand(moveCmd)
}

func completeMoveArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeRepositoryNames(false)(cmd, args, toComplete)
}

func runRepoMove(cmd *cobra.Command, args []string) error {
	query, destArg := args[0], args[1]

	cfg, err := global.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	roots := cfg.Roots
	if len(roots) == 0 {
		return fmt.Errorf("no root directories configured")
	}

	repos, err := global.ScanRepositories(roots)
	if err != nil {
		return fmt.Errorf("failed to scan repositories: %w", err)
	}

	match, ambiguousMatches, err := resolveRepository(repos, query)
	if err != nil {
		if len(ambiguousMatches) > 0 {
			fmt.Fprintf(os.Stderr, "Ambiguous repository name '%s'. Did you mean one of these?\n\n", query)
			for _, repo := range ambiguousMatches {
				fmt.Fprintf(os.Stderr, "  %s\n", repo.RelativePath)
			}
			fmt.Fprintln(os.Stderr)
			return fmt.Errorf("ambiguous repository name")
		}
		return err
	}
	source := match.Path

	destination, err := resolveMoveDestination(match, query, destArg)
	if err != nil {
		return err
	}

	if err := validateMove(source, destination); err != nil {
		return err
	}

	fmt.Printf("Moving repository:\n")
	fmt.Printf("  Source: %s\n", source)
	fmt.Printf("  Destination: %s\n", destination)

	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return fmt.Errorf("failed to create destination parent directory: %w", err)
	}

	if err := os.Rename(source, destination); err != nil {
		if errors.Is(err, syscall.EXDEV) {
			return fmt.Errorf("cannot move repository across filesystems: %s -> %s", source, destination)
		}
		return fmt.Errorf("failed to move repository: %w", err)
	}

	if err := updateWorktreePaths(source, destination); err != nil {
		if rollbackErr := os.Rename(destination, source); rollbackErr != nil {
			return fmt.Errorf("failed to update worktree paths: %w (rollback also failed: %v)", err, rollbackErr)
		}
		if restoreErr := updateWorktreePaths(destination, source); restoreErr != nil {
			return fmt.Errorf("failed to update worktree paths: %w (moved back, but restoring worktree paths failed: %v)", err, restoreErr)
		}
		return fmt.Errorf("failed to update worktree paths (moved back): %w", err)
	}

	cleanupEmptyParents(source, roots)

	fmt.Printf("\n✓ Repository moved: %s\n", destination)

	if !isUnderAnyRoot(destination, roots) {
		fmt.Printf("\nNote: the destination is outside the baretree root, so 'bt repo list' and 'bt go' will not find it.\n")
	}

	return nil
}

// resolveMoveDestination replaces as many trailing path components of the
// repository as the query has, so the destination takes the same form as the query.
func resolveMoveDestination(match *global.RepoInfo, query, destArg string) (string, error) {
	destination := global.ExpandTilde(destArg)
	if filepath.IsAbs(destination) {
		return filepath.Clean(destination), nil
	}

	relParts := strings.Split(filepath.ToSlash(match.RelativePath), "/")
	root := filepath.Clean(strings.TrimSuffix(match.Path, match.RelativePath))
	queryParts := strings.Split(strings.Trim(filepath.ToSlash(query), "/"), "/")
	destParts := strings.Split(strings.Trim(filepath.ToSlash(destination), "/"), "/")

	for _, part := range destParts {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("invalid destination: %s", destArg)
		}
	}

	if len(queryParts) >= len(relParts) {
		return filepath.Join(append([]string{root}, destParts...)...), nil
	}

	if len(destParts) != len(queryParts) {
		form := "a repository name"
		if len(queryParts) == 2 {
			form = "an organization/repository"
		}
		return "", fmt.Errorf("destination must be %s like the source '%s', or an absolute path: %s", form, query, destArg)
	}

	kept := relParts[:len(relParts)-len(queryParts)]
	return filepath.Join(append(append([]string{root}, kept...), destParts...)...), nil
}

func validateMove(source, destination string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	if isSubPath(source, cwd) {
		return fmt.Errorf("cannot move repository while inside it: %s", source)
	}

	if _, err := os.Lstat(destination); err == nil {
		return fmt.Errorf("destination already exists: %s", destination)
	}

	if isSubPath(source, destination) || isSubPath(destination, source) {
		return fmt.Errorf("source and destination paths overlap")
	}

	for dir := filepath.Dir(destination); ; dir = filepath.Dir(dir) {
		if repository.IsBaretreeRepo(dir) {
			return fmt.Errorf("destination is inside another baretree repository: %s", dir)
		}
		if filepath.Dir(dir) == dir {
			break
		}
	}

	return nil
}

func isUnderAnyRoot(path string, roots []string) bool {
	for _, root := range roots {
		if isSubPath(root, path) {
			return true
		}
	}
	return false
}
