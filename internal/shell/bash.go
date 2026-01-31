package shell

const BashScript = `# baretree shell integration for bash

bt() {
    # Check if this is a completion request
    for arg in "$@"; do
        if [[ "$arg" == "--generate-shell-completion" ]]; then
            command bt "$@"
            return $?
        fi
    done

    # Handle cd command specially (worktree navigation)
    if [[ "$1" == "cd" ]]; then
        local target_dir
        if [[ -z "$2" ]]; then
            target_dir=$(command bt cd 2>/dev/null)
        else
            target_dir=$(command bt cd "$2" 2>/dev/null)
        fi

        if [[ $? -eq 0 && -n "$target_dir" ]]; then
            cd "$target_dir"
        else
            # Show error from bt command
            if [[ -z "$2" ]]; then
                command bt cd
            else
                command bt cd "$2"
            fi
        fi
    # Handle repo cd command specially (repository navigation)
    elif [[ "$1" == "repo" && "$2" == "cd" ]]; then
        local target_dir
        target_dir=$(command bt repo cd "$3" 2>/dev/null)

        if [[ $? -eq 0 && -n "$target_dir" ]]; then
            cd "$target_dir"
        else
            # Show error from bt command
            command bt repo cd "$3"
        fi
    # Handle go command specially (alias for repo cd)
    elif [[ "$1" == "go" ]]; then
        local target_dir
        target_dir=$(command bt go "$2" 2>/dev/null)

        if [[ $? -eq 0 && -n "$target_dir" ]]; then
            cd "$target_dir"
        else
            # Show error from bt command
            command bt go "$2"
        fi
    else
        # Pass through all other commands
        command bt "$@"
    fi
}

# Bash completion for bt
_bt_completion() {
    local cur prev words cword
    _init_completion || return

    if [[ $cword -eq 1 ]]; then
        # Complete subcommands
        COMPREPLY=($(compgen -W "add list ls remove rm cd status shell-init repo shared rename repair root unbare config version init clone migrate get go repos" -- "$cur"))
    elif [[ ${words[1]} == "cd" && $cword -eq 2 ]]; then
        # Complete worktree names for cd command
        local worktrees=$(command bt list --paths 2>/dev/null | xargs -n1 basename)
        COMPREPLY=($(compgen -W "$worktrees @ -" -- "$cur"))
    elif [[ ${words[1]} == "remove" || ${words[1]} == "rm" ]]; then
        # Complete worktree names for remove command
        local worktrees=$(command bt list --paths 2>/dev/null | xargs -n1 basename)
        COMPREPLY=($(compgen -W "$worktrees" -- "$cur"))
    elif [[ ${words[1]} == "go" && $cword -eq 2 ]]; then
        # Complete repository names for go command (alias for repo cd)
        local repos=$(command bt repo list 2>/dev/null | xargs -n1 basename)
        COMPREPLY=($(compgen -W "$repos -" -- "$cur"))
    elif [[ ${words[1]} == "repos" && $cword -eq 2 ]]; then
        # Complete repository names for repos command (alias for repo list)
        local repos=$(command bt repo list 2>/dev/null | xargs -n1 basename)
        COMPREPLY=($(compgen -W "$repos" -- "$cur"))
    elif [[ ${words[1]} == "repo" ]]; then
        if [[ $cword -eq 2 ]]; then
            # Complete repo subcommands
            COMPREPLY=($(compgen -W "init clone migrate list root get cd config" -- "$cur"))
        elif [[ ${words[2]} == "cd" && $cword -eq 3 ]]; then
            # Complete repository names for repo cd command
            local repos=$(command bt repo list 2>/dev/null | xargs -n1 basename)
            COMPREPLY=($(compgen -W "$repos -" -- "$cur"))
        fi
    elif [[ ${words[1]} == "shared" && $cword -eq 2 ]]; then
        # Complete shared subcommands
        COMPREPLY=($(compgen -W "add remove apply list" -- "$cur"))
    elif [[ ${words[1]} == "config" && $cword -eq 2 ]]; then
        # Complete config subcommands
        COMPREPLY=($(compgen -W "export import" -- "$cur"))
    fi
}

complete -F _bt_completion bt
`
