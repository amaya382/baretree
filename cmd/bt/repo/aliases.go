package repo

import (
	"github.com/spf13/cobra"
)

// Top-level alias commands for quick access
// These are registered in main.go as root-level commands

// InitAliasCmd is a top-level alias for "bt repo init"
var InitAliasCmd = &cobra.Command{
	Use:   "init [directory]",
	Short: "Create a new baretree repository (alias for 'bt repo init')",
	Long: `Create a new Git repository with baretree structure.

This is an alias for 'bt repo init'. See 'bt repo init --help' for details.

Examples:
  bt init                      # Initialize in current directory
  bt init my-project           # Initialize in my-project directory
  bt init . --branch develop   # Use 'develop' as default branch`,
	Args:               cobra.MaximumNArgs(1),
	RunE:               runInit,
	DisableFlagParsing: false,
}

// CloneAliasCmd is a top-level alias for "bt repo clone"
var CloneAliasCmd = &cobra.Command{
	Use:   "clone <repository-url> [destination]",
	Short: "Clone a remote repository with baretree structure (alias for 'bt repo clone')",
	Long: `Clone a Git repository as a bare repository and set up baretree structure.

This is an alias for 'bt repo clone'. See 'bt repo clone --help' for details.

Examples:
  bt clone git@github.com:user/repo.git my-project
  bt clone https://github.com/user/repo.git
  bt clone git@github.com:user/repo.git --branch develop`,
	Args:               cobra.RangeArgs(1, 2),
	RunE:               runClone,
	DisableFlagParsing: false,
}

// MigrateAliasCmd is a top-level alias for "bt repo migrate"
var MigrateAliasCmd = &cobra.Command{
	Use:   "migrate <existing-repo-path>",
	Short: "Convert existing Git repository to baretree structure (alias for 'bt repo migrate')",
	Long: `Convert an existing Git repository to baretree structure.

This is an alias for 'bt repo migrate'. See 'bt repo migrate --help' for details.

Examples:
  bt migrate /path/to/existing-repo --in-place
  bt migrate . -i
  bt migrate ~/projects/myapp --destination ../my-project-baretree`,
	Args:               cobra.ExactArgs(1),
	RunE:               runMigrate,
	DisableFlagParsing: false,
}

// GetAliasCmd is a top-level alias for "bt repo get"
var GetAliasCmd = &cobra.Command{
	Use:   "get <repository>",
	Short: "Clone a repository into root directory (alias for 'bt repo get')",
	Long: `Clone a repository into the baretree root directory with ghq-style path structure.

This is an alias for 'bt repo get'. See 'bt repo get --help' for details.

Examples:
  bt get github.com/amaya382/baretree
  bt get git@github.com:amaya382/baretree.git
  bt get amaya382/dotfiles`,
	Args:               cobra.ExactArgs(1),
	RunE:               runGet,
	DisableFlagParsing: false,
}

// GoAliasCmd is a top-level alias for "bt repo cd"
var GoAliasCmd = &cobra.Command{
	Use:   "go <repository>",
	Short: "Change to a repository directory (alias for 'bt repo cd')",
	Long: `Output the absolute path to a repository for use with shell integration.

This is an alias for 'bt repo cd'. See 'bt repo cd --help' for details.

Examples:
  bt go baretree                    # Match by repo name
  bt go amaya382/baretree           # Match by org/repo
  bt go github.com/amaya382/baretree # Match by full path
  bt go -                           # Go to previous repository`,
	Args: cobra.ExactArgs(1),
	RunE: runRepoCd,
}

// ReposAliasCmd is a top-level alias for "bt repo list"
var ReposAliasCmd = &cobra.Command{
	Use:   "repos [query]",
	Short: "List all baretree repositories (alias for 'bt repo list')",
	Long: `List all baretree repositories under the configured root directory.

This is an alias for 'bt repo list'. See 'bt repo list --help' for details.

Examples:
  bt repos
  bt repos --paths
  bt repos github.com
  bt repos --json`,
	Args:               cobra.MaximumNArgs(1),
	RunE:               runList,
	DisableFlagParsing: false,
}

func init() {
	// Copy flags from original commands to aliases
	InitAliasCmd.Flags().StringVarP(&initDefaultBranch, "branch", "b", "main", "Default branch name")

	CloneAliasCmd.Flags().StringVarP(&cloneBranch, "branch", "b", "", "Checkout specific branch instead of default")

	MigrateAliasCmd.Flags().BoolVarP(&migrateInPlace, "in-place", "i", false, "Replace the original repository in-place (recommended)")
	MigrateAliasCmd.Flags().StringVarP(&migrateDestination, "destination", "d", "", "Destination directory for the new baretree structure")
	MigrateAliasCmd.Flags().BoolVarP(&migrateToRoot, "to-root", "r", false, "Move repository to baretree root with ghq-style path")
	MigrateAliasCmd.Flags().StringVar(&migrateRepoPath, "path", "", "Repository path for --to-root (e.g., github.com/user/repo)")

	GetAliasCmd.Flags().StringVarP(&getBranch, "branch", "b", "", "Checkout specific branch")
	GetAliasCmd.Flags().BoolVar(&getShallow, "shallow", false, "Perform a shallow clone")
	GetAliasCmd.Flags().BoolVarP(&getUpdate, "update", "u", false, "Update existing repository")

	ReposAliasCmd.Flags().BoolVarP(&listPaths, "paths", "p", false, "Show full paths")
	ReposAliasCmd.Flags().BoolVarP(&listJSON, "json", "j", false, "Output as JSON")
}
