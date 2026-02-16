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
| `TestMigrate_WithNestedBranchName_InPlace` | In-place migration when checked out to nested branch (e.g., feat/xxx) |
| `TestMigrate_WithNestedBranchNameAndExistingDir_InPlace` | In-place migration with nested branch when existing directory matches branch prefix |
| `TestMigrate_WithNestedBranchName_Destination` | Migration with `-d` when checked out to nested branch |

### migrate_default_branch_test.go

Default branch detection tests for migration.

| Test Case | Test Purpose |
|-----------|--------------|
| `TestMigrate_DefaultBranchDetection_InPlace` | Migration from feature branch creates main worktree |
| `TestMigrate_DefaultBranchDetection_FromMain` | Migration from main only creates main worktree |
| `TestMigrate_DefaultBranchDetection_Destination` | Default branch detection with `-d` flag |
| `TestMigrate_DefaultBranchDetection_FallbackToMaster` | Fallback to master when no remote |
| `TestMigrate_DefaultBranchDetection_FallbackToCurrentBranch` | Fallback to current branch when no remote/main/master |
| `TestMigrate_DefaultBranchDetection_FallbackFromFeatureToCurrentBranch` | Feature branch becomes default when no remote/main/master |

### journey_remote_test.go

Remote branch operation tests.

| Test Case | Test Purpose |
|-----------|--------------|
| `TestAddRemoteBranch` | Auto-detection and addition of remote branch |
| `TestAddRemoteBranchExplicit` | Explicit addition with `origin/branch` format |
| `TestAddAutoFetch` | Auto-fetch from remotes by default when adding worktree |
| `TestAddNoFetch` | `--no-fetch` option skips auto-fetch |
| `TestAddUpstreamBehindWarning` | Warning when local branch is behind its upstream |
| `TestAddBranchNotFound` | Error message when adding non-existent branch |
| `TestAddLocalBranchPriority` | Local branch priority over remote |
| `TestAddNewBranchWithRemoteBase` | `--base` with remote-only branch resolves correctly (DWIM bug fix) |
| `TestAddNewBranchWithLocalBase` | `--base` with local branch works correctly |
| `TestAddNewBranchWithNonexistentBase` | Error when `--base` specifies non-existent branch |
| `TestAddNewBranchShowsBaseInfo` | Display of base branch information (HEAD fallback) |
| `TestAddNewBranchWithCommitBase` | `--base` with commit hash (full and short) works correctly |

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
| `TestPostCreateAdd` | Symlink configuration with `bt post-create add symlink` (verifies relative symlinks) |
| `TestPostCreateAddManaged` | Managed mode with .shared directory (verifies relative symlinks) |
| `TestPostCreateAddConflict` | Conflict detection with existing files |
| `TestPostCreateRemove` | Configuration removal with `bt post-create remove` |
| `TestPostCreateList` | Configuration listing with `bt post-create list` |
| `TestPostCreateApply` | Applying to existing worktrees with `bt post-create apply` |
| `TestPostCreateApplyConflict` | Conflict detection during apply |
| `TestStatusShowsPostCreateInfo` | Post-create status display in status command |
| `TestPostCreateAddCommand` | Command type configuration with `bt post-create add command` |
| `TestPostCreateCommandExecution` | Command execution when creating worktree |
| `TestPostCreateCommandFailure` | Graceful handling of command failures |
| `TestPostCreateCommandWithSpaces` | Commands containing spaces (e.g., `echo hello world`) are handled correctly |
| `TestPostCreateCommandWithChainedCommands` | Commands with `&&` and `;` operators are handled correctly |
| `TestPostCreateCommandWithQuotes` | Commands containing double quotes are handled correctly |

### config_default_branch_test.go

Config default-branch command tests.

| Test Case | Test Purpose |
|-----------|--------------|
| `TestConfigDefaultBranch_Get` | Getting the current default branch |
| `TestConfigDefaultBranch_Set` | Setting the default branch |
| `TestConfigDefaultBranch_SetMultipleTimes` | Setting the default branch multiple times |
| `TestConfigDefaultBranch_FromWorktree` | Running command from within a worktree |
| `TestConfigDefaultBranch_NotInBaretreeRepo` | Error handling outside baretree repository |
| `TestConfigDefaultBranch_Unset` | Unsetting the default branch (reverts to 'main') |
| `TestConfigDefaultBranch_UnsetWithArg` | Error when using --unset with branch argument |
| `TestConfigDefaultBranch_Help` | Help output |

### repo_config_root_test.go

Repo config root command tests.

| Test Case | Test Purpose |
|-----------|--------------|
| `TestRepoConfigRoot_Get` | Getting the current root directory |
| `TestRepoConfigRoot_Set` | Setting the root directory |
| `TestRepoConfigRoot_SetSamePath` | Setting the same path shows already set message |
| `TestRepoConfigRoot_Unset` | Unsetting the root directory (reverts to ~/baretree) |
| `TestRepoConfigRoot_UnsetWithArg` | Error when using --unset with path argument |
| `TestRepoConfigRoot_EnvVarWarning` | Warning when BARETREE_ROOT environment variable is set |
| `TestRepoConfigRoot_Help` | Help output |

### journey_synctoroot_test.go

