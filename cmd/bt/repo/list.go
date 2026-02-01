package repo

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/amaya382/baretree/internal/global"
	"github.com/spf13/cobra"
)

var (
	listPaths bool
	listJSON  bool
)

var listCmd = &cobra.Command{
	Use:     "list [query]",
	Aliases: []string{"ls"},
	Short:   "List all baretree repositories under root [bt repos]",
	Long: `List all baretree repositories under the configured root directory.

By default, shows the relative path (e.g., github.com/user/repo).
Use --paths to show full absolute paths.
Use --json for JSON output.

Examples:
  bt repo list
  bt repo ls
  bt repo list --paths
  bt repo list github.com
  bt repo list --json`,
	Args:              cobra.MaximumNArgs(1),
	RunE:              runList,
	ValidArgsFunction: completeRepositoryNames(false),
}

func init() {
	listCmd.Flags().BoolVarP(&listPaths, "paths", "p", false, "Show full paths")
	listCmd.Flags().BoolVarP(&listJSON, "json", "j", false, "Output as JSON")
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := global.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	repos, err := global.ScanRepositories(cfg.Roots)
	if err != nil {
		return fmt.Errorf("failed to scan repositories: %w", err)
	}

	// Filter by query if provided
	if len(args) > 0 {
		repos = global.FilterRepositories(repos, args[0])
	}

	if listJSON {
		return outputJSON(repos)
	}

	for _, repo := range repos {
		if listPaths {
			fmt.Println(repo.Path)
		} else {
			fmt.Println(repo.RelativePath)
		}
	}

	return nil
}

type jsonRepo struct {
	Path         string `json:"path"`
	RelativePath string `json:"relative_path"`
	Name         string `json:"name"`
}

func outputJSON(repos []global.RepoInfo) error {
	output := make([]jsonRepo, len(repos))
	for i, repo := range repos {
		output[i] = jsonRepo{
			Path:         repo.Path,
			RelativePath: repo.RelativePath,
			Name:         repo.Name,
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
