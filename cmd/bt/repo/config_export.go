package repo

import (
	"fmt"
	"os"

	"github.com/amaya382/baretree/internal/global"
	"github.com/spf13/cobra"
)

var (
	configExportFile string
)

var configExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export global baretree configuration to TOML format",
	Long: `Export the global baretree configuration to TOML format.

This exports global settings including:
  - roots: Root directories where repositories are stored

By default, outputs to stdout. Use -o to write to a file.

Examples:
  bt repo config export                  # Output to stdout
  bt repo config export -o config.toml   # Write to file`,
	RunE: runConfigExport,
}

func init() {
	configExportCmd.Flags().StringVarP(&configExportFile, "output", "o", "", "Output file (default: stdout)")
}

func runConfigExport(cmd *cobra.Command, args []string) error {
	cfg, err := global.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	tomlContent, err := global.ExportConfigToTOML(cfg)
	if err != nil {
		return fmt.Errorf("failed to export config: %w", err)
	}

	if configExportFile != "" {
		if err := os.WriteFile(configExportFile, []byte(tomlContent), 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("Exported global configuration to %s\n", configExportFile)
	} else {
		fmt.Print(tomlContent)
	}

	return nil
}
