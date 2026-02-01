package config

import (
	"fmt"
	"os"

	"github.com/amaya382/baretree/internal/config"
	"github.com/amaya382/baretree/internal/repository"
	"github.com/spf13/cobra"
)

var (
	exportFile string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export baretree configuration to TOML format",
	Long: `Export the current baretree configuration to TOML format.

This exports all baretree-related settings including:
  - Repository settings (default branch)
  - Post-create actions (symlink, copy, command)

By default, outputs to stdout. Use -o to write to a file.

Examples:
  bt config export                       # Output to stdout
  bt config export -o config.toml        # Write to file
  bt config export -o baretree.toml      # Write to baretree.toml`,
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVarP(&exportFile, "output", "o", "", "Output file (default: stdout)")
}

func runExport(cmd *cobra.Command, args []string) error {
	// Find repository root
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("not in a baretree repository: %w", err)
	}

	// Load config
	cfg, err := config.LoadConfig(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Export to TOML
	tomlContent, err := config.ExportConfigToTOML(cfg)
	if err != nil {
		return fmt.Errorf("failed to export config: %w", err)
	}

	// Write output
	if exportFile != "" {
		if err := os.WriteFile(exportFile, []byte(tomlContent), 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("Exported baretree configuration to %s\n", exportFile)
	} else {
		fmt.Print(tomlContent)
	}

	return nil
}
