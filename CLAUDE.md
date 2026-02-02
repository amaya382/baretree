# CLAUDE.md

This file provides guidance for Claude Code (claude.ai/claude-code) when working with this repository.

## Project Overview

baretree (`bt`) is a Git worktree management tool centered around bare repositories, written in Go. It provides a higher-level interface for managing multiple worktrees with support for shared files and directories.

## Repository Structure

```
baretree/
├── cmd/bt/           # CLI entry point and commands
├── internal/
│   ├── config/       # TOML configuration handling
│   ├── git/          # Git command execution and parsing
│   ├── repository/   # Repository-level operations
│   ├── shell/        # Shell integration (bash/zsh/fish)
│   └── worktree/     # Worktree management
├── e2e/              # End-to-end tests
├── .github/workflows/ # CI/CD workflows
├── Taskfile.yml      # Task runner configuration
└── .goreleaser.yml   # Release configuration
```

## Development Commands

### Using Task (recommended)

```bash
task build          # Build binary
task test           # Run unit tests
task test:e2e       # Run E2E tests
task test:all       # Run all tests
task fmt            # Format code
task lint           # Run linter
task check          # Run all checks (fmt, vet, lint, test)
task dev -- <args>  # Run with arguments
```

### Using Go directly

```bash
go build -buildvcs=false ./cmd/bt/
go test -v -short ./...        # Unit tests only
go test -v ./e2e/...           # E2E tests
go test -v ./...               # All tests
go vet ./...
```

## Architecture

### Key Packages

- **cmd/bt**: Cobra-based CLI commands
- **internal/git**: Wraps Git operations using `os/exec` (same approach as ghq/wtp)
- **internal/config**: Configuration handling (git-config for storage, TOML for export/import)
- **internal/worktree**: Core worktree management logic
- **internal/repository**: Repository discovery and initialization

### Design Decisions

1. **Bare repository foundation**: Unlike wtp, baretree uses bare repositories as the foundation
2. **Branch-based directory structure**: `feature/auth` branch creates `feature/auth/` directory
3. **Git-config based configuration**: Per-repository config stored in `.git/config` under `[baretree]` section
4. **Shell function for `cd`**: Technical necessity for changing directory in parent shell

### Design Philosophy

1. **Config-less by default**: baretree should work out of the box without any configuration
   - Provide sensible defaults that work for most users
   - baretree itself is stateless - it relies on Git and filesystem state
   - Per-repository state is stored in git-config (`[baretree]` section in `.git/config`)

2. **Git-config for all configuration**: Use git-config (`[baretree]` section) for all configuration
   - Per-repository settings stored in `.git/config`
   - Cross-repository settings (like root directory) in global `~/.gitconfig`
   - TOML format only used for `bt config export/import` (portable configuration sharing)
   - Avoid external config files that duplicate Git's native mechanisms

## Testing

### Unit Tests

Located alongside source files (`*_test.go`). Run with:
```bash
go test -v -short ./...
```

### E2E Tests

Located in `e2e/` directory. Uses actual Git operations with a test repository:
```bash
go test -v ./e2e/...
```

E2E tests clone `https://github.com/amaya382/dotfiles` as a test repository. Some tests (like `bt init`) don't require network access.

**Important**: When adding or modifying E2E tests, you must update `e2e/README.md` to reflect the changes. This README documents the purpose of each test case.

## CI/CD

### GitHub Actions Workflows

- **ci.yml**: Runs on PR and push to main
  - Unit tests and E2E tests
  - golangci-lint
  - Build verification

- **release.yml**: Runs on GitHub Release creation
  - Uses goreleaser to build and publish
  - Creates binaries for Linux/macOS/Windows (amd64/arm64)
  - Updates Homebrew formula in `amaya382/homebrew-tap`

### Release Process

1. Create a new tag: `git tag v1.0.0`
2. Push tag: `git push origin v1.0.0`
3. Create GitHub Release from the tag
4. goreleaser automatically:
   - Builds binaries for all platforms
   - Uploads to GitHub Release
   - Updates Homebrew formula

### Required Secrets

- `GITHUB_TOKEN`: Automatically provided by GitHub Actions
- `HOMEBREW_TAP_GITHUB_TOKEN`: PAT with `repo` scope for updating homebrew-tap

## Installation Methods

### Homebrew (macOS/Linux)

```bash
brew tap amaya382/tap
brew install baretree
```

### Go install

```bash
go install github.com/amaya382/baretree/cmd/bt@latest
```

### From source

```bash
git clone https://github.com/amaya382/baretree.git
cd baretree
task build
```

## Code Style

- Follow standard Go conventions
- All code and comments in English
- Use `go fmt` for formatting
- Run `golangci-lint` before committing

## Pre-completion Checklist

Before considering any task complete, ensure all of the following pass:

```bash
task check          # Or run individually: task build && task test && task lint
```

- **Build**: `task build` must succeed without errors
- **Tests**: `task test` (or `task test:all` for full coverage) must pass
- **Lint**: `task lint` must pass without errors

Do not mark work as complete until all checks pass.

## Documentation Updates

**Important**: When making changes, always update relevant documentation and help texts:

- **README.md**: User-facing documentation
- **CLAUDE.md**: Development guidance for Claude Code
- **examples/**: Example files and usage demonstrations
- **Code comments**: Inline documentation and godoc comments
- **Help texts**: CLI command descriptions (`Short`, `Long`, `Use` in Cobra commands)

## Adding New Commands

1. Create new file in `cmd/bt/` (e.g., `newcmd.go`)
2. Define cobra command with `Use`, `Short`, `Long`, `RunE`
3. Register in `cmd/bt/main.go` via `rootCmd.AddCommand()`
4. Add tests in `e2e/` if appropriate

## Modifying Commands with Aliases

Some commands have aliases defined in `cmd/bt/repo/aliases.go`:

| Original Command | Alias |
|-----------------|-------|
| `bt repo init` | `bt init` |
| `bt repo clone` | `bt clone` |
| `bt repo get` | `bt get` |
| `bt repo list` | `bt repos` |
| `bt repo cd` | `bt go` |
| `bt repo migrate` | `bt migrate` |

**Important**: When modifying a command that has an alias, you must update both:

1. **Original command** (e.g., `cmd/bt/repo/migrate.go`)
2. **Alias command** (in `cmd/bt/repo/aliases.go`)

Ensure the following are synchronized:
- **Flags**: All flags must be registered on both commands
- **Help message (Long)**: Examples should use the appropriate command name (`bt migrate` for alias, `bt repo migrate` for original)
- **Functionality**: Both commands share the same `RunE` function, so logic is automatically synchronized

## Configuration

Per-repository configuration is stored in git-config (`.git/config`):

```ini
[baretree]
    defaultBranch = main

```

Post-create actions are stored as `baretree.postcreate`:
```ini
[baretree]
    postcreate = .env:symlink:managed
    postcreate = config/local.json:copy
    postcreate = npm install:command
```

Key settings:
- `baretree.defaultBranch`: Used to identify the main worktree for post-create files
- `baretree.postcreate`: List of post-create actions (format: `source:type` or `source:type:managed`)

### Export/Import (TOML format)

For sharing configuration between repositories, use `bt config export/import`:

```bash
bt config export -o config.toml    # Export to TOML file
bt config import config.toml       # Import from TOML file
```

TOML format (for export/import only):
```toml
[[postcreate]]
source = ".env"
type = "symlink"
managed = true

[[postcreate]]
source = "npm install"
type = "command"
```
