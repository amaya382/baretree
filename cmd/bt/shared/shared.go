package shared

import (
	"github.com/spf13/cobra"
)

// Cmd is the parent command for shared file management
var Cmd = &cobra.Command{
	Use:   "shared",
	Short: "Manage files/directories shared across worktrees (add, remove, apply, ...)",
	Long: `Manage files and directories that are shared (symlinked or copied) across all worktrees.

Shared files can be managed in two modes:
  - Default: Files are sourced from the default branch worktree
  - Managed (--managed): Files are stored in .shared/ directory, independent of any worktree

Examples:
  bt shared add .env --type symlink --managed
  bt shared add config/local.json --type copy
  bt shared remove .env
  bt shared apply
  bt shared list`,
}

func init() {
	// Custom help template with alias information (uses nameWithAlias registered in main.go)
	Cmd.SetHelpTemplate(`{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}
{{end}}{{if .HasAvailableSubCommands}}
Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad (nameWithAlias .) .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}
Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`)

	Cmd.AddCommand(addCmd)
	Cmd.AddCommand(removeCmd)
	Cmd.AddCommand(applyCmd)
	Cmd.AddCommand(listCmd)
}
