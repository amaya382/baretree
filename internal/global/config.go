package global

import (
	"bytes"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/amaya382/baretree/internal/git"
)

const (
	// DefaultRoot is the default root directory for repositories
	DefaultRoot = "~/baretree"
)

// Config holds the global baretree configuration from git-config
type Config struct {
	// Roots is the list of root directories (last one is primary)
	Roots []string
	// User is the default user name for short repository names
	User string
}

// LoadConfig loads the global configuration from git-config and environment
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	executor := git.NewExecutor("")

	// Load roots from git-config (can be multiple)
	// Priority: BARETREE_ROOT env > git-config > default
	if envRoot := os.Getenv("BARETREE_ROOT"); envRoot != "" {
		cfg.Roots = []string{expandTilde(envRoot)}
	} else {
		roots, err := executor.Execute("config", "--get-all", "baretree.root")
		if err == nil && roots != "" {
			for _, root := range strings.Split(roots, "\n") {
				if root = strings.TrimSpace(root); root != "" {
					cfg.Roots = append(cfg.Roots, expandTilde(root))
				}
			}
		}
		if len(cfg.Roots) == 0 {
			cfg.Roots = []string{expandTilde(DefaultRoot)}
		}
	}

	// Load user from git-config, fallback to OS user
	userVal, err := executor.Execute("config", "--get", "baretree.user")
	if err == nil && userVal != "" {
		cfg.User = userVal
	} else {
		if currentUser, err := user.Current(); err == nil {
			cfg.User = currentUser.Username
		}
	}

	return cfg, nil
}

// PrimaryRoot returns the primary (last) root directory
func (c *Config) PrimaryRoot() string {
	if len(c.Roots) == 0 {
		return expandTilde(DefaultRoot)
	}
	return c.Roots[len(c.Roots)-1]
}

// expandTilde expands ~ to the user's home directory
func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// ExpandTilde expands ~ to the user's home directory (exported version)
func ExpandTilde(path string) string {
	return expandTilde(path)
}

// ExportConfig represents the exportable global configuration
type ExportConfig struct {
	Roots []string `toml:"roots"`
}

// ExportConfigToTOML exports the global configuration to TOML format
func ExportConfigToTOML(cfg *Config) (string, error) {
	exportCfg := ExportConfig{
		Roots: cfg.Roots,
	}

	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(exportCfg); err != nil {
		return "", fmt.Errorf("failed to encode config: %w", err)
	}
	return buf.String(), nil
}

// ImportConfigFromTOML imports global configuration from TOML format
func ImportConfigFromTOML(data string) (*ExportConfig, error) {
	var cfg ExportConfig
	if err := toml.Unmarshal([]byte(data), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}
	return &cfg, nil
}

// SaveRootsToGitConfig saves roots to git-config (global)
func SaveRootsToGitConfig(roots []string) error {
	executor := git.NewExecutor("")

	// First, unset all existing roots
	_, err := executor.Execute("config", "--global", "--unset-all", "baretree.root")
	if err != nil {
		return err
	}

	// Add each root
	for _, root := range roots {
		if _, err := executor.Execute("config", "--global", "--add", "baretree.root", root); err != nil {
			return fmt.Errorf("failed to save root %s: %w", root, err)
		}
	}

	return nil
}

// SetRoot sets the primary root directory in git-config (global)
func SetRoot(path string) error {
	executor := git.NewExecutor("")

	// Expand tilde to absolute path for storage
	expandedPath := expandTilde(path)

	// Set single root (replaces any existing)
	if _, err := executor.Execute("config", "--global", "--replace-all", "baretree.root", expandedPath); err != nil {
		return fmt.Errorf("failed to set root: %w", err)
	}

	return nil
}

// AddRoot appends a root directory to git-config (global)
func AddRoot(path string) error {
	executor := git.NewExecutor("")

	// Expand tilde to absolute path for storage
	expandedPath := expandTilde(path)

	// Add root (appends to existing)
	if _, err := executor.Execute("config", "--global", "--add", "baretree.root", expandedPath); err != nil {
		return fmt.Errorf("failed to add root: %w", err)
	}

	return nil
}

// UnsetRoot removes the root directory setting from git-config (global)
func UnsetRoot() error {
	executor := git.NewExecutor("")

	// Unset all root values
	_, err := executor.Execute("config", "--global", "--unset-all", "baretree.root")
	// Ignore error if key doesn't exist
	if err != nil {
		// Check if it's just "key not found" which is OK
		output, _ := executor.Execute("config", "--global", "--get", "baretree.root")
		if output == "" {
			return nil // Already unset, that's fine
		}
		return fmt.Errorf("failed to unset root: %w", err)
	}

	return nil
}

// GetRootSource returns the source of the current root configuration
// Returns "env" if from BARETREE_ROOT, "git-config" if from git config, "default" if using default
func GetRootSource() string {
	if os.Getenv("BARETREE_ROOT") != "" {
		return "env"
	}

	executor := git.NewExecutor("")
	roots, err := executor.Execute("config", "--get-all", "baretree.root")
	if err == nil && roots != "" {
		return "git-config"
	}

	return "default"
}
