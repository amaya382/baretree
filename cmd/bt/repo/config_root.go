package repo

import (
	"fmt"
	"os"
	"slices"

	"github.com/amaya382/baretree/internal/global"
	"github.com/spf13/cobra"
)

var (
	configRootUnset bool
	configRootAdd   bool
	configRootAll   bool
)

var configRootCmd = &cobra.Command{
	Use:   "root [path]",
	Short: "Get or set the baretree root directory",
	Long: `Get or set the root directory where baretree repositories are stored.

Without arguments, displays the current root directory.
With --all flag, displays all configured root directories.
With a path argument, sets the root directory (replaces existing).
With --add flag, appends the path to existing roots.
With --unset flag, removes the setting (reverts to default '~/baretree').

Note: If BARETREE_ROOT environment variable is set, it takes precedence over
the git-config setting. Use 'unset BARETREE_ROOT' to clear the environment variable.

Examples:
  bt repo config root                    # Show current root directory
  bt repo config root --all              # Show all root directories
  bt repo config root ~/code             # Set root to ~/code (replaces existing)
  bt repo config root --add ~/ghq        # Add ~/ghq to roots
  bt repo config root --unset            # Remove setting (reverts to ~/baretree)`,
	Args: cobra.MaximumNArgs(1),
	RunE: runConfigRoot,
}

func init() {
	configRootCmd.Flags().BoolVar(&configRootUnset, "unset", false, "Remove the root setting (reverts to default '~/baretree')")
	configRootCmd.Flags().BoolVar(&configRootAdd, "add", false, "Add path to existing roots instead of replacing")
	configRootCmd.Flags().BoolVarP(&configRootAll, "all", "a", false, "Show all root directories")
	configCmd.AddCommand(configRootCmd)
}

func runConfigRoot(cmd *cobra.Command, args []string) error {
	// Check if BARETREE_ROOT is set
	envRoot := os.Getenv("BARETREE_ROOT")

	// Validate flag combinations
	if configRootUnset && configRootAdd {
		return fmt.Errorf("cannot use --unset and --add together")
	}
	if configRootAll && (configRootUnset || configRootAdd || len(args) > 0) {
		return fmt.Errorf("--all cannot be used with other flags or arguments")
	}

	// Unset mode
	if configRootUnset {
		if len(args) > 0 {
			return fmt.Errorf("cannot specify path with --unset flag")
		}

		if envRoot != "" {
			fmt.Println("Warning: BARETREE_ROOT environment variable is set.")
			fmt.Println("The environment variable takes precedence over git-config.")
			fmt.Println("Use 'unset BARETREE_ROOT' to clear it.")
			fmt.Println()
		}

		if err := global.UnsetRoot(); err != nil {
			return fmt.Errorf("failed to unset root: %w", err)
		}
		fmt.Println("Root setting removed (will use default '~/baretree')")
		return nil
	}

	// Add mode requires a path argument
	if configRootAdd && len(args) == 0 {
		return fmt.Errorf("--add requires a path argument")
	}

	// Load current config
	cfg, err := global.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get mode: display current root(s)
	if len(args) == 0 {
		if configRootAll {
			for _, root := range cfg.Roots {
				fmt.Println(root)
			}
		} else {
			fmt.Println(cfg.PrimaryRoot())
		}
		return nil
	}

	// Set mode
	newRoot := args[0]
	currentRoot := cfg.PrimaryRoot()

	// Warn if environment variable is set
	if envRoot != "" {
		fmt.Println("Warning: BARETREE_ROOT environment variable is set.")
		fmt.Println("The environment variable takes precedence over git-config.")
		fmt.Println("Use 'unset BARETREE_ROOT' to use the git-config setting.")
		fmt.Println()
	}

	// Add mode: append to existing roots
	if configRootAdd {
		// Check if root already exists
		expandedNew := global.ExpandTilde(newRoot)
		if slices.Contains(cfg.Roots, expandedNew) {
			fmt.Printf("Root '%s' already exists\n", newRoot)
			return nil
		}

		if err := global.AddRoot(newRoot); err != nil {
			return fmt.Errorf("failed to add root: %w", err)
		}

		fmt.Printf("Root '%s' added\n", newRoot)
		return nil
	}

	// Replace mode (default): check if root is actually changing
	expandedNew := global.ExpandTilde(newRoot)
	if expandedNew == currentRoot && global.GetRootSource() == "git-config" {
		fmt.Printf("Root is already '%s'\n", newRoot)
		return nil
	}

	if err := global.SetRoot(newRoot); err != nil {
		return fmt.Errorf("failed to set root: %w", err)
	}

	fmt.Printf("Root set to '%s'\n", newRoot)
	return nil
}
