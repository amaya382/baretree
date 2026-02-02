package shell

const ZshScript = `# baretree shell integration for zsh

bt() {
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

# Source Cobra-generated completion (defines _bt function)
if (( $+commands[bt] )); then
    source <(command bt completion zsh)
fi

# Custom completion for bt with substring matching
_bt_custom() {
    local cur="${words[CURRENT]}"
    local cmd="${words[2]}"
    local subcmd="${words[3]}"

    # Worktree completion (bt cd, bt remove, bt rename, bt repair, bt unbare)
    if [[ "$cmd" == "cd" ]] || \
       [[ "$cmd" == "remove" ]] || \
       [[ "$cmd" == "rename" ]] || \
       [[ "$cmd" == "repair" ]] || \
       [[ "$cmd" == "unbare" ]]; then
        local completions
        completions=("${(@f)$(command bt __complete "$cmd" "$cur" 2>/dev/null | sed '$d')}")
        # -U: suppress usual matching, -V: keep order (no sorting)
        compadd -U -V unsorted -a completions
        return 0
    fi

    # Repository completion (bt go, bt repo cd, bt repos)
    if [[ "$cmd" == "go" ]] || \
       [[ "$cmd" == "repos" ]] || \
       [[ "$cmd" == "repo" && "$subcmd" == "cd" ]]; then
        local completions
        if [[ "$cmd" == "go" ]]; then
            completions=("${(@f)$(command bt __complete go "$cur" 2>/dev/null | sed '$d')}")
        elif [[ "$cmd" == "repos" ]]; then
            completions=("${(@f)$(command bt __complete repos "$cur" 2>/dev/null | sed '$d')}")
        else
            completions=("${(@f)$(command bt __complete repo cd "$cur" 2>/dev/null | sed '$d')}")
        fi
        # -U: suppress usual matching, -V: keep order (no sorting)
        compadd -U -V unsorted -a completions
        return 0
    fi

    # Fall back to Cobra-generated completion for other commands
    _bt "$@"
}

# Register custom completion for bt (wraps Cobra's completion)
compdef _bt_custom bt
`
