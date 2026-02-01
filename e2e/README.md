# E2E Test Specification

This directory contains E2E (End-to-End) tests for the baretree CLI.

## Test Files

### journey_basic_test.go

Basic workflow tests.

| Test Case | Test Purpose |
|-----------|--------------|
| `TestJourney1_BasicWorkflow` | Basic operation flow: clone → list → add → cd → status |
| `TestAddExistingWorktree` | Guidance message when adding an existing worktree |
| `TestJourney2_MultipleFeaturesAndCleanup` | Adding and removing multiple feature branches |
| `TestJourney3_MigrateExistingRepo` | Migrating an existing git repository |

### journey_error_test.go

Error handling tests.

| Test Case | Test Purpose |
|-----------|--------------|
| `TestJourney7_ErrorHandling/clone invalid url` | Clone failure with invalid URL |
| `TestJourney7_ErrorHandling/add with existing branch` | Failure when adding with `-b` using existing branch name |
| `TestJourney7_ErrorHandling/remove non-existent worktree` | Failure when removing non-existent worktree |
| `TestJourney7_ErrorHandling/commands outside baretree repo` | Failure when running commands outside baretree |
| `TestJourney7_ErrorHandling/cd to non-existent worktree` | Failure when cd to non-existent worktree |
| `TestJourney7_ErrorHandling/clone to existing directory` | Failure when cloning to existing directory |
| `TestJourney7_ErrorHandling/repair non-existent worktree` | Failure when repairing non-existent worktree |
| `TestJourney7_ErrorHandling/migrate non-git directory` | Failure when migrating non-git directory |
| `TestErrorMessages` | Error message helpfulness |

### journey_hierarchy_test.go

Hierarchical branch name tests.

| Test Case | Test Purpose |
|-----------|--------------|
| `TestJourney8_HierarchicalBranchNames` | Directory hierarchy creation for branch names with slashes |
| `TestRefConflict` | User-friendly error for conflicting ref names (e.g., `feat` and `feat/child`) |
| `TestPathOutput` | `list --paths` outputs paths only |

### journey_init_test.go

Repository initialization tests.

| Test Case | Test Purpose |
|-----------|--------------|
| `TestJourney_Init` | New repository initialization and basic operations |
| `TestInit_CustomBranch` | Custom default branch specification |
| `TestInit_InCurrentDirectory` | Initialization in current directory |
| `TestInit_WithExistingFiles` | Initialization with existing files (file relocation) |
| `TestInit_ErrorCases` | Failure when already a baretree/git repository |

### journey_migrate_test.go

Repository migration tests.

