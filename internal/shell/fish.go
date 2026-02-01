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

# Source Cobra-generated completion
if type -q bt
    command bt completion fish | source
end
`
