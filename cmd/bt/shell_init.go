package main

import (
	"fmt"

	"github.com/amaya382/baretree/internal/shell"
	"github.com/spf13/cobra"
)

var shellInitCmd = &cobra.Command{
	Use:   "shell-init <shell>",
	Short: "Generate shell integration code",
	Long: `Generate shell integration code for the specified shell.

This sets up:
  - Shell function to intercept 'bt cd' command
  - Tab completion for bt commands

Supported shells: bash, zsh, fish

Setup:
  For bash, add to ~/.bashrc:
    eval "$(bt shell-init bash)"

  For zsh, add to ~/.zshrc:
    eval "$(bt shell-init zsh)"

  For fish, add to ~/.config/fish/config.fish:
    bt shell-init fish | source

Examples:
  bt shell-init bash
  bt shell-init zsh
  bt shell-init fish`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish"},
	RunE:      runShellInit,
}

func runShellInit(cmd *cobra.Command, args []string) error {
	shellType := args[0]

	var script string

	switch shellType {
	case "bash":
		script = shell.BashScript
	case "zsh":
		script = shell.ZshScript
	case "fish":
		script = shell.FishScript
	default:
		return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish)", shellType)
	}

	fmt.Print(script)

	return nil
}
