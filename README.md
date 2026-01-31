> [!WARNING]
> **This is an experimental project that is not yet thoroughly tested; use with caution as breaking changes are likely. Bug reports and contributions are welcome.**

<h1 align="center">🪾 baretree</h1>

<p align="center"><b>Git repositories and worktrees, organized.</b></p>

baretree combines centralized repository management (inspired by [ghq](https://github.com/x-motemen/ghq)) with powerful Git worktree support. Manage all your repositories in one place, keep branches organized, and switch contexts instantly.

## Why baretree?

### Before
```
~/projects/
├── my-app/                    # Clone 1
├── my-app-feature/            # Clone 2 for feature work
├── my-app-hotfix/             # Clone 3 for hotfix
└── ...scattered everywhere
```

### After
```
~/baretree/                              # All repositories organized
├── github.com/
│   └── user/
│       ├── my-app/                      # One repository
│       │   ├── .bare/                   # Git data
│       │   ├── main/                    # main branch
│       │   ├── feature/auth/            # feature/auth branch
│       │   └── .shared/                 # Shared files (.env, etc.)
│       └── another-project/
└── gitlab.com/
    └── ...
```

- **All worktrees in one directory** - No more scattered clones for each branch
- **Shared files across worktrees** - `.env`, `node_modules` linked automatically
- **Instant context switching** - Jump between worktrees (`bt cd`) and repositories (`bt go`)
- **Migrate existing repos instantly** - One command converts any repository
- **Centralized repository management** - All repos in `~/baretree/{host}/{user}/{repo}`
- **AI Agent ready** - Parallel tasks and isolated workspaces for AI coding assistants

## 🚀 Quick Start

### 1. Install

```bash
# Homebrew (macOS/Linux)
brew install amaya382/tap/baretree

# From source
go install github.com/amaya382/baretree/cmd/bt@latest
```

### 2. Shell Integration (required for `bt cd`)

This is required for `bt cd` and completion, however, you can skip the manual setup if installed via Homebrew (shell integration is automatically configured).

> **Note (Homebrew users):** After installing via Homebrew, open a new terminal window for shell integration to take effect.

```bash
# Bash
echo 'eval "$(bt shell-init bash)"' >> ~/.bashrc && source ~/.bashrc

# Zsh
echo 'eval "$(bt shell-init zsh)"' >> ~/.zshrc && source ~/.zshrc

# Fish
echo 'bt shell-init fish | source' >> ~/.config/fish/config.fish && source ~/.config/fish/config.fish
```

---

## 📖 Getting Started

Choose your style: **Centralized** (recommended) or **Standalone**.

### Option A: Centralized (Recommended)

Manage all repositories in a central location (`~/baretree` by default).

#### Migrate existing repositories to baretree root

```bash
cd ~/projects/my-existing-repo
bt migrate . --to-root
# Your repo is now baretree-structured and placed under ~/baretree
# Working tree state (staged, unstaged, untracked) is preserved
bt go my-existing-repo
```

#### Clone a new repository

```bash
# Clone to ~/baretree/github.com/user/repo
bt get user/repo  # Defaults to github.com
bt get github.com/user/repo # With domain
bt get git@github.com:user/repo.git # By full URL
```

#### Navigate between repositories

```bash
bt repos                  # List all repositories
bt go my-repo             # Jump to repository
bt go user/repo           # Jump with more specific path
```

#### Work with worktrees

```bash
bt add -b feature/auth            # Create feature branch
bt cd feature/auth                # Jump to worktree
bt ls                             # List all worktrees
bt rm feature/auth                # Remove when done
bt unbare main ~/standalone-repo  # Export worktree as standalone repo
```

### Option B: Standalone (without centralized management)

Use baretree for a single project without centralized repository management.

#### Start fresh

```bash
bt init my-project
cd my-project
bt cd @
# Start coding!
```

#### Migrate an existing repository in-place

```bash
cd ~/projects/my-repo
bt migrate . --in-place
# Your repo is now baretree-structured
# Working tree state (staged, unstaged, untracked) is preserved
```

#### Clone to a specific location

```bash
bt clone git@github.com:user/repo.git ~/projects/my-project
cd ~/projects/my-project
```

#### Work with worktrees

```bash
bt add -b feature/auth            # Create feature branch
bt cd feature/auth                # Jump to worktree
bt ls                             # List all worktrees
bt rm feature/auth                # Remove when done
bt unbare main ~/standalone-repo  # Export worktree as standalone repo
```

---

## 🔗 Shared Files

Share files like `.env` or `node_modules` across all worktrees automatically.

```bash
# Add shared files (symlink by default, stored in .shared/ directory)
bt shared add .env
bt shared add node_modules

# Use copy instead of symlink for files that need independent copies
bt shared add .vscode/settings.json --type copy

# Use --no-managed to source from default branch instead of .shared/
bt shared add .env --no-managed
```

New worktrees automatically get these files linked.

### More commands

```bash
bt shared list              # Show configured files
bt shared apply             # Apply to all worktrees
bt shared remove .env       # Remove configuration
```

---

## 🤖 AI Agent Integration

baretree's worktree-based structure is ideal for AI coding assistants like Claude Code, Cursor, and GitHub Copilot Workspace. Each worktree provides an isolated workspace.

### Agent Rules Template

An example rules file for AI agents is available at [`examples/rules/git_worktree_with_baretree.md`](examples/rules/git_worktree_with_baretree.md). This template helps AI agents understand the baretree structure and work effectively with worktrees.

Copy to your project's rules directory:

```bash
# For Claude Code
cp examples/rules/git_worktree_with_baretree.md .claude/rules/

# For Cursor
cp examples/rules/git_worktree_with_baretree.md .cursor/rules/
```

---

## 📚 Command Reference

### Worktree Management

| Command | Description |
|---------|-------------|
| `bt add <branch>` | Add worktree (`-b` for new branch) |
| `bt list` / `bt ls` | List worktrees |
| `bt remove` / `bt rm` | Remove worktree (`--with-branch` to delete branch) |
| `bt cd <name>` | Switch to worktree (`@` for default, `-` for previous) |
| `bt status` | Show repository status |
| `bt repair` | Repair worktree/branch name mismatches |
| `bt rename [old] <new>` | Rename worktree and branch |
| `bt unbare <wt> <dest>` | Convert worktree to standalone repository |
| `bt root` | Show repository root directory path |

### Repository Management (Centralized)

| Command | Alias | Description |
|---------|-------|-------------|
| `bt repo get <url>` | `bt get` | Clone to baretree root (centralized-style) |
| `bt repo list` | `bt repos` | List all managed repositories |
| `bt repo cd <name>` | `bt go` | Jump to a repository |
| `bt repo migrate <path> --to-root` | `bt migrate` | Migrate and move to baretree root |
| `bt repo remove <name>` | `bt repo rm` | Remove a baretree repository |
| `bt repo root` | | Show baretree root directory |
| `bt repo config` | | Manage global configuration |

### Repository Management (Standalone)

| Command | Alias | Description |
|---------|-------|-------------|
| `bt repo init [dir]` | `bt init` | Initialize new baretree repository |
| `bt repo clone <url> [dest]` | `bt clone` | Clone to specific location |
| `bt repo migrate <path> -i` | `bt migrate` | Convert existing repo in-place |
| `bt repo migrate <path> -d <dest>` | `bt migrate` | Convert and copy to destination |

### Shared Files

| Command | Description |
|---------|-------------|
| `bt shared add <file>` | Add shared file |
| `bt shared remove <file>` | Remove shared file |
| `bt shared list` | List shared files |
| `bt shared apply` | Apply to all worktrees |

### Configuration

| Command | Description |
|---------|-------------|
| `bt config export` | Export repository config to TOML |
| `bt config import` | Import repository config from TOML |
| `bt repo config export` | Export global config to TOML |
| `bt repo config import` | Import global config from TOML |

---

## ⚙️ Configuration

All configuration is stored in git-config (no extra config files needed).

### Baretree Root

Set where repositories are stored (default: `~/baretree`):

```bash
# Environment variable
export BARETREE_ROOT=~/code

# Or git config
git config --global baretree.root ~/code
```

> [!TIP]
> Set `BARETREE_ROOT=~/ghq` to use baretree alongside ghq.
> Run `bt repo migrate . -i` to add worktree support in each repository.

---

## 📦 Install

### Homebrew (macOS/Linux)

```bash
brew install amaya382/tap/baretree
```

### [In preparation] Snap (Linux)

```bash
sudo snap install baretree --classic
```

### [In preparation] Scoop (Windows)

```powershell
scoop bucket add amaya382 https://github.com/amaya382/scoop-bucket
scoop install baretree
```

### From Source

```bash
go install github.com/amaya382/baretree/cmd/bt@latest
```

---

## 🔧 Troubleshooting

### `bt cd` doesn't work

Make sure shell integration is enabled:

```bash
# Check if the shell function and the command exist
type bt
type -p bt

# Re-add shell integration
eval "$(bt shell-init bash)"  # or zsh/fish
```

### Symlinks don't work on Windows

1. Enable Developer Mode (Windows 10+)
2. Or use `--type copy` instead

### Can't remove worktree (uncommitted changes)

```bash
bt rm feature/branch --force
```

### Worktree and branch names don't match

```bash
bt repair --dry-run --all     # Preview changes
bt repair --all               # Fix (use branch name as source)
bt repair --source=dir --all  # Fix (use directory name as source)
```

---

## 📋 Requirements

- Git 2.15+
- Go 1.23+ (building from source)

## Related Projects

- [ghq](https://github.com/x-motemen/ghq) - Repository management tool (inspiration for `bt repo get/list/remove`)
- [wtp](https://github.com/satococoa/wtp) - Worktree management tool (inspiration for `bt shared`)