Sync-to-root functionality tests.

| Test Case | Test Purpose |
|-----------|--------------|
| `TestJourneySyncToRoot/setup test files` | Setup test files in main worktree |
| `TestJourneySyncToRoot/add file sync-to-root` | Adding sync-to-root for a file (CLAUDE.md) |
| `TestJourneySyncToRoot/add directory sync-to-root` | Adding sync-to-root for a directory (.claude) |
| `TestJourneySyncToRoot/add sync-to-root with custom target` | Custom target path (docs/guide.md -> guide.md) |
| `TestJourneySyncToRoot/list sync-to-root entries` | Listing all sync-to-root entries |
| `TestJourneySyncToRoot/remove sync-to-root entry` | Removing sync-to-root entry |
| `TestJourneySyncToRoot/apply recreates missing symlinks` | Apply command recreates missing symlinks |
| `TestJourneySyncToRoot/status shows sync-to-root` | Status command shows sync-to-root entries |
| `TestSyncToRootErrors/error on non-existent source` | Error when source does not exist |
| `TestSyncToRootErrors/error on existing non-symlink target` | Error when target is a regular file |
| `TestSyncToRootErrors/error on duplicate entry` | Error when adding duplicate entry |
| `TestSyncToRootForce/force overwrites wrong symlink` | --force flag overwrites incorrect symlinks |

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

### completion_test.go

Shell completion tests using Cobra's `__complete` mechanism.

| Test Case | Test Purpose |
|-----------|--------------|
| `TestCompletion_Worktree/cd_command_completes_worktree_names_with_special_args` | cd command completes worktree names and @/- |
| `TestCompletion_Worktree/remove_command_completes_worktree_names_without_special_args` | remove command completes worktree names only |
| `TestCompletion_Worktree/rename_command_completes_worktree_names` | rename command completes worktree names |
| `TestCompletion_Worktree/repair_command_completes_worktree_names` | repair command completes worktree names |
| `TestCompletion_Worktree/unbare_first_arg_completes_worktree_names` | unbare first arg completes worktree names |
| `TestCompletion_Worktree/unbare_second_arg_falls_back_to_file_completion` | unbare second arg uses file completion |
| `TestCompletion_Flags/add_command_completes_flags` | add command completes --branch, --base, etc. |
| `TestCompletion_Flags/remove_command_completes_flags` | remove command completes --force, --with-branch |
| `TestCompletion_Flags/list_command_completes_flags` | list command completes --json, --paths |
| `TestCompletion_Flags/repair_command_completes_flags` | repair command completes --dry-run, --source, etc. |
| `TestCompletion_Subcommands/root_command_completes_subcommands` | Root command completes add, cd, repo, etc. |
| `TestCompletion_Subcommands/repo_command_completes_subcommands` | repo command completes init, clone, etc. |
| `TestCompletion_Subcommands/config_command_completes_subcommands` | config command completes default-branch, export, import |
| `TestCompletion_Subcommands/post-create_command_completes_subcommands` | post-create command completes add, remove, etc. |
| `TestCompletion_Subcommands/sync-to-root_command_completes_subcommands` | sync-to-root command completes add, remove, etc. |
| `TestCompletion_Subcommands/shell-init_completes_shell_names` | shell-init completes bash, zsh, fish |
| `TestCompletion_Repository/go_command_completes_repository_names_with_-` | go command completes repository names and - |
| `TestCompletion_Repository/repo_cd_command_completes_repository_names` | repo cd command completes repository names |
| `TestCompletion_Repository/repos_command_completes_repository_names_without_-` | repos command completes repository names (no -) |
| `TestCompletion_PartialMatch/partial_match_filters_completions` | Partial input filters completion candidates |
| `TestCompletion_SubstringMatch/worktree_completion_includes_both_prefix_and_substring_matches` | Worktree completion includes prefix and substring matches |
| `TestCompletion_SubstringMatch/worktree_completion_prioritizes_prefix_matches_over_substring_matches` | Prefix matches come before substring matches |
| `TestCompletion_SubstringMatch/special_completions_excluded_when_filter_is_applied` | @/- excluded when filter is applied |
| `TestCompletion_RepositorySubstringMatch/repository_completion_includes_both_prefix_and_substring_matches` | Repository completion includes prefix and substring matches |
| `TestCompletion_RepositorySubstringMatch/repository_completion_prioritizes_prefix_matches` | Repository prefix matches come before substring matches |
| `TestCompletion_RepositorySubstringMatch/special_completion_'-'_excluded_when_filter_is_applied` | - excluded when filter is applied |
| `TestCompletion_SyncToRoot/sync-to-root_add_completes_files_in_main_worktree` | sync-to-root add completes files in default worktree |
| `TestCompletion_SyncToRoot/sync-to-root_add_filters_by_prefix` | sync-to-root add filters completions by prefix |
| `TestCompletion_SyncToRoot/sync-to-root_remove_completes_configured_entries` | sync-to-root remove completes configured entries |

## Running Tests

```bash
# Run all tests
go test -v ./e2e/...

# Short mode (skip E2E tests)
go test -short ./e2e/...

# Run specific test
go test -v -run TestJourney1_BasicWorkflow ./e2e/...
```
