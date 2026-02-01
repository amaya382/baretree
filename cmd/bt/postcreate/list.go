package postcreate

import (
	"fmt"

	"github.com/amaya382/baretree/internal/repository"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List post-create action configurations",
	Long: `List all post-create action configurations.

Examples:
  bt post-create list
  bt post-create ls`,
	RunE: runPostCreateList,
}

func runPostCreateList(cmd *cobra.Command, args []string) error {
	// Find repository root
	cwd, err := cmd.Flags().GetString("cwd")
	if err != nil || cwd == "" {
		cwd = "."
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("not in a baretree repository: %w", err)
	}

	// Load config
	repoMgr, err := repository.NewManager(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	cfg := repoMgr.Config

	if len(cfg.PostCreate) == 0 {
		fmt.Println("No post-create actions configured.")
		return nil
	}

	// Get default branch
	defaultBranch := cfg.Repository.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	fmt.Println("Post-create actions:")

	// Calculate column widths
	maxSourceLen := 6 // "Source"
	for _, action := range cfg.PostCreate {
		if len(action.Source) > maxSourceLen {
			maxSourceLen = len(action.Source)
		}
	}

	for _, action := range cfg.PostCreate {
		var modeStr string
		switch action.Type {
		case "command":
			modeStr = ""
		case "symlink", "copy":
			if action.Managed {
				modeStr = "managed"
			} else {
				modeStr = fmt.Sprintf("(from %s)", defaultBranch)
			}
		}

		fmt.Printf("  [%-7s] %-*s  %s\n",
			action.Type,
			maxSourceLen, action.Source,
			modeStr,
		)
	}

	return nil
}
