package config

import (
	"fmt"
	"io"
	"os"

	"github.com/amaya382/baretree/internal/config"
	"github.com/amaya382/baretree/internal/repository"
	"github.com/amaya382/baretree/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	importMerge  bool
	importApply  bool
	importDryRun bool
)

var importCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import baretree configuration from TOML format",
	Long: `Import baretree configuration from TOML format.

This imports all baretree-related settings including:
  - Repository settings (bare directory, default branch)
  - Shared files configuration

Reads from a file or stdin if no file is specified.

By default, replaces the existing configuration.
Use --merge to add shared entries without removing existing ones
(repository and worktree settings are always updated).
Use --apply to immediately apply shared file changes to all worktrees.

Examples:
  bt config import config.toml           # Import from file
  cat config.toml | bt config import     # Import from stdin
  bt config import config.toml --merge   # Merge shared files with existing
  bt config import config.toml --apply   # Import and apply shared files`,
	Args: cobra.MaximumNArgs(1),
	RunE: runImport,
}

func init() {
	importCmd.Flags().BoolVar(&importMerge, "merge", false, "Merge shared files with existing configuration instead of replacing")
	importCmd.Flags().BoolVar(&importApply, "apply", false, "Apply shared file changes to all worktrees after import")
	importCmd.Flags().BoolVar(&importDryRun, "dry-run", false, "Show what would be imported without making changes")
}

func runImport(cmd *cobra.Command, args []string) error {
	// Find repository root
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("not in a baretree repository: %w", err)
	}

	// Read input
	var data []byte
	if len(args) == 1 {
		data, err = os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
	} else {
		// Check if stdin has data
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
	importedCfg, err := config.ImportConfigFromTOML(string(data))
	if err != nil {
		return fmt.Errorf("failed to parse TOML: %w", err)
	}

	// Load current config
	currentCfg, err := config.LoadConfig(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Show what will be imported
	fmt.Println("Importing configuration:")
	fmt.Println()
	fmt.Println("[repository]")
	fmt.Printf("  default_branch: %s\n", importedCfg.Repository.DefaultBranch)
	fmt.Println()
	fmt.Printf("[shared] (%d entries)\n", len(importedCfg.Shared))
	for _, s := range importedCfg.Shared {
		modeStr := ""
		if s.Managed {
			modeStr = " (managed)"
		}
		fmt.Printf("  %s [%s]%s\n", s.Source, s.Type, modeStr)
	}
	fmt.Println()

	if importDryRun {
		fmt.Println("Dry run - no changes made")
		return nil
	}

	// Update configuration
	// Repository and worktree settings are always updated
	currentCfg.Repository = importedCfg.Repository

	// Handle shared files based on merge flag
	if importMerge {
		// Merge: add new entries, skip existing
		existingMap := make(map[string]bool)
		for _, s := range currentCfg.Shared {
			existingMap[s.Source] = true
		}

		added := 0
		for _, s := range importedCfg.Shared {
			if !existingMap[s.Source] {
				currentCfg.Shared = append(currentCfg.Shared, s)
				added++
			}
		}
		fmt.Printf("Repository and worktree settings updated\n")
		fmt.Printf("Shared files: added %d new entry(ies), %d already existed\n", added, len(importedCfg.Shared)-added)
	} else {
		// Replace
		currentCfg.Shared = importedCfg.Shared
		fmt.Printf("Replaced entire configuration\n")
	}

	// Save configuration
	if err := config.SaveConfig(repoRoot, currentCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Apply if requested
	if importApply && len(currentCfg.Shared) > 0 {
		bareDir, err := repository.GetBareRepoPath(repoRoot)
		if err != nil {
			return fmt.Errorf("failed to get bare repo path: %w", err)
		}

		wtMgr := worktree.NewManager(repoRoot, bareDir, currentCfg)
		results, err := wtMgr.ApplyAllShared()
		if err != nil {
			return fmt.Errorf("failed to apply shared files: %w", err)
		}

		fmt.Println()
		fmt.Println("Applied shared files:")
		for _, result := range results {
			fmt.Printf("  %s: applied to %d worktree(s)\n", result.Source, len(result.Applied))
		}
	}

	fmt.Println()
	fmt.Println("Import completed successfully")
	if !importApply && len(currentCfg.Shared) > 0 {
		fmt.Println("Run 'bt shared apply' to apply shared file changes to all worktrees")
	}

	return nil
}
