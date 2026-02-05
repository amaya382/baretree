package config

import (
	"bytes"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// ConfigFileName is used only for 'bt config export/import' operations.
// Runtime configuration is stored in git-config, not in this file.
const ConfigFileName = "baretree.toml"

// LoadConfig loads configuration from git-config in the bare repository
func LoadConfig(repoRoot string) (*Config, error) {
	return LoadConfigFromGit(repoRoot)
}

// SaveConfig saves configuration to git-config in the bare repository
func SaveConfig(repoRoot string, config *Config) error {
	return SaveConfigToGit(repoRoot, config)
}

// FindRepoRoot finds the repository root by looking for a bare repo with baretree config
func FindRepoRoot(startPath string) (string, error) {
	return FindRepoRootGit(startPath)
}

// ExportConfigToTOML exports the configuration to TOML format
func ExportConfigToTOML(config *Config) (string, error) {
	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(config); err != nil {
		return "", fmt.Errorf("failed to encode config: %w", err)
	}
	return buf.String(), nil
}

// ImportConfigFromTOML imports configuration from TOML format
func ImportConfigFromTOML(data string) (*Config, error) {
	var config Config
	if err := toml.Unmarshal([]byte(data), &config); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	// Set defaults if not specified
	if config.Repository.DefaultBranch == "" {
		config.Repository.DefaultBranch = "main"
	}

	// Ensure slices are not nil
	if config.SyncToRoot == nil {
		config.SyncToRoot = []SyncToRootAction{}
	}

	return &config, nil
}

// ExportPostCreateToTOML exports only the post-create configuration to TOML format
func ExportPostCreateToTOML(actions []PostCreateAction) (string, error) {
	// Create a wrapper struct for clean TOML output
	type postCreateConfig struct {
		PostCreate []PostCreateAction `toml:"postcreate"`
	}
	cfg := postCreateConfig{PostCreate: actions}

	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(cfg); err != nil {
		return "", fmt.Errorf("failed to encode post-create config: %w", err)
	}
	return buf.String(), nil
}

// ImportPostCreateFromTOML imports post-create configuration from TOML format
func ImportPostCreateFromTOML(data string) ([]PostCreateAction, error) {
	type postCreateConfig struct {
		PostCreate []PostCreateAction `toml:"postcreate"`
	}
	var cfg postCreateConfig
	if err := toml.Unmarshal([]byte(data), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}
	return cfg.PostCreate, nil
}

// ExportSyncToRootToTOML exports only the sync-to-root configuration to TOML format
func ExportSyncToRootToTOML(actions []SyncToRootAction) (string, error) {
	type syncToRootConfig struct {
		SyncToRoot []SyncToRootAction `toml:"synctoroot"`
	}
	cfg := syncToRootConfig{SyncToRoot: actions}

	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(cfg); err != nil {
		return "", fmt.Errorf("failed to encode sync-to-root config: %w", err)
	}
	return buf.String(), nil
}

// ImportSyncToRootFromTOML imports sync-to-root configuration from TOML format
func ImportSyncToRootFromTOML(data string) ([]SyncToRootAction, error) {
	type syncToRootConfig struct {
		SyncToRoot []SyncToRootAction `toml:"synctoroot"`
	}
	var cfg syncToRootConfig
	if err := toml.Unmarshal([]byte(data), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}
	return cfg.SyncToRoot, nil
}

// SaveConfigToTOMLFile saves the configuration to a TOML file (for export)
func SaveConfigToTOMLFile(filePath string, config *Config) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}
	return nil
}

// LoadConfigFromTOMLFile loads configuration from a TOML file (for import)
func LoadConfigFromTOMLFile(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return ImportConfigFromTOML(string(data))
}
