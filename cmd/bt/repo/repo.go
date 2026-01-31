package repo

import (
	"github.com/spf13/cobra"
)

// Command group IDs for repo subcommands
const (
	groupRepoMgmt = "repo-mgmt"
	groupCross    = "cross"
)

// Cmd is the parent command for repository management
var Cmd = &cobra.Command{
	Use:   "repo",
	Short: "Repository setup and cross-repository management commands (get, init, remove, ...)",
	Long:  `Repository setup and cross-repository management commands`,
}

func init() {
	// Custom help template with alias information (uses nameWithAlias registered in main.go)
	Cmd.SetHelpTemplate(`{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}
{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}
Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad (nameWithAlias .) .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}
{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad (nameWithAlias .) .NamePadding }} {{.Short}}{{end}}{{end}}
{{end}}{{if not .AllChildCommandsHaveGroup}}
Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad (nameWithAlias .) .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}
Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`)

	// Define command groups for repo subcommands
	Cmd.AddGroup(
		&cobra.Group{ID: groupRepoMgmt, Title: "Repository Management [top-level aliases available]:"},
		&cobra.Group{ID: groupCross, Title: "Cross-Repository Management [top-level aliases available]:"},
	)

	// Repository Management commands
	initCmd.GroupID = groupRepoMgmt
	cloneCmd.GroupID = groupRepoMgmt
	migrateCmd.GroupID = groupRepoMgmt

	// Cross-Repository Management commands
	listCmd.GroupID = groupCross
	rootCmd.GroupID = groupCross
	getCmd.GroupID = groupCross
	configCmd.GroupID = groupCross
	// Note: cdCmd and removeCmd are added in their respective files with GroupID set

	Cmd.AddCommand(initCmd)
	Cmd.AddCommand(cloneCmd)
	Cmd.AddCommand(migrateCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(rootCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(configCmd)
}
