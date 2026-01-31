# Baretree Git Worktree Integration

## Overview

This project uses [baretree](https://github.com/amaya382/baretree) (`bt`) for git worktree management. Baretree organizes the repository around a bare repository (`.git/`) with multiple worktrees for different branches.

**Repository Structure:**

```
project/                       # Repository root
├── .git/                      # Bare git repository (git internals)
├── .shared/                   # Shared files (symlinked across worktrees)
├── main/                      # Main branch worktree
├── feature/
│   ├── auth/                  # Worktree for feature/auth branch
│   └── api/                   # Worktree for feature/api branch
└── task/
    └── explore-1/             # Temporary worktree for parallel tasks
```

## Detection and Context Awareness

**Check baretree environment:**

```bash
# Verify baretree repository and show status
bt status

# List all worktrees (shows current worktree with *, default with @)
bt list

# Show repository root directory path
bt root
```

**Working Directory Rules:**

- Always operate in the current worktree directory
- Use absolute paths for file operations
- `bt list` shows the current worktree marked with `*` and the default worktree marked with `@`
- `bt status` provides repository information, current branch, and shared file status

## Worktree Creation Workflow

When implementing new features that require a new branch:

1. **Detect Need**: User requests feature development that doesn't exist in current branch
2. **Ask Permission**: Confirm worktree creation with the user:
   - Question: "Create new worktree for `feature/<feature-name>`?"
   - Include branch name suggestion based on feature description
3. **Create Worktree**:
   ```bash
   bt add -b feature/<feature-name>
   ```
4. **Navigate**: Inform user of new worktree location:
   ```
   New worktree created at: /path/to/project/feature/<feature-name>/
   To switch: bt cd feature/<feature-name>
   ```
5. **Work in New Context**: Use absolute path for subsequent operations

**Branch Naming Convention:**

- Features: `feature/<feature-name>`
- Bugfixes: `bugfix/<bug-name>` or `fix/<bug-name>`
- Temporary tasks: `task/<task-id>`

## Parallel Task Isolation

When running parallel tasks (e.g., multiple agents simultaneously):

1. **Create Temporary Worktrees**: For each parallel task, create isolated worktree:

   ```bash
   bt add -b task/explore-error-handling
   bt add -b task/explore-database-layer
   ```

2. **Pass Context to Subagents**: Include worktree path in agent prompts:

   ```
   Explore the error handling patterns in the codebase.

   IMPORTANT: Work in the worktree at:
   /path/to/project/task/explore-error-handling/

   All file operations must use absolute paths within this worktree.
   ```

3. **Unique Naming**: Use descriptive, unique names to avoid conflicts:
   - `task/explore-<topic>-<timestamp>`
   - `task/plan-<feature>-<id>`

4. **Cleanup After Completion**: See Cleanup section below

## Navigation

- Always use absolute paths: `/path/to/project/feature/auth/src/...`
- Don't rely on `cd` commands - they won't persist across tool calls
- Verify worktree exists before operations: `ls /path/to/worktree`
- Use `bt root` to get the repository root path

Provide copy-pasteable commands for users:

```bash
bt cd feature/auth   # Switch to feature worktree
bt cd @              # Switch to default branch
bt cd -              # Return to previous worktree
```

## Cleanup After Task Completion

When feature is merged or task is complete:

1. **Detect Completion**: After successful merge or PR merge
2. **Ask Permission**:
   - Question: "Remove worktree `feature/<feature>` and its branch?"
   - Explain: "Feature has been merged to main"
3. **Remove Worktree**:
   ```bash
   bt rm feature/<feature> --with-branch
   ```

**Temporary Task Cleanup:**
For parallel task worktrees (`task/*`), clean up immediately after task:

```bash
bt rm task/explore-1 --with-branch
```

## Shared File Awareness

Baretree can share files across worktrees via symlinks. New worktrees automatically get configured shared files linked.

**Check Shared Configuration:**

```bash
bt shared list
```

**Understanding Shared Files:**

- Files listed by `bt shared list` are shared across all worktrees
- Shared files are stored in `.shared/` directory and symlinked to each worktree
- Changes to symlinked files affect **all worktrees simultaneously**

**Before Editing Shared Files:**

1. **Check if file is shared**: Run `bt shared list`
2. **Detect symlinks**: Verify with `ls -la <filename>` (symlinks show `->` arrow)
3. **Warn user** if file is shared:

   ```
   Warning: <filename> is a shared file (symlink).
   Changes will affect all worktrees.
   ```

## Troubleshooting

**Worktree Mismatch:**
If worktrees and branches are out of sync:

```bash
bt repair --dry-run --all   # Preview changes
bt repair --all             # Fix mismatches
```

## Commands Reference

**Status:**

```bash
bt status              # Show repository status
bt list                # List all worktrees (* = current, @ = default)
bt root                # Show repository root path
```

**Worktree Management:**

```bash
bt add <branch>                # Add worktree for existing branch
bt add -b <branch>             # Create new branch and worktree
bt add --fetch <branch>        # Fetch latest before adding remote branch
bt rm <worktree>               # Remove worktree (keep branch)
bt rm <worktree> --with-branch # Remove worktree and branch
bt repair --all                # Fix worktree/branch mismatches
```

**Shared Files:**

```bash
bt shared list         # List configured shared files
bt shared add <file>   # Add shared file
bt shared remove <file># Remove from shared config
bt shared apply        # Apply shared files to all worktrees
```
