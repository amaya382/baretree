package config

// BareDir is the fixed directory name for the bare repository.
// This is intentionally fixed to ".git" to ensure compatibility with git submodules.
const BareDir = ".git"

// Config represents the baretree configuration.
// Runtime storage: git-config ([baretree] section in .git/config)
// Export/import format: TOML (for 'bt config export/import')
type Config struct {
	Repository Repository         `toml:"repository"`
	PostCreate []PostCreateAction `toml:"postcreate"`
}

// Repository configuration
type Repository struct {
	DefaultBranch string `toml:"default_branch"`
}

// PostCreateAction represents an action to perform after worktree creation.
// Type can be "symlink", "copy", or "command".
type PostCreateAction struct {
	Source  string `toml:"source"`  // file path for symlink/copy, command string for command
	Type    string `toml:"type"`    // "symlink", "copy", or "command"
	Managed bool   `toml:"managed"` // if true, source is in .shared/ directory (symlink/copy only)
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Repository: Repository{
			DefaultBranch: "main",
		},
		PostCreate: []PostCreateAction{},
	}
}
