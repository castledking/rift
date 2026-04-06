# Rift Editor

A terminal-native code editor with a diff-first workflow, designed for SSH environments.

## Overview

Rift is NOT modal (not like Vim). It behaves more like VS Code:
- Direct typing (no modes to switch)
- Mouse support
- Standard shortcuts (Ctrl+C, Ctrl+V, etc.)

The defining feature: All automated or AI-driven edits are proposed as diffs and must be reviewed and confirmed before applying.

## Features

### MVP Features (Implemented)

1. **Layout System**
   - Left panel: collapsible file explorer
   - Center: main editor view
   - Bottom: status bar with shortcuts
   - Support for resizing and focus switching
   - Toggle explorer with Ctrl+B

2. **File Explorer**
   - Tree view of current working directory
   - Expand/collapse directories
   - Click to open file in editor
   - Keyboard navigation (up/down, enter to expand/open)

3. **Text Editor (Core)**
   - Open, edit, save files
   - Cursor movement (arrows, word jump with Home/End)
   - Mouse click to place cursor
   - Basic scrolling (PgUp/PgDown)
   - Undo/redo stack (Ctrl+Z, Ctrl+Shift+Z)

4. **Keybindings (VS Code-like)**
   - `Ctrl+C` → copy current line
   - `Ctrl+V` → paste
   - `Ctrl+X` → cut current line
   - `Ctrl+Z` → undo
   - `Ctrl+Shift+Z` → redo
   - `Ctrl+S` → save
   - `Ctrl+B` → toggle explorer
   - `Ctrl+Q` → quit

5. **Clipboard Layer**
   - Internal clipboard always works
   - OSC52 support for system clipboard over SSH
   - Graceful degradation if OSC52 not supported

6. **Diff / Patch Review System (Core Feature)**
   - Proposed changes shown as inline diffs
   - Review mode with key controls:
     - `y` or `Enter` → accept hunk
     - `n` or `Backspace` → reject hunk
     - `a` → accept all remaining
     - `r` → reject all remaining
     - `q` or `Esc` → exit review mode
     - `Tab` → next hunk
     - `Shift+Tab` → previous hunk

## Architecture

```
/cmd/rift          → Entry point
/internal/app      → Main app state & layout management
/internal/editor   → Text buffer, cursor, selection, clipboard
/internal/explorer  → File tree navigation
/internal/diff     → Diff model, hunk system, and review UI
/internal/clipboard → Clipboard abstraction with OSC52
/internal/keymap   → Keybindings
/internal/ui       → (placeholder for reusable components)
```

Design principles:
- No global state
- Clear separation of concerns
- All UI state flows through Bubble Tea model/update/view

## Building and Running

```bash
# Build
go build ./cmd/rift

# Run
./rift

# Or run directly
go run ./cmd/rift
```

## Usage

1. Start Rift in any directory - it will show the file explorer on the left
2. Navigate with arrow keys or click with mouse
3. Press Enter or click to open a file
4. Edit directly (no modes)
5. Use Ctrl+S to save, Ctrl+Q to quit
6. Use Ctrl+B to toggle the file explorer

## Future Enhancements

- Find/Replace (Ctrl+F)
- Quick file open (Ctrl+P)
- Command palette (Ctrl+Shift+P)
- LSP integration
- AI integration with diff review
- Plugin system
- Syntax highlighting
- Line numbers
- Split panes
- Configuration file support

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling
- [go-osc52](https://github.com/aymanbagabas/go-osc52) - OSC52 clipboard integration

## License

MIT
