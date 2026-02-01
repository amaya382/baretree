package shell

const BashScript = `# baretree shell integration for bash

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

# Source Cobra-generated completion
if command -v bt &> /dev/null; then
    source <(command bt completion bash)
fi
`
