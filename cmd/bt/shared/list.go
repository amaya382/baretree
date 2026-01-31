package shared

import (
	"fmt"

	"github.com/amaya382/baretree/internal/repository"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List shared file configurations",
	Long: `List all shared file and directory configurations.

Examples:
  bt shared list
  bt shared ls`,
	RunE: runSharedList,
}

func runSharedList(cmd *cobra.Command, args []string) error {
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

	if len(cfg.Shared) == 0 {
		fmt.Println("No shared files configured.")
		return nil
	}

	// Get default branch
	defaultBranch := cfg.Repository.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	fmt.Println("Shared files:")

	// Calculate column widths
	maxSourceLen := 6 // "Source"
	for _, shared := range cfg.Shared {
		if len(shared.Source) > maxSourceLen {
			maxSourceLen = len(shared.Source)
		}
	}

	for _, shared := range cfg.Shared {
		modeStr := fmt.Sprintf("(from %s)", defaultBranch)
		if shared.Managed {
			modeStr = "managed"
		}

		fmt.Printf("  %-*s  %-7s  %s\n",
			maxSourceLen, shared.Source,
			shared.Type,
			modeStr,
		)
	}

	return nil
}
