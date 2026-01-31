package config

// BareDir is the fixed directory name for the bare repository.
// This is intentionally fixed to ".git" to ensure compatibility with git submodules.
const BareDir = ".git"

// Config represents the baretree configuration.
// Runtime storage: git-config ([baretree] section in .git/config)
// Export/import format: TOML (for 'bt shared export/import')
type Config struct {
	Repository Repository `toml:"repository"`
	Shared     []Shared   `toml:"shared"`
}

// Repository configuration
type Repository struct {
	DefaultBranch string `toml:"default_branch"`
}

// Shared file/directory configuration
type Shared struct {
	Source  string `toml:"source"`
	Type    string `toml:"type"`    // "symlink" or "copy"
	Managed bool   `toml:"managed"` // if true, source is in .shared/ directory
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Repository: Repository{
			DefaultBranch: "main",
		},
		Shared: []Shared{},
	}
}
