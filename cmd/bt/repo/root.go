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
	Short: "Show the root directory path for repository management (Default: ~/baretree)",
	Long: `Show the root directory where baretree repositories are stored (Default: ~/baretree).

By default, shows the primary root directory (the last one configured).
Use --all to show all configured root directories.

To change the root directory, use 'bt repo config root'.

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
