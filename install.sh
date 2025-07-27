#!/bin/bash

# zcd installation script

set -e

BINARY_NAME="zcd"
INSTALL_DIR="/usr/local/bin"
REPO_URL="https://github.com/nshmdayo/zcd"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go first: https://golang.org/dl/"
        exit 1
    fi
    print_status "Go found: $(go version)"
}

# Check if we have write permissions to install directory
check_permissions() {
    if [ ! -w "$INSTALL_DIR" ]; then
        print_error "No write permission to $INSTALL_DIR"
        print_status "Please run with sudo or choose a different install directory"
        exit 1
    fi
}

# Build and install zcd
install_zcd() {
    local temp_dir=$(mktemp -d)
    
    print_status "Downloading zcd source code..."
    
    if command -v git &> /dev/null; then
        git clone "$REPO_URL" "$temp_dir"
    else
        print_error "Git is not installed. Please install git first."
        exit 1
    fi
    
    cd "$temp_dir"
    
    print_status "Building zcd..."
    go build -ldflags "-s -w" -o "$BINARY_NAME" .
    
    print_status "Installing zcd to $INSTALL_DIR..."
    cp "$BINARY_NAME" "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
    
    print_status "Copying shell integration script..."
    cp shell_integration.sh "$INSTALL_DIR/zcd_shell_integration.sh"
    
    # Cleanup
    cd /
    rm -rf "$temp_dir"
    
    print_status "zcd installed successfully!"
}

# Setup shell integration
setup_shell_integration() {
    local shell_config=""
    local shell_name=$(basename "$SHELL")
    
    case "$shell_name" in
        "bash")
            shell_config="$HOME/.bashrc"
            ;;
        "zsh")
            shell_config="$HOME/.zshrc"
            ;;
        "fish")
            shell_config="$HOME/.config/fish/config.fish"
            ;;
        *)
            print_warning "Unknown shell: $shell_name"
            print_status "Please manually add the following line to your shell config:"
            print_status "source $INSTALL_DIR/zcd_shell_integration.sh"
            return
            ;;
    esac
    
    if [ -f "$shell_config" ]; then
        print_status "Adding shell integration to $shell_config..."
        
        # Check if already added
        if grep -q "zcd_shell_integration.sh" "$shell_config"; then
            print_warning "Shell integration already exists in $shell_config"
        else
            echo "" >> "$shell_config"
            echo "# zcd shell integration" >> "$shell_config"
            echo "source $INSTALL_DIR/zcd_shell_integration.sh" >> "$shell_config"
            print_status "Shell integration added to $shell_config"
        fi
    else
        print_warning "$shell_config not found"
        print_status "Please manually add the following line to your shell config:"
        print_status "source $INSTALL_DIR/zcd_shell_integration.sh"
    fi
}

# Show post-installation instructions
show_instructions() {
    echo ""
    print_status "Installation complete!"
    echo ""
    echo "To start using zcd:"
    echo "1. Restart your shell or run: source ~/.bashrc (or ~/.zshrc)"
    echo "2. Use 'cd' command as usual - it now has smart features!"
    echo ""
    echo "Usage examples:"
    echo "  cd project      # Smart search for directories containing 'project'"
    echo "  cd --add        # Bookmark current directory"
    echo "  cd --history    # Show navigation history"
    echo "  cd --bookmarks  # Show bookmarked directories"
    echo ""
    echo "For more information, run: cd --help"
}

# Main installation function
main() {
    print_status "Installing zcd - Smart Directory Navigation"
    echo ""
    
    check_go
    check_permissions
    install_zcd
    setup_shell_integration
    show_instructions
}

# Run installation
main "$@"
