# zcd - Smart Directory Navigation

`zcd` is an intelligent `cd` command replacement written in Go that provides enhanced directory navigation features including history tracking, bookmarks, and smart suggestions.

## Features

- **Directory History**: Automatically tracks visited directories with usage frequency
- **Bookmarks**: Register frequently used directories as bookmarks for quick access
- **Smart Suggestions**: Intelligent directory suggestions based on partial input
- **Shell Integration**: Seamlessly replaces the built-in `cd` command
- **Cross-platform**: Works on Linux, macOS, and Windows

## Installation

### Build from source

```bash
git clone https://github.com/nshmdayo/zcd.git
cd zcd
make build
make install
```

### Enable Shell Integration

Add the following line to your shell configuration file (`.bashrc`, `.zshrc`, etc.):

```bash
source /path/to/zcd/shell_integration.sh
```

After sourcing the script, restart your shell or run:
```bash
source ~/.bashrc  # or ~/.zshrc
```

## Usage

### Basic Navigation

```bash
# Change to home directory
cd

# Change to a specific directory
cd /path/to/directory

# Go back to previous directory
cd -

# Smart search - finds directories matching "doc"
cd doc

# Use history number
cd 3
```

### Directory History

```bash
# Show directory history
cd --history
# or
zcd-history
```

### Bookmarks

```bash
# Add current directory to bookmarks
cd --add
# or
zcd-add

# Add specific directory to bookmarks
cd --add /path/to/directory
# or
zcd-add /path/to/directory

# Show bookmarks
cd --bookmarks
# or
zcd-bookmarks

# Remove bookmark
cd --remove /path/to/directory
# or
zcd-remove /path/to/directory
```

### Smart Suggestions

When you type a partial directory name, `zcd` will:

1. Look for exact matches first
2. Search through your history for directories containing the input
3. Prioritize bookmarked directories
4. Consider usage frequency and recency

If multiple matches are found, you'll be presented with a numbered list to choose from.

## Configuration

`zcd` stores its data in `~/.zcd_data.json`. The configuration includes:

- `max_history`: Maximum number of history entries to keep (default: 100)
- `max_bookmarks`: Maximum number of bookmarks to keep (default: 50)

## Examples

```bash
# Navigate to a project directory by partial name
cd myproj
# If multiple matches found:
# 1. /home/user/projects/myproject
# 2. /home/user/work/myproject-old
# Select number (1-2): 1

# Bookmark your current working directory
cd --add

# Show your navigation history
cd --history
# Directory History:
# ==================
#  1. /home/user/projects/myproject (used 15 times) [★]
#  2. /home/user/documents (used 8 times)
#  3. /home/user/downloads (used 3 times)

# Quick access using history numbers
cd 2  # Goes to /home/user/documents
```

## Data Storage

- History and bookmarks are stored in `~/.zcd_data.json`
- Data includes path, usage count, last access time, and bookmark status
- Automatic cleanup of old entries when limits are exceeded
- Non-existent directories are automatically filtered out

## Development

### Building

```bash
# Development build
make dev

# Production build
make build

# Build for multiple platforms
make build-all

# Run tests
make test

# Clean build artifacts
make clean
```

### Project Structure

```
zcd/
├── main.go                 # Main application logic
├── shell_integration.sh    # Shell integration script
├── Makefile               # Build configuration
├── go.mod                 # Go module definition
└── README.md              # This file
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

Inspired by tools like `z`, `autojump`, and `fasd`, but designed to be simple, fast, and written in Go for easy installation and cross-platform compatibility.