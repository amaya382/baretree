package synctoroot

import (
	"github.com/spf13/cobra"
)

// Cmd is the parent command for sync-to-root action management
var Cmd = &cobra.Command{
	Use:   "sync-to-root",
	Short: "Manage symlinks from default branch worktree to repository root",
	Long: `Manage symlinks that sync files from the default branch worktree to the repository root.

This feature allows files in the default branch worktree (e.g., main/CLAUDE.md)
to be accessible from the repository root (e.g., CLAUDE.md) via symlinks.

Use case: When running tools like Claude Code from the repository root,
files like CLAUDE.md can be recognized without navigating into the worktree.

Examples:
  bt sync-to-root add CLAUDE.md
  bt sync-to-root add .claude
  bt sync-to-root add docs/guide.md guide.md
  bt sync-to-root remove CLAUDE.md
  bt sync-to-root apply
  bt sync-to-root list`,
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
