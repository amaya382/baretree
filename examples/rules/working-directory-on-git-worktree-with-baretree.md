# Working Directory on Git Worktree with baretree

## Applicability

**This rule applies ONLY when working in a baretree-managed repository.**

To verify if this is a baretree repository, check:

1. Run `bt status` - it should succeed and show worktree information
2. The repository root should contain a bare `.git` directory and multiple worktree directories

If `bt status` fails or the repository structure doesn't match baretree format, this rule does NOT apply.

## CRITICAL: What This Rule Requires

**When making any changes in a baretree repository, you MUST create a new worktree before starting work.**

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

## When starting implementation (MOST IMPORTANT)

### 1. Determine if New Worktree is Needed

Create a new worktree when you make changes on the repository:

- ✅ Adding new features (New components, etc.)
- ✅ Changes to existing features
- ✅ Changes spanning multiple files
- ✅ Bug fixes (if affecting multiple files)
- ✅ Chore tasks (CI, refactoring, etc.)

### 2. Ask User Permission (MANDATORY - ALWAYS ASK)

**ALWAYS ask the user for permission with the `AskUserQuestion` tool before proceeding, regardless of your decision.**

If you determine a new worktree IS needed:

Use `AskUserQuestion` tool with:

- Question: "Create new worktree for this feature?"
- Options:
  - "Yes - Create <worktree-name>"
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
bt add --behind=pull -b <worktree-name>
```

#### Worktree Naming Conventions

- New features, updates to existing features: `feat/<feature-name>`
- Chore tasks: `chore/<task-name>`
- Bug fixes: `fix/<bug-name>`
- Temporary exploration: `task/<task-id>`

Note that worktree name can containe `/`.

### 4. Inform User

```
New worktree created at:
/path/to/project/<worktree-name>/

To switch:
bt cd <worktree-name>
```

### 5. Work with cd && Commands

Each Bash tool call runs in a separate shell session, so `cd` alone doesn't persist. **Always use this pattern** for working in a target worktree:

```bash
# Use cd && pattern for all operations
cd "<path-to-target-worktree>" && cat README.md
cd "<path-to-target-worktree>" && mkdir -p src && touch src/index.ts
```

## Cleanup After Task Completion

### Temporary Task Worktree Cleanup

Clean up immediately after temporary task completion:

```bash
bt rm task/<task-id> --with-branch
```

## Pre-Task Checklist

- [ ] Verified this is a baretree repository with `bt status` (if failed, this rule does NOT apply)
- [ ] Checked current worktree with `bt list`
- [ ] Determined if this is an implementation requiring a new worktree
- [ ] **Used `AskUserQuestion` tool to ask user permission** (NOT regular chat responses)
- [ ] If creating new worktree: Created with `bt add --behind=pull -b <worktree-name>`
- [ ] If creating new worktree: Using `cd "<path-to-target-worktree>" && command` pattern for all file operations
- [ ] If editing shared files: Used `AskUserQuestion` tool to warn about impact on all worktrees
