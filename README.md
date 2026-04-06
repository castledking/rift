# Rift Editor

A terminal-based text editor written in Go with a modern, intuitive interface. Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss).

![Rift Editor](https://img.shields.io/badge/Rift-Editor-blue)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)
![License](https://img.shields.io/badge/License-MIT-green)

## Features

- **File Explorer** - Navigate your project with an expandable tree view
- **Mouse Support** - Scroll with your mouse wheel in both explorer and editor
- **Keyboard Navigation** - VS Code-like shortcuts
- **Undo/Redo** - Full undo/redo history
- **New File / Open Folder** - Quick file creation and folder switching (Ctrl+N, Ctrl+O)
- **Command Line Support** - Open files and folders directly from the terminal
- **Clipboard Integration** - OSC52 support for system clipboard over SSH

## Quick Install

### Using the Setup Script (Recommended)

```bash
# Clone the repository
git clone https://github.com/castledking/rift.git
cd rift

# Build and install
./setup_rift.sh

# Reload your shell configuration
source ~/.bashrc

# Run Rift
rift
```

The setup script will:
- Build the binary from source (requires Go 1.21+)
- Install to `~/.local/bin`
- Add a convenient `rift` shell function to your `.bashrc` that properly handles file/folder arguments

### Manual Installation

```bash
# Clone the repository
git clone https://github.com/castledking/rift.git
cd rift

# Build the binary
go build -o rift ./cmd/rift

# Move to your preferred location
mv rift ~/.local/bin/
```

## Usage

### Opening Files and Folders

```bash
# Open current directory
rift

# Open specific folder
rift /path/to/project

# Open specific file
rift /path/to/file.txt
```

### Keybindings

| Key | Action |
|-----|--------|
| `Ctrl+S` | Save file |
| `Ctrl+Q` | Quit |
| `Ctrl+N` | New file |
| `Ctrl+O` | Open folder |
| `Ctrl+B` | Toggle file explorer |
| `Ctrl+F` | Find in file |
| `Ctrl+Z` | Undo |
| `Ctrl+Shift+Z` | Redo |
| `Ctrl+C` | Copy line/selection |
| `Ctrl+X` | Cut line/selection |
| `Ctrl+V` | Paste |
| `Ctrl+A` | Select all |
| `Ctrl+G` | Toggle line numbers |
| `F1` | Show help |

### File Explorer Navigation

- **↑/↓** - Navigate up/down
- **Enter** - Open file or expand/collapse folder
- **Scroll Wheel** - Scroll through the tree

### Editor Navigation

- **Arrow Keys** - Move cursor
- **Home/End** - Beginning/end of line
- **Page Up/Down** - Scroll viewport
- **Scroll Wheel** - Scroll through file

## Requirements

- Go 1.21 or later (for building from source)
- Terminal with mouse support (optional but recommended)

## Building from Source

```bash
# Clone the repository
git clone https://github.com/castledking/rift.git
cd rift

# Build
go build -o rift ./cmd/rift

# Run
./rift
```

## Project Structure

```
rift/
├── cmd/rift/          # Main entry point
├── internal/
│   ├── app/           # Application orchestration
│   ├── editor/        # Text editor implementation
│   ├── explorer/      # File tree explorer
│   ├── keymap/        # Keybinding management
│   ├── clipboard/     # Clipboard operations
│   └── diff/          # Diff/review functionality
├── go.mod
├── go.sum
├── setup_rift.sh      # Installation script
└── README.md
```

## License

MIT License - see LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.
