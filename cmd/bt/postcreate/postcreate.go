package postcreate

import (
	"github.com/spf13/cobra"
)

// Cmd is the parent command for post-create action management
var Cmd = &cobra.Command{
	Use:   "post-create",
	Short: "Manage post-create actions (symlink, copy, command) for new worktrees",
	Long: `Manage actions that are performed after worktree creation.

Post-create actions can be one of three types:
  - symlink: Create a symlink to a shared file
  - copy: Copy a file to the new worktree
  - command: Execute a shell command in the new worktree

File-based actions (symlink/copy) can be managed in two modes:
  - Managed (default): Files are stored in .shared/ directory, independent of any worktree
  - Non-managed (--no-managed): Files are sourced from the default branch worktree

Examples:
  bt post-create add symlink .env
  bt post-create add symlink .env --managed
  bt post-create add copy config/local.json
  bt post-create add command "direnv allow"
  bt post-create add command "npm install"
  bt post-create remove .env
  bt post-create remove "direnv allow"
  bt post-create apply
  bt post-create list`,
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
