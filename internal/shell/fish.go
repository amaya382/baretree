package shell

const FishScript = `# baretree shell integration for fish

function bt
    # Handle cd command specially (worktree navigation)
    if test "$argv[1]" = "cd"
        set -l target_dir
        if test (count $argv) -eq 1
            set target_dir (command bt cd 2>/dev/null)
        else
            set target_dir (command bt cd $argv[2] 2>/dev/null)
        end

        if test $status -eq 0 -a -n "$target_dir"
            cd $target_dir
        else
            # Show error from bt command
            if test (count $argv) -eq 1
                command bt cd
            else
                command bt cd $argv[2]
            end
        end
    # Handle repo cd command specially (repository navigation)
    else if test "$argv[1]" = "repo" -a "$argv[2]" = "cd"
        set -l target_dir
        set target_dir (command bt repo cd $argv[3] 2>/dev/null)

        if test $status -eq 0 -a -n "$target_dir"
            cd $target_dir
        else
            # Show error from bt command
            command bt repo cd $argv[3]
        end
    # Handle go command specially (alias for repo cd)
    else if test "$argv[1]" = "go"
        set -l target_dir
        set target_dir (command bt go $argv[2] 2>/dev/null)

        if test $status -eq 0 -a -n "$target_dir"
            cd $target_dir
        else
            # Show error from bt command
            command bt go $argv[2]
        end
    else
        # Pass through all other commands
        command bt $argv
    end
end

# Fish completion for bt
complete -c bt -f

# Subcommands
complete -c bt -n "__fish_use_subcommand" -a "add" -d "Create a worktree for a branch"
complete -c bt -n "__fish_use_subcommand" -a "list" -d "List all worktrees"
complete -c bt -n "__fish_use_subcommand" -a "ls" -d "List all worktrees"
complete -c bt -n "__fish_use_subcommand" -a "remove" -d "Remove a worktree"
complete -c bt -n "__fish_use_subcommand" -a "rm" -d "Remove a worktree"
complete -c bt -n "__fish_use_subcommand" -a "cd" -d "Change to worktree directory"
complete -c bt -n "__fish_use_subcommand" -a "status" -d "Show repository status"
complete -c bt -n "__fish_use_subcommand" -a "rename" -d "Rename a worktree"
complete -c bt -n "__fish_use_subcommand" -a "repair" -d "Repair unmanaged worktrees"
complete -c bt -n "__fish_use_subcommand" -a "root" -d "Show repository root directory"
complete -c bt -n "__fish_use_subcommand" -a "unbare" -d "Convert worktree to standalone repository"
complete -c bt -n "__fish_use_subcommand" -a "config" -d "Manage repository configuration"
complete -c bt -n "__fish_use_subcommand" -a "repo" -d "Repository management"
complete -c bt -n "__fish_use_subcommand" -a "shared" -d "Shared files management"
complete -c bt -n "__fish_use_subcommand" -a "shell-init" -d "Generate shell integration"
complete -c bt -n "__fish_use_subcommand" -a "version" -d "Print version information"

# Top-level aliases
complete -c bt -n "__fish_use_subcommand" -a "init" -d "Create new baretree repository (alias)"
complete -c bt -n "__fish_use_subcommand" -a "clone" -d "Clone with baretree structure (alias)"
complete -c bt -n "__fish_use_subcommand" -a "migrate" -d "Convert existing repository (alias)"
complete -c bt -n "__fish_use_subcommand" -a "get" -d "Clone into root (alias)"
complete -c bt -n "__fish_use_subcommand" -a "go" -d "Change to repository directory (alias)"
complete -c bt -n "__fish_use_subcommand" -a "repos" -d "List all repositories (alias)"

# Worktree name completion for cd and remove
complete -c bt -n "__fish_seen_subcommand_from cd remove rm" -a "(command bt list --paths 2>/dev/null | xargs -n1 basename)" -a "@ -"

# Repository name completion for go and repos
complete -c bt -n "__fish_seen_subcommand_from go" -a "(command bt repo list 2>/dev/null | xargs -n1 basename)" -a "-"
complete -c bt -n "__fish_seen_subcommand_from repos" -a "(command bt repo list 2>/dev/null | xargs -n1 basename)"

# Repo subcommands
complete -c bt -n "__fish_seen_subcommand_from repo; and not __fish_seen_subcommand_from init clone migrate list root get cd config" -a "init" -d "Create new baretree repository"
complete -c bt -n "__fish_seen_subcommand_from repo; and not __fish_seen_subcommand_from init clone migrate list root get cd config" -a "clone" -d "Clone with baretree structure"
complete -c bt -n "__fish_seen_subcommand_from repo; and not __fish_seen_subcommand_from init clone migrate list root get cd config" -a "migrate" -d "Convert existing repository"
complete -c bt -n "__fish_seen_subcommand_from repo; and not __fish_seen_subcommand_from init clone migrate list root get cd config" -a "list" -d "List all repositories"
complete -c bt -n "__fish_seen_subcommand_from repo; and not __fish_seen_subcommand_from init clone migrate list root get cd config" -a "root" -d "Show root directory"
complete -c bt -n "__fish_seen_subcommand_from repo; and not __fish_seen_subcommand_from init clone migrate list root get cd config" -a "get" -d "Clone into root (ghq-style)"
complete -c bt -n "__fish_seen_subcommand_from repo; and not __fish_seen_subcommand_from init clone migrate list root get cd config" -a "cd" -d "Change to repository"
complete -c bt -n "__fish_seen_subcommand_from repo; and not __fish_seen_subcommand_from init clone migrate list root get cd config" -a "config" -d "Manage global configuration"

# Repo cd completion
complete -c bt -n "__fish_seen_subcommand_from repo; and __fish_seen_subcommand_from cd" -a "(command bt repo list 2>/dev/null | xargs -n1 basename)" -a "-"

# Shared subcommands
complete -c bt -n "__fish_seen_subcommand_from shared; and not __fish_seen_subcommand_from add remove apply list" -a "add" -d "Add shared file"
complete -c bt -n "__fish_seen_subcommand_from shared; and not __fish_seen_subcommand_from add remove apply list" -a "remove" -d "Remove shared file"
complete -c bt -n "__fish_seen_subcommand_from shared; and not __fish_seen_subcommand_from add remove apply list" -a "apply" -d "Apply configuration"
complete -c bt -n "__fish_seen_subcommand_from shared; and not __fish_seen_subcommand_from add remove apply list" -a "list" -d "List shared files"

# Config subcommands
complete -c bt -n "__fish_seen_subcommand_from config; and not __fish_seen_subcommand_from export import" -a "export" -d "Export configuration"
complete -c bt -n "__fish_seen_subcommand_from config; and not __fish_seen_subcommand_from export import" -a "import" -d "Import configuration"

# Repo config subcommands
complete -c bt -n "__fish_seen_subcommand_from repo; and __fish_seen_subcommand_from config; and not __fish_seen_subcommand_from export import" -a "export" -d "Export configuration"
complete -c bt -n "__fish_seen_subcommand_from repo; and __fish_seen_subcommand_from config; and not __fish_seen_subcommand_from export import" -a "import" -d "Import configuration"
`
