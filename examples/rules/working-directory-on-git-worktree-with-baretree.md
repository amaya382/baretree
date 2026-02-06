# Working Directory on Git Worktree with baretree

## Applicability

**This rule applies ONLY when working in a baretree-managed repository.**

To verify if this is a baretree repository, check:

1. Run `bt status` - it should succeed and show worktree information
2. The repository root should contain a bare `.git` directory and multiple worktree directories

If `bt status` fails or the repository structure doesn't match baretree format, this rule does NOT apply.

## CRITICAL: What This Rule Requires

**When implementing new features in a baretree repository, you MUST create a new worktree before starting work.**

This project uses baretree for worktree management. Each feature should be developed in its own isolated worktree. DO NOT work directly in the main worktree.

## Repository Structure

```
project/                       # Repository root
├── .git/                      # Bare git repository
├── main/                      # Main branch worktree (treat as read-only)
├── feat/
│   ├── auth/                  # Worktree for feat/auth branch
│   └── api/                   # Worktree for feat/api branch
└── task/
    └── explore-1/             # Temporary worktree for exploration tasks
```

## Pre-Task Checklist (MANDATORY)

Before starting ANY task in this repository, run these commands:

```bash
# 1. Verify this is a baretree-managed repository
bt status
# If this command fails, this rule does NOT apply - skip to regular workflow

# 2. Check current worktree (* = current, @ = default)
bt list

# 3. Get repository root path
bt root
```

**If `bt status` fails:** This is not a baretree repository. Do not follow the worktree creation workflow in this rule. Work directly in the current directory as you would in a standard git repository.

## New Feature Workflow (MOST IMPORTANT)

### 1. Determine if New Worktree is Needed

Create a new worktree when:

- ✅ Adding new features (New components, etc.)
- ✅ Major changes to existing features
- ✅ Changes spanning multiple files
- ✅ Bug fixes (if affecting multiple files)

### 2. Ask User Permission (MANDATORY - ALWAYS ASK)

**ALWAYS ask the user for permission with the `AskUserQuestion` tool before proceeding, regardless of your decision.**

If you determine a new worktree IS needed:

Use `AskUserQuestion` tool with:

- Question: "Create new worktree for this feature?"
- Options:
  - "Yes - Create feat/<feature-name>"
  - "No - Work in current worktree"

If you determine a new worktree is NOT needed:

Use `AskUserQuestion` tool with:

- Question: "Work in current worktree without creating a new one?"
- Explanation: <brief explanation why new worktree is not needed>
- Options:
  - "Yes - Work in current worktree"
  - "No - Create new worktree instead"

### 3. Create Worktree

```bash
bt add -b feat/<feature-name>
```

### 4. Inform User

```
New worktree created at:
/path/to/project/feat/<feature-name>/

To switch:
bt cd feat/<feature-name>
```

### 5. Work with cd && Commands

Each Bash tool call runs in a separate shell session, so `cd` alone doesn't persist. **Always use this pattern** for working in a target worktree:

```bash
# Use cd && pattern for all operations
cd "<target-worktree>" && cat README.md
cd "<target-worktree>" && mkdir -p src && touch src/index.ts
```

## Branch Naming Conventions

- Features: `feat/<feature-name>`
- Bug fixes: `fix/<bug-name>`
- Temporary exploration: `task/<task-id>`

## Shared Files Warning

### Check Shared Files

```bash
bt post-create list
```

### Impact

- Shared files are stored in `.shared/` directory and symlinked to each worktree
- **Changes to shared files affect ALL worktrees simultaneously**
- **Use `AskUserQuestion` tool to warn user before editing shared files:**

Example:

- Question: "This file is a shared file (symlink). Changes will affect ALL worktrees. Proceed with editing?"
- Options:
  - "Yes - Edit the shared file"
  - "No - Cancel editing"

### Verify Symlinks

```bash
ls -la <filename>  # Shows -> arrow if symlink
```

## Cleanup After Task Completion

### Feature Branch Cleanup

After PR is merged:

1. **Use `AskUserQuestion` tool to ask:**
   - Question: "Feature has been merged to main. Remove worktree and branch?"
   - Options:
     - "Yes - Remove feat/<feature> worktree and branch"
     - "No - Keep the worktree"

2. Execute:
   ```bash
   bt rm feat/<feature> --with-branch
   ```

### Temporary Task Worktree Cleanup

Clean up immediately after task completion:

```bash
bt rm task/<task-id> --with-branch
```

## Common Commands (For User Reference)

```bash
# Switch worktrees
bt cd feat/auth   # Switch to feat/auth worktree
bt cd @              # Switch to default worktree
bt cd -              # Return to previous worktree

# Check status
bt status            # Show repository status
bt list              # List all worktrees
bt root              # Show repository root path
```

## Pre-Task Checklist

- [ ] Verified this is a baretree repository with `bt status` (if failed, this rule does NOT apply)
- [ ] Checked current worktree with `bt list`
- [ ] Determined if this is a new feature implementation requiring new worktree
- [ ] **Used `AskUserQuestion` tool to ask user permission** (NOT regular chat responses)
- [ ] If creating new worktree: Created with `bt add -b feat/<name>`
- [ ] If creating new worktree: Using `cd "<target-worktree>" && command` pattern for all file operations
- [ ] If editing shared files: Used `AskUserQuestion` tool to warn about impact on all worktrees
