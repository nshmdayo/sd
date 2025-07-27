#!/bin/bash

# zcd shell integration script
# Source this file in your shell configuration (.bashrc, .zshrc, etc.)

# Function to replace the built-in cd command
function cd() {
    # Store the current directory as OLDPWD before changing
    local current_dir="$(pwd)"
    
    # If zcd binary exists and is executable, use it
    if command -v zcd >/dev/null 2>&1; then
        # Call zcd and capture its output
        local zcd_output
        if zcd_output=$(zcd "$@" 2>&1); then
            # If zcd succeeded, change to the current directory (zcd already changed it)
            # We need to sync the shell's working directory with the one zcd set
            builtin cd "$(pwd)" 2>/dev/null || builtin cd "$current_dir"
            
            # Only show output if it's not just the "Changed to:" message
            if [[ "$zcd_output" != "Changed to:"* ]]; then
                echo "$zcd_output"
            fi
        else
            # If zcd failed, show the error and don't change directories
            echo "$zcd_output" >&2
            return 1
        fi
    else
        # Fallback to built-in cd if zcd is not available
        builtin cd "$@"
    fi
    
    # Update OLDPWD for shell compatibility
    export OLDPWD="$current_dir"
}

# Alias for quick access to zcd features
alias zcd-history='zcd --history'
alias zcd-bookmarks='zcd --bookmarks'
alias zcd-add='zcd --add'
alias zcd-remove='zcd --remove'

# Auto-completion function for zcd
function _zcd_completion() {
    local current_word="${COMP_WORDS[COMP_CWORD]}"
    local suggestions
    
    # Get directory suggestions from zcd (if available)
    if command -v zcd >/dev/null 2>&1; then
        # This is a simplified completion - you could enhance it further
        suggestions=$(find . -maxdepth 1 -type d -name "*${current_word}*" 2>/dev/null | sed 's|^\./||')
        COMPREPLY=($(compgen -W "$suggestions" -- "$current_word"))
    fi
}

# Register completion function for bash
if [[ -n "$BASH_VERSION" ]]; then
    complete -F _zcd_completion cd
fi

# For zsh users, add this to enable completion
if [[ -n "$ZSH_VERSION" ]]; then
    autoload -U compinit
    compinit
    
    # Simple zsh completion
    _zcd() {
        local context state line
        _arguments \
            '1:directory:_directories' \
            '*::arguments:_directories'
    }
    
    compdef _zcd cd
fi

echo "zcd shell integration loaded successfully!"
echo "Usage: cd [directory] - now uses zcd for smart navigation"
echo "Additional commands:"
echo "  zcd-history     - show directory history"
echo "  zcd-bookmarks   - show bookmarks"
echo "  zcd-add [path]  - add bookmark"
echo "  zcd-remove path - remove bookmark"
