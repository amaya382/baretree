package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/amaya382/baretree/cmd/bt/config"
	"github.com/amaya382/baretree/cmd/bt/repo"
	"github.com/amaya382/baretree/cmd/bt/shared"
	"github.com/spf13/cobra"
)

// NameWithAlias returns command name with aliases, e.g. "list (ls)"
func NameWithAlias(cmd *cobra.Command) string {
	if len(cmd.Aliases) == 0 {
		return cmd.Name()
	}
	return cmd.Name() + " (" + strings.Join(cmd.Aliases, ", ") + ")"
}

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Command group IDs
const (
	groupWorktree  = "worktree"
	groupRepo      = "repo"
	groupRepoAlias = "repo-alias"
	groupMisc      = "misc"
)

var rootCmd = &cobra.Command{
	Use:     "bt",
	Short:   "baretree - Powerful git worktree manager built on bare repositories with multi-repo support",
	Version: version,
}

var versionCmd = &cobra.Command{
	Use:    "version",
	Short:  "Print version information",
	Hidden: true, // Use --version flag instead; this is kept for backward compatibility
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()
	},
}

func printVersion() {
	fmt.Printf("baretree %s\n", version)
	fmt.Printf("  commit: %s\n", commit)
	fmt.Printf("  built:  %s\n", date)
}

func init() {
	// Set version template to match the version subcommand output
	rootCmd.SetVersionTemplate(fmt.Sprintf("baretree %s\n  commit: %s\n  built:  %s\n", version, commit, date))

	// Disable the auto-generated completion command (use shell-init instead)
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Add custom template function for displaying command name with aliases
	cobra.AddTemplateFunc("nameWithAlias", NameWithAlias)

	// Custom help template without "Usage:" section for root command
	// Use nameWithAlias function to display "list (ls)" format
	// Miscellaneous commands and flags are combined into one section
	rootCmd.SetHelpTemplate(`{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}
{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}Available Commands:{{range $cmds}}{{if .IsAvailableCommand}}
  {{rpad (nameWithAlias .) .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}{{if ne $group.ID "misc"}}
{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) .IsAvailableCommand)}}
  {{rpad (nameWithAlias .) .NamePadding }} {{.Short}}{{end}}{{end}}
{{end}}{{end}}{{end}}{{end}}
Miscellaneous and Flags:{{$cmds := .Commands}}{{range $cmds}}{{if (and (eq .GroupID "misc") .IsAvailableCommand)}}
  {{rpad (nameWithAlias .) 16}}{{.Short}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`)

	// Define command groups
	rootCmd.AddGroup(
		&cobra.Group{ID: groupWorktree, Title: "Worktree Management:"},
		&cobra.Group{ID: groupRepo, Title: "Repository Management:"},
		&cobra.Group{ID: groupRepoAlias, Title: "  [Top-level Aliases]"},
		&cobra.Group{ID: groupMisc, Title: "Miscellaneous:"},
	)

	// Worktree management commands (operations within a repository)
	addCmd.GroupID = groupWorktree
	listCmd.GroupID = groupWorktree
	removeCmd.GroupID = groupWorktree
	cdCmd.GroupID = groupWorktree
	statusCmd.GroupID = groupWorktree
	renameCmd.GroupID = groupWorktree
	repairCmd.GroupID = groupWorktree
	showRootCmd.GroupID = groupWorktree
	shared.Cmd.GroupID = groupWorktree
	unbareCmd.GroupID = groupWorktree
	config.Cmd.GroupID = groupWorktree

	// Repository management commands (init, clone, migrate + ghq-like operations)
	repo.Cmd.GroupID = groupRepo

	// Miscellaneous commands
	shellInitCmd.GroupID = groupMisc

	// Hide the help subcommand (use -h/--help flag instead)
	// The help command is still available but won't appear in the command list
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "help [command]",
		Short:  "Help about any command",
		Hidden: true,
		Run: func(c *cobra.Command, args []string) {
			cmd, _, e := rootCmd.Find(args)
			if cmd == nil || e != nil {
				c.Printf("Unknown help topic %#q\n", args)
				_ = rootCmd.Usage()
			} else {
				cmd.InitDefaultHelpFlag()
				_ = cmd.Help()
			}
		},
	})

	// Add all subcommands
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(cdCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(repairCmd)
	rootCmd.AddCommand(shellInitCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(repo.Cmd)
	rootCmd.AddCommand(shared.Cmd)
	rootCmd.AddCommand(renameCmd)
	rootCmd.AddCommand(unbareCmd)
	rootCmd.AddCommand(config.Cmd)
	rootCmd.AddCommand(showRootCmd)

	// Top-level aliases for repo commands
	repo.InitAliasCmd.GroupID = groupRepoAlias
	repo.CloneAliasCmd.GroupID = groupRepoAlias
	repo.MigrateAliasCmd.GroupID = groupRepoAlias
	repo.GetAliasCmd.GroupID = groupRepoAlias
	repo.GoAliasCmd.GroupID = groupRepoAlias
	repo.ReposAliasCmd.GroupID = groupRepoAlias
	rootCmd.AddCommand(repo.InitAliasCmd)
	rootCmd.AddCommand(repo.CloneAliasCmd)
	rootCmd.AddCommand(repo.MigrateAliasCmd)
	rootCmd.AddCommand(repo.GetAliasCmd)
	rootCmd.AddCommand(repo.GoAliasCmd)
	rootCmd.AddCommand(repo.ReposAliasCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
