package shell

const ZshScript = `# baretree shell integration for zsh

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

# Zsh completion for bt
_bt_completion() {
    local -a commands repo_commands shared_commands config_commands worktrees repos

    commands=(
        'add:Create a worktree for a branch'
        'list:List all worktrees'
        'ls:List all worktrees'
        'remove:Remove a worktree'
        'rm:Remove a worktree'
        'cd:Change to worktree directory'
        'status:Show repository status'
        'rename:Rename a worktree'
        'repair:Repair unmanaged worktrees into baretree structure'
        'root:Show repository root directory'
        'unbare:Convert worktree to standalone repository'
        'config:Manage repository configuration'
        'repo:Repository management'
        'shared:Shared files management'
        'shell-init:Generate shell integration code'
        'version:Print version information'
        'init:Create a new baretree repository (alias)'
        'clone:Clone with baretree structure (alias)'
        'migrate:Convert existing repository (alias)'
        'get:Clone into root directory (alias)'
        'go:Change to repository directory (alias)'
        'repos:List all repositories (alias)'
    )

    repo_commands=(
        'init:Create a new baretree repository'
        'clone:Clone a repository with baretree structure'
        'migrate:Convert existing repository to baretree'
        'list:List all repositories'
        'root:Show root directory path'
        'get:Clone into root directory (ghq-style)'
        'cd:Change to repository directory'
        'config:Manage global configuration'
    )

    shared_commands=(
        'add:Add shared file configuration'
        'remove:Remove shared file configuration'
        'apply:Apply shared configuration'
        'list:List shared files'
    )

    config_commands=(
        'export:Export configuration to TOML'
        'import:Import configuration from TOML'
    )

    if (( CURRENT == 2 )); then
        _describe 'command' commands
    elif (( CURRENT == 3 )); then
        case ${words[2]} in
            cd|remove|rm)
                # Complete worktree names
                worktrees=(${(f)"$(command bt list --paths 2>/dev/null | xargs -n1 basename)"})
                worktrees+=('@' '-')
                _describe 'worktree' worktrees
                ;;
            go)
                # Complete repository names for go command (alias for repo cd)
                repos=(${(f)"$(command bt repo list 2>/dev/null | xargs -n1 basename)"})
                repos+=('-')
                _describe 'repository' repos
                ;;
            repos)
                # Complete repository names for repos command (alias for repo list)
                repos=(${(f)"$(command bt repo list 2>/dev/null | xargs -n1 basename)"})
                _describe 'repository' repos
                ;;
            repo)
                _describe 'repo command' repo_commands
                ;;
            shared)
                _describe 'shared command' shared_commands
                ;;
            config)
                _describe 'config command' config_commands
                ;;
        esac
    elif (( CURRENT == 4 )); then
        if [[ ${words[2]} == "repo" && ${words[3]} == "cd" ]]; then
            # Complete repository names for repo cd
            repos=(${(f)"$(command bt repo list 2>/dev/null | xargs -n1 basename)"})
            repos+=('-')
            _describe 'repository' repos
        elif [[ ${words[2]} == "repo" && ${words[3]} == "config" ]]; then
            _describe 'config command' config_commands
        fi
    fi
}

compdef _bt_completion bt
`
