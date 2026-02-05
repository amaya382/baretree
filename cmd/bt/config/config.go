package config

import (
	"github.com/spf13/cobra"
)

// Cmd is the parent command for configuration management
var Cmd = &cobra.Command{
	Use:   "config",
	Short: "Manage baretree configuration",
	Long: `Manage baretree configuration including repository settings and post-create actions.

Subcommands:
  default-branch    Get or set the default branch
  export            Export configuration to TOML format
  import            Import configuration from TOML format

Examples:
  bt config default-branch               # Show current default branch
  bt config default-branch main          # Set default branch to 'main'
  bt config default-branch --unset       # Remove setting (reverts to 'main')
  bt config export                       # Output to stdout
  bt config export -o config.toml        # Write to file
  bt config import config.toml           # Import from file
  bt config import config.toml --merge   # Merge with existing`,
}

func init() {
	Cmd.AddCommand(exportCmd)
	Cmd.AddCommand(importCmd)
}
