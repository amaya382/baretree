package repo

import (
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage global baretree configuration",
	Long: `Manage global baretree configuration.

Global configuration includes:
  - root: Root directory where baretree repositories are stored

Subcommands:
  root      Get or set the root directory
  export    Export configuration to TOML format
  import    Import configuration from TOML format

Examples:
  bt repo config root                    # Show current root directory
  bt repo config root ~/code             # Set root to ~/code
  bt repo config root --unset            # Remove setting (reverts to ~/baretree)
  bt repo config export                  # Output to stdout
  bt repo config export -o config.toml   # Write to file
  bt repo config import config.toml      # Import from file`,
}

func init() {
	configCmd.AddCommand(configExportCmd)
	configCmd.AddCommand(configImportCmd)
}