| Test Case | Test Purpose |
|-----------|--------------|
| `TestMigrate_InPlace` | In-place migration with `-i` flag |
| `TestMigrate_Destination` | Migration to another directory with `-d` flag |
| `TestMigrate_Destination_RemoveSource` | Migration with `-d` and `--remove-source` to delete original |
| `TestMigrate_PreservesWorkingTreeState` | Preserving unstaged/staged/untracked files |
| `TestMigrate_PreservesWorkingTreeState_InPlace` | Working tree state preservation in in-place migration |
| `TestMigrate_WithExistingWorktrees` | Automatic conversion of external worktrees during migration |
| `TestMigrate_WithExistingWorktrees_InPlace` | Automatic conversion of external worktrees in in-place migration |
| `TestMigrate_WithMultipleExternalWorktrees` | Automatic conversion of multiple external worktrees |
| `TestMigrate_WithHierarchicalBranchWorktree` | Migration with hierarchical branch names (e.g., feature/foo) |
| `TestMigrate_RequiresFlag` | `-i` or `-d` flag is required |
| `TestMigrate_ToRoot` | Migration to BARETREE_ROOT with `--to-managed` (preserves original by default) |
| `TestMigrate_ToRoot_RemoveSource` | Migration with `--to-managed` and `--remove-source` to delete original |
| `TestMigrate_ToRoot_WithPath` | Explicit path specification with `--to-managed --path` |
| `TestMigrate_ToRoot_ExistingBaretree` | Moving existing baretree repository to root |
| `TestMigrate_ToRoot_ExistingBaretreeWithHierarchicalWorktree` | Moving existing baretree with hierarchical worktrees to root |
| `TestMigrate_ToRoot_PreservesState` | Working tree state preservation in root migration |
| `TestMigrate_ToRoot_RequiresRemoteOrPath` | Failure without remote or --path |
| `TestMigrate_ToRoot_DestinationExists` | Failure when destination already exists |
| `TestMigrate_PreservesDeletedFiles` | Preserving deleted files (staged/unstaged) |
| `TestMigrate_PreservesDeletedFiles_InPlace` | Deleted file state preservation in in-place migration |
| `TestMigrate_PreservesSymlinks` | Symlink preservation |
| `TestMigrate_PreservesHiddenFiles` | Hidden file/directory preservation |
| `TestMigrate_PreservesGitignored` | .gitignore target file preservation |
| `TestMigrate_PreservesRenamedFiles` | Renamed file preservation |
| `TestMigrate_PreservesSubdirectoryFiles` | Subdirectory file preservation |
| `TestMigrate_PreservesEmptyDirectories` | Empty directory preservation |
| `TestMigrate_WithSubmodule` | Git submodule preservation |
| `TestMigrate_WithSubmodule_InPlace` | Submodule preservation in in-place migration |
| `TestMigrate_ToRoot_WithSubmodule` | Submodule preservation with `--to-managed` migration |
| `TestMigrate_WithMultipleSubmodules` | Multiple submodules preservation |
| `TestMigrate_WithMultipleSubmodules_InPlace` | Multiple submodules in in-place migration |
| `TestMigrate_WithNestedSubmodule` | Nested submodules (submodule within submodule) preservation |
| `TestMigrate_WithNestedSubmodule_InPlace` | Nested submodules in in-place migration |
| `TestMigrate_WithExternalWorktreesAndSubmodule` | Migration with both external worktrees and submodules |
| `TestMigrate_WithExternalWorktreesAndSubmodule_InPlace` | In-place migration with external worktrees and submodules |
| `TestMigrate_Destination_CustomWorktreeName` | Migration with custom worktree directory name (different from branch) |
| `TestMigrate_ToRoot_WithExternalWorktrees` | `--to-managed` migration of regular git repo with external worktrees |
| `TestMigrate_ToRoot_DeepHierarchicalBranch` | `--to-managed` with deep hierarchical branch names (3+ levels) |
| `TestMigrate_ToRoot_MultipleHierarchicalWorktrees` | `--to-managed` with multiple hierarchical worktrees |
| `TestMigrate_Destination_DetachedHead` | Migration with detached HEAD worktree |
| `TestMigrate_ToRoot_PreservesWorkingStateWithHierarchicalWorktree` | Working state preservation in hierarchical worktree during `--to-managed` |

### journey_remote_test.go

Remote branch operation tests.

| Test Case | Test Purpose |
|-----------|--------------|
| `TestAddRemoteBranch` | Auto-detection and addition of remote branch |
| `TestAddRemoteBranchExplicit` | Explicit addition with `origin/branch` format |
| `TestAddWithFetch` | Fetching latest branches with `--fetch` option |
| `TestAddBranchNotFound` | Error message when adding non-existent branch |
| `TestAddLocalBranchPriority` | Local branch priority over remote |

### journey_rename_test.go

Worktree rename tests.

| Test Case | Test Purpose |
|-----------|--------------|
| `TestRename_Basic` | Basic rename with two arguments |
| `TestRename_CurrentWorktree` | Single argument rename from inside worktree |
| `TestRename_HierarchicalToFlat` | Rename from hierarchical to flat name |
| `TestRename_FlatToHierarchical` | Rename from flat to hierarchical name |
| `TestRename_ErrorCases` | Errors for non-existent/existing/same name/inconsistent state |

