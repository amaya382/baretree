# Examples

This directory contains example files and templates for using baretree.

## Directory Structure

```
examples/
└── rules/      # AI agent rules templates
```

## Rules

The `rules/` directory contains templates for configuring AI coding assistants to work with baretree-managed repositories.

### Available Rules

| File | Description |
|------|-------------|
| [`workting-directory-on-git-worktree-with-baretree.md`](rules/workting-directory-on-git-worktree-with-baretree.md) | Working directory management rule for AI agents in baretree repositories |

### workting-directory-on-git-worktree-with-baretree.md

This rule teaches AI agents how to properly work with baretree's worktree-based structure. It covers:

- **Applicability check**: How to verify if a repository is baretree-managed
- **Pre-task checklist**: Commands to run before starting any task
- **New feature workflow**: When and how to create new worktrees
- **User permission**: Always ask before creating worktrees or modifying shared files
- **Working with worktrees**: The `cd && command` pattern for shell sessions
- **Shared files handling**: Warnings about symlinked files that affect all worktrees
- **Cleanup procedures**: How to remove worktrees after task completion

### Installation

#### Claude Code (Global)

Place in your global rules directory so it applies to all baretree repositories:

```bash
cp examples/rules/workting-directory-on-git-worktree-with-baretree.md ~/.claude/rules/
```

#### Claude Code (Per-project)

Place in the project's rules directory:

```bash
cp examples/rules/workting-directory-on-git-worktree-with-baretree.md .claude/rules/
```

#### Cursor

```bash
cp examples/rules/workting-directory-on-git-worktree-with-baretree.md .cursor/rules/
```

#### Other AI Tools

Copy the rule file to the appropriate rules/prompts directory for your AI tool.

### Customization

Feel free to customize the rules file for your specific workflow:

- Modify branch naming conventions to match your team's style
- Add project-specific post-create commands
- Adjust the worktree creation criteria
