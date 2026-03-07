# smart-cd bash integration
# Usage: eval "$(scd --init bash)"

function cd() {
    # No arguments: go home
    if [ $# -eq 0 ]; then
        builtin cd "$HOME"
        return $?
    fi

    # Capture scd output; UI and errors go to the terminal via /dev/tty
    local target
    target=$(scd "$@" 2>/dev/tty)
    local exit_code=$?

    if [ $exit_code -eq 0 ] && [ -n "$target" ]; then
        if builtin cd "$target"; then
            scd --record "$target" &>/dev/null &
        fi
    fi
    return $exit_code
}

# Tab completion for bookmark names
_scd_completion() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    if [[ "$cur" == @* ]]; then
        local names
        names=$(scd --list-bookmarks 2>/dev/null)
        COMPREPLY=($(compgen -W "$names" -- "$cur"))
    else
        # Fall back to directory completion
        COMPREPLY=($(compgen -d -- "$cur"))
    fi
}
complete -F _scd_completion cd
