package repo

import (
	"github.com/amaya382/baretree/internal/global"
	"github.com/spf13/cobra"
)

// completeRepositoryNames returns repository names for shell completion.
// If includePrevious is true, it also includes - (previous repository).
func completeRepositoryNames(includePrevious bool) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		completions := []string{}

		// Load global config
		cfg, err := global.LoadConfig()
		if err != nil {
			return completions, cobra.ShellCompDirectiveNoFileComp
		}

		// Get root directories
		roots := cfg.Roots
		if len(roots) == 0 {
			return completions, cobra.ShellCompDirectiveNoFileComp
		}

		// Scan for repositories
		repos, err := global.ScanRepositories(roots)
		if err != nil {
			return completions, cobra.ShellCompDirectiveNoFileComp
		}

		for _, repo := range repos {
			// Add relative path for completion
			completions = append(completions, repo.RelativePath)
		}

		// Add special completions
		if includePrevious {
			completions = append(completions, "-")
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}
