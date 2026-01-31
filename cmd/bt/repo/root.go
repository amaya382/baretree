package repo

import (
	"fmt"

	"github.com/amaya382/baretree/internal/global"
	"github.com/spf13/cobra"
)

var (
	rootAll bool
)

var rootCmd = &cobra.Command{
	Use:   "root",
	Short: "Show the root directory path for repository management",
	Long: `Show the root directory where baretree repositories are stored.

By default, shows the primary root directory (the last one configured).
Use --all to show all configured root directories.

Configuration:
  - Environment variable: BARETREE_ROOT
  - Git config: baretree.root (can be set multiple times)
  - Default: ~/baretree

Examples:
  bt repo root
  bt repo root --all`,
	Args: cobra.NoArgs,
	RunE: runRoot,
}

func init() {
	rootCmd.Flags().BoolVarP(&rootAll, "all", "a", false, "Show all root directories")
}

func runRoot(cmd *cobra.Command, args []string) error {
	cfg, err := global.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if rootAll {
		for _, root := range cfg.Roots {
			fmt.Println(root)
		}
	} else {
		fmt.Println(cfg.PrimaryRoot())
	}

	return nil
}
