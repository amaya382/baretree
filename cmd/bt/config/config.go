package config

import (
	"github.com/spf13/cobra"
)

// Cmd is the parent command for configuration management
var Cmd = &cobra.Command{
	Use:   "config",
	Short: "Manage baretree configuration (export, import)",
	Long: `Manage baretree configuration including repository settings, worktree settings, and shared files.

Export and import operations allow you to backup, share, or transfer baretree configurations
between repositories.

Examples:
  bt config export                       # Output to stdout
  bt config export -o config.toml        # Write to file
  bt config import config.toml           # Import from file
  bt config import config.toml --merge   # Merge with existing`,
}

func init() {
	Cmd.AddCommand(exportCmd)
	Cmd.AddCommand(importCmd)
}
