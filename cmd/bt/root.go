package main

import (
	"fmt"
	"os"

	"github.com/amaya382/baretree/internal/repository"
	"github.com/spf13/cobra"
)

var showRootCmd = &cobra.Command{
	Use:   "root",
	Short: "Show the repository root directory path",
	Long: `Show the root directory path of the current baretree repository.

The root directory is the directory containing the bare .git directory.

Examples:
  bt root
  cd $(bt root)`,
	RunE: runRoot,
}

func runRoot(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("not in a baretree repository: %w", err)
	}

	fmt.Println(repoRoot)
	return nil
}
