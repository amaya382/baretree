package repo

import (
	"fmt"
	"io"
	"os"

	"github.com/amaya382/baretree/internal/global"
	"github.com/spf13/cobra"
)

var (
	configImportMerge  bool
	configImportDryRun bool
)

var configImportCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import global baretree configuration from TOML format",
	Long: `Import global baretree configuration from TOML format.

This imports global settings including:
  - roots: Root directories where repositories are stored

Reads from a file or stdin if no file is specified.

By default, replaces the existing configuration.
Use --merge to add roots without removing existing ones.

Examples:
  bt repo config import config.toml           # Import from file
  cat config.toml | bt repo config import     # Import from stdin
  bt repo config import config.toml --merge   # Merge with existing`,
	Args: cobra.MaximumNArgs(1),
	RunE: runConfigImport,
}

func init() {
	configImportCmd.Flags().BoolVar(&configImportMerge, "merge", false, "Merge with existing configuration instead of replacing")
	configImportCmd.Flags().BoolVar(&configImportDryRun, "dry-run", false, "Show what would be imported without making changes")
}

func runConfigImport(cmd *cobra.Command, args []string) error {
	// Read input
	var data []byte
	var err error
	if len(args) == 1 {
		data, err = os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
	} else {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			return fmt.Errorf("no input file specified and stdin is empty")
		}
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}
	}

	// Parse TOML
	importedCfg, err := global.ImportConfigFromTOML(string(data))
	if err != nil {
		return fmt.Errorf("failed to parse TOML: %w", err)
	}

	// Load current config
	currentCfg, err := global.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load current config: %w", err)
	}

	// Show what will be imported
	fmt.Println("Importing global configuration:")
	fmt.Println()
	fmt.Printf("roots (%d entries):\n", len(importedCfg.Roots))
	for _, root := range importedCfg.Roots {
		fmt.Printf("  %s\n", root)
	}
	fmt.Println()

	if configImportDryRun {
		fmt.Println("Dry run - no changes made")
		return nil
	}

	// Determine final roots
	var finalRoots []string
	if configImportMerge {
		// Merge: add new roots, skip existing
		existingMap := make(map[string]bool)
		for _, root := range currentCfg.Roots {
			existingMap[root] = true
		}
		finalRoots = currentCfg.Roots
		added := 0
		for _, root := range importedCfg.Roots {
			if !existingMap[root] {
				finalRoots = append(finalRoots, root)
				added++
			}
		}
		fmt.Printf("Added %d new root(s), %d already existed\n", added, len(importedCfg.Roots)-added)
	} else {
		// Replace
		finalRoots = importedCfg.Roots
		fmt.Printf("Replaced roots with %d entry(ies)\n", len(finalRoots))
	}

	// Save to git-config
	if err := global.SaveRootsToGitConfig(finalRoots); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	fmt.Println("Import completed successfully")

	return nil
}
