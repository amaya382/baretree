package repo

import (
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage global baretree configuration (export, import)",
	Long: `Manage global baretree configuration.

Global configuration includes:
  - roots: Root directories where baretree repositories are stored

Export and import operations allow you to backup or share your global configuration.

Examples:
  bt repo config export                  # Output to stdout
  bt repo config export -o config.toml   # Write to file
  bt repo config import config.toml      # Import from file`,
}

func init() {
	configCmd.AddCommand(configExportCmd)
	configCmd.AddCommand(configImportCmd)
}
