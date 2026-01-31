package config

import (
	"bytes"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// ConfigFileName is used only for 'bt shared export/import' operations.
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
	if config.Repository.BareDir == "" {
		config.Repository.BareDir = ".bare"
	}
	if config.Repository.DefaultBranch == "" {
		config.Repository.DefaultBranch = "main"
	}

	return &config, nil
}

// ExportSharedToTOML exports only the shared configuration to TOML format
func ExportSharedToTOML(shared []Shared) (string, error) {
	// Create a wrapper struct for clean TOML output
	type sharedConfig struct {
		Shared []Shared `toml:"shared"`
	}
	cfg := sharedConfig{Shared: shared}

	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(cfg); err != nil {
		return "", fmt.Errorf("failed to encode shared config: %w", err)
	}
	return buf.String(), nil
}

// ImportSharedFromTOML imports shared configuration from TOML format
func ImportSharedFromTOML(data string) ([]Shared, error) {
	type sharedConfig struct {
		Shared []Shared `toml:"shared"`
	}
	var cfg sharedConfig
	if err := toml.Unmarshal([]byte(data), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}
	return cfg.Shared, nil
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