### journey_repair_test.go

Worktree repair tests.

| Test Case | Test Purpose |
|-----------|--------------|
| `TestRepair_BranchAsSource` | Repair using branch name as source of truth (directory rename) |
| `TestRepair_DirAsSource` | Repair using directory name as source of truth (branch rename) |
| `TestRepair_SpecificWorktree` | Repairing specific worktree only |
| `TestRepair_CurrentWorktree` | Repair from inside worktree |
| `TestRepair_NoInconsistency` | Behavior when no inconsistencies exist |
| `TestRepair_ErrorCases` | Invalid source value / non-existent worktree |
| `TestRepair_MultipleInconsistencies` | Batch repair of multiple worktrees |
| `TestRepair_ManuallyMovedWorktree` | Repair worktree moved to external location using `bt repair --fix-paths <path>` |
| `TestRepair_MovedBareRepository` | Repair after entire project moved using `bt repair --fix-paths` |
| `TestRepair_PathChanged` | Repair after path change (home directory rename) using `bt repair --fix-paths` |

### journey_postcreate_test.go

Post-create file configuration tests.

| Test Case | Test Purpose |
|-----------|--------------|
| `TestJourney4_PostCreateFiles` | Post-create file configuration and application to new worktree |
| `TestPostCreateFileCopy` | Copy type post-create files |

### journey_postcreate_cmd_test.go

Post-create command tests.

| Test Case | Test Purpose |
|-----------|--------------|
| `TestPostCreateAdd` | Symlink configuration with `bt post-create add symlink` |
| `TestPostCreateAddManaged` | Managed mode with .shared directory |
| `TestPostCreateAddConflict` | Conflict detection with existing files |
| `TestPostCreateRemove` | Configuration removal with `bt post-create remove` |
| `TestPostCreateList` | Configuration listing with `bt post-create list` |
| `TestPostCreateApply` | Applying to existing worktrees with `bt post-create apply` |
| `TestPostCreateApplyConflict` | Conflict detection during apply |
| `TestStatusShowsPostCreateInfo` | Post-create status display in status command |
| `TestPostCreateAddCommand` | Command type configuration with `bt post-create add command` |
| `TestPostCreateCommandExecution` | Command execution when creating worktree |
| `TestPostCreateCommandFailure` | Graceful handling of command failures |

### journey_unbare_test.go

Conversion from baretree to regular repository tests.

| Test Case | Test Purpose |
|-----------|--------------|
| `TestUnbare_Basic` | Basic unbare operation |
| `TestUnbare_PreservesWorkingTreeState` | Preserving unstaged/staged/untracked files |
| `TestUnbare_PreservesDeletedFiles` | Deleted file state preservation |
| `TestUnbare_PreservesSymlinks` | Symlink preservation |
| `TestUnbare_PreservesHiddenFiles` | Hidden file/directory preservation |
| `TestUnbare_PreservesGitignored` | .gitignore target file preservation |
| `TestUnbare_PreservesRenamedFiles` | Renamed file preservation |
| `TestUnbare_PreservesSubdirectoryFiles` | Subdirectory file preservation |
| `TestUnbare_PreservesEmptyDirectories` | Empty directory preservation |
| `TestUnbare_WithSubmodule` | Git submodule preservation |
| `TestUnbare_WithMultipleSubmodules` | Multiple submodules preservation in unbare |
| `TestUnbare_WithNestedSubmodule` | Nested submodules preservation in unbare |
| `TestUnbare_SubmoduleOperations` | Submodule operations work after unbare |
| `TestUnbare_SubmoduleStagingState` | Submodule staging state preservation |
| `TestUnbare_DestinationExists` | Failure when destination already exists |

## Running Tests

```bash
# Run all tests
go test -v ./e2e/...

# Short mode (skip E2E tests)
go test -short ./e2e/...

# Run specific test
go test -v -run TestJourney1_BasicWorkflow ./e2e/...
```
