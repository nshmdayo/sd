# smart-cd zsh integration
# Usage: eval "$(scd --init zsh)"

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

# zsh completion for bookmark names
_scd_complete() {
    local state
    _arguments '*:: :->args'
    case $state in
        args)
            if [[ "${words[2]}" == @* ]]; then
                local -a bookmarks
                bookmarks=($(scd --list-bookmarks 2>/dev/null))
                compadd -P @ -- "${bookmarks[@]#@}"
            else
                _directories
            fi
            ;;
    esac
}
compdef _scd_complete cd
