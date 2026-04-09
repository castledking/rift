package editor

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"rift/internal/clipboard"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StatusMsg is sent to update the status bar
type StatusMsg string

// Buffer holds text content with undo/redo support
type Buffer struct {
	lines     []string
	undoStack []BufferState
	redoStack []BufferState
	modified  bool
}

// BufferState represents a snapshot for undo/redo
type BufferState struct {
	lines []string
}

// Cursor represents the cursor position
type Cursor struct {
	line int
	col  int
}

// Selection represents a text selection
type Selection struct {
	startLine int
	startCol  int
	endLine   int
	endCol    int
	active    bool
}

// Viewport tracks scrolling state
type Viewport struct {
	topLine int
	leftCol int
}

// Model is the editor state
type Model struct {
	width           int
	height          int
	buffer          *Buffer
	cursor          Cursor
	selection       Selection
	viewport        Viewport
	filepath        string
	clipboard       *clipboard.Clipboard
	showLineNumbers bool
	lexer           chroma.Lexer
	syntaxEnabled   bool
}

// New creates a new editor model
func New() *Model {
	return &Model{
		buffer: &Buffer{
			lines: []string{""},
		},
		cursor:          Cursor{line: 0, col: 0},
		viewport:        Viewport{topLine: 0, leftCol: 0},
		clipboard:       clipboard.New(),
		showLineNumbers: true,
		syntaxEnabled:   true,
	}
}

// Init initializes the editor
func (m *Model) Init() tea.Cmd {
	return nil
}

// SetSize updates the editor dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Reset viewport if it's out of bounds after resize
	if m.viewport.topLine >= len(m.buffer.lines) {
		m.viewport.topLine = len(m.buffer.lines) - 1
		if m.viewport.topLine < 0 {
			m.viewport.topLine = 0
		}
	}
}

// detectLexer returns the appropriate lexer for the current file
func (m *Model) detectLexer() chroma.Lexer {
	if m.filepath == "" {
		return nil
	}

	ext := filepath.Ext(m.filepath)
	lexer := lexers.Match(m.filepath)
	if lexer == nil {
		lexer = lexers.Get(ext)
	}
	return lexer
}

// highlightLine applies syntax highlighting to a line using chroma
func (m *Model) highlightLine(line string, lineNum int) string {
	if !m.syntaxEnabled || m.lexer == nil {
		return line
	}

	// Tokenize the line
	iterator, err := m.lexer.Tokenise(nil, line)
	if err != nil {
		return line
	}

	// Use a simple style mapping
	style := styles.Get("monokai")
	if style == nil {
		return line
	}

	// Build the highlighted line
	var result strings.Builder
	for _, token := range iterator.Tokens() {
		// Get chroma style for this token type
		chromaStyle := style.Get(token.Type)
		if !chromaStyle.IsZero() {
			// Convert chroma color to lipgloss color
			var fg lipgloss.Color
			if chromaStyle.Colour != 0 {
				fg = lipgloss.Color(chromaStyle.Colour.String())
			}

			// Apply style if we have a foreground color
			if fg != "" {
				style := lipgloss.NewStyle().Foreground(fg)
				result.WriteString(style.Render(token.Value))
			} else {
				result.WriteString(token.Value)
			}
		} else {
			result.WriteString(token.Value)
		}
	}

	return result.String()
}

// OpenFile loads a file into the editor
func (m *Model) OpenFile(path string) tea.Cmd {
	return func() tea.Msg {
		content, err := os.ReadFile(path)
		if err != nil {
			return StatusMsg("Error opening file: " + err.Error())
		}

		lines := []string{}
		scanner := bufio.NewScanner(strings.NewReader(string(content)))
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return StatusMsg("Error reading file: " + err.Error())
		}
		if len(lines) == 0 {
			lines = []string{""}
		}

		m.filepath = path
		m.lexer = m.detectLexer()
		m.buffer = &Buffer{
			lines:    lines,
			modified: false,
		}
		m.cursor = Cursor{line: 0, col: 0}
		m.viewport = Viewport{topLine: 0, leftCol: 0}
		m.selection = Selection{}

		return StatusMsg(fmt.Sprintf("File loaded: %s (%d lines)", path, len(lines)))
	}
}

// Update processes messages
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.MouseMsg:
		return m.handleMouseEvent(msg)
	}

	return m, nil
}

// handleKeyPress handles keyboard input
func (m *Model) handleKeyPress(msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyRunes:
		m.insertText(string(msg.Runes))

	case tea.KeySpace:
		m.insertText(" ")

	case tea.KeyEnter:
		// Clear selection when pressing enter
		if m.selection.active {
			m.selection.active = false
		}
		m.insertText("\n")

	case tea.KeyBackspace:
		m.backspace()

	case tea.KeyDelete:
		m.delete()

	case tea.KeyTab:
		m.insertText("    ") // 4 spaces for tab

	case tea.KeyUp:
		m.moveCursor(-1, 0)

	case tea.KeyDown:
		m.moveCursor(1, 0)

	case tea.KeyLeft:
		m.moveCursor(0, -1)

	case tea.KeyRight:
		m.moveCursor(0, 1)

	case tea.KeyHome:
		m.cursor.col = 0

	case tea.KeyEnd:
		if m.cursor.line < len(m.buffer.lines) {
			m.cursor.col = len(m.buffer.lines[m.cursor.line])
		}

	case tea.KeyPgUp:
		m.moveCursor(-m.height, 0)

	case tea.KeyPgDown:
		m.moveCursor(m.height, 0)
	}

	m.adjustViewport()
	return m, nil
}

// handleMouseEvent handles mouse input
func (m *Model) handleMouseEvent(msg tea.MouseMsg) (*Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		// Scroll up
		if m.viewport.topLine > 0 {
			m.viewport.topLine--
		}
		if m.cursor.line > m.viewport.topLine+m.height-1 {
			m.cursor.line = m.viewport.topLine + m.height - 1
		}
		return m, nil
	case tea.MouseButtonWheelDown:
		// Scroll down
		if m.viewport.topLine < len(m.buffer.lines)-m.height {
			m.viewport.topLine++
		}
		if m.cursor.line < m.viewport.topLine {
			m.cursor.line = m.viewport.topLine
		}
		return m, nil
	}

	// Convert screen coordinates to buffer coordinates
	line := msg.Y + m.viewport.topLine
	col := msg.X + m.viewport.leftCol

	// Clamp to valid buffer positions
	if line < 0 {
		line = 0
	}
	if line >= len(m.buffer.lines) {
		line = len(m.buffer.lines) - 1
		if line < 0 {
			line = 0
		}
	}
	lineLen := len(m.buffer.lines[line])
	if col > lineLen {
		col = lineLen
	}
	if col < 0 {
		col = 0
	}

	switch msg.Action {
	case tea.MouseActionPress:
		// Start new selection at cursor position
		m.cursor.line = line
		m.cursor.col = col
		m.selection.startLine = line
		m.selection.startCol = col
		m.selection.endLine = line
		m.selection.endCol = col
		m.selection.active = true

	case tea.MouseActionMotion:
		// Extend selection while dragging
		if m.selection.active {
			m.cursor.line = line
			m.cursor.col = col
			m.selection.endLine = line
			m.selection.endCol = col
		}

	case tea.MouseActionRelease:
		// Finalize selection
		if m.selection.active {
			m.cursor.line = line
			m.cursor.col = col
			m.selection.endLine = line
			m.selection.endCol = col
			// Keep selection active for copy operations
		}
	}

	return m, nil
}

// View renders the editor
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	// Calculate line number width
	lineNumWidth := 0
	if m.showLineNumbers {
		// Calculate width based on total lines (e.g., 3 digits for 100+ lines)
		lineNumWidth = len(fmt.Sprintf("%d", len(m.buffer.lines))) + 1
		if lineNumWidth < 4 {
			lineNumWidth = 4 // Minimum width
		}
	}

	contentWidth := m.width - lineNumWidth

	// Ensure viewport is valid
	if m.viewport.topLine < 0 {
		m.viewport.topLine = 0
	}
	if m.viewport.topLine >= len(m.buffer.lines) {
		m.viewport.topLine = len(m.buffer.lines) - 1
		if m.viewport.topLine < 0 {
			m.viewport.topLine = 0
		}
	}

	lineNumStyle := lipgloss.NewStyle().
		Width(lineNumWidth).
		Background(lipgloss.Color("#1e1e1e")).
		Foreground(lipgloss.Color("#858585"))

	selectionStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#264f78"))

	cursorStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#aeafad")).
		Foreground(lipgloss.Color("#1e1e1e"))

	// Get normalized selection range
	selStartLine, selStartCol, selEndLine, selEndCol := m.getSelectionRange()

	endLine := m.viewport.topLine + m.height
	if endLine > len(m.buffer.lines) {
		endLine = len(m.buffer.lines)
	}

	var resultLines []string
	for i := m.viewport.topLine; i < endLine; i++ {
		// Safety check: ensure index is valid
		if i < 0 || i >= len(m.buffer.lines) {
			continue
		}
		lineContent := m.buffer.lines[i]

		// Handle horizontal scrolling
		if len(lineContent) < m.viewport.leftCol {
			lineContent = ""
		} else {
			lineContent = lineContent[m.viewport.leftCol:]
		}
		if len(lineContent) > contentWidth {
			lineContent = lineContent[:contentWidth]
		}

		// Apply syntax highlighting
		lineContent = m.highlightLine(lineContent, i)

		// Check if this line has selection
		hasSelection := m.selection.active && i >= selStartLine && i <= selEndLine

		if hasSelection {
			// Calculate selection portion on this line
			var selStart, selEnd int
			if i == selStartLine && i == selEndLine {
				// Single line selection
				selStart = selStartCol - m.viewport.leftCol
				selEnd = selEndCol - m.viewport.leftCol
			} else if i == selStartLine {
				// First line of multi-line
				selStart = selStartCol - m.viewport.leftCol
				selEnd = len(lineContent)
			} else if i == selEndLine {
				// Last line of multi-line
				selStart = 0
				selEnd = selEndCol - m.viewport.leftCol
			} else {
				// Middle line (fully selected)
				selStart = 0
				selEnd = len(lineContent)
			}

			// Clamp to visible range
			if selStart < 0 {
				selStart = 0
			}
			if selEnd > len(lineContent) {
				selEnd = len(lineContent)
			}

			// Apply selection styling
			if selStart < selEnd && selStart < len(lineContent) {
				before := lineContent[:selStart]
				selected := lineContent[selStart:selEnd]
				after := lineContent[selEnd:]
				lineContent = before + selectionStyle.Render(selected) + after
			}
		}

		// Add line number FIRST (before cursor/selection styling)
		if m.showLineNumbers {
			lineNum := fmt.Sprintf("%*d ", lineNumWidth-1, i+1)
			lineContent = lineNumStyle.Render(lineNum) + lineContent
		}

		// Render cursor (on top of selection)
		if i == m.cursor.line && m.cursor.col >= m.viewport.leftCol && m.cursor.col < m.viewport.leftCol+contentWidth {
			cursorCol := m.cursor.col - m.viewport.leftCol
			// Get the raw line without line number prefix for cursor calculation
			rawLine := m.buffer.lines[i]
			if len(rawLine) < m.viewport.leftCol {
				rawLine = ""
			} else {
				rawLine = rawLine[m.viewport.leftCol:]
			}
			if len(rawLine) > contentWidth {
				rawLine = rawLine[:contentWidth]
			}

			// Safety check: clamp cursorCol to valid range
			if cursorCol < 0 {
				cursorCol = 0
			}
			if cursorCol > len(rawLine) {
				cursorCol = len(rawLine)
			}

			before := ""
			if cursorCol > 0 && cursorCol <= len(rawLine) {
				before = rawLine[:cursorCol]
			}
			char := " "
			if cursorCol < len(rawLine) {
				char = string(rawLine[cursorCol])
			}
			after := ""
			if cursorCol+1 < len(rawLine) {
				after = rawLine[cursorCol+1:]
			}

			// Rebuild line content with cursor styling
			styledLine := before + cursorStyle.Render(char) + after

			// Add line number prefix back if enabled
			if m.showLineNumbers {
				lineNum := fmt.Sprintf("%*d ", lineNumWidth-1, i+1)
				lineContent = lineNumStyle.Render(lineNum) + styledLine
			} else {
				lineContent = styledLine
			}
		}

		resultLines = append(resultLines, lineContent)
	}

	// Fill empty lines
	for len(resultLines) < m.height {
		emptyLine := ""
		if m.showLineNumbers {
			emptyLine = lineNumStyle.Render(strings.Repeat(" ", lineNumWidth))
		}
		resultLines = append(resultLines, emptyLine)
	}

	// Wrap with background color to ensure proper layering
	editorStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Background(lipgloss.Color("#1e1e1e"))

	return editorStyle.Render(strings.Join(resultLines, "\n"))
}

// getSelectionRange returns normalized selection coordinates
func (m *Model) getSelectionRange() (startLine, startCol, endLine, endCol int) {
	if !m.selection.active {
		return 0, 0, 0, 0
	}

	startLine = m.selection.startLine
	startCol = m.selection.startCol
	endLine = m.selection.endLine
	endCol = m.selection.endCol

	// Normalize selection (start before end)
	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}

	return startLine, startCol, endLine, endCol
}

// insertText inserts text at cursor position
func (m *Model) insertText(text string) {
	m.saveUndoState()

	// If there's an active selection, delete it first
	if m.selection.active {
		m.deleteSelection()
	}

	line := m.buffer.lines[m.cursor.line]
	before := line[:m.cursor.col]
	after := line[m.cursor.col:]

	if text == "\n" {
		m.buffer.lines[m.cursor.line] = before
		m.buffer.lines = append(m.buffer.lines[:m.cursor.line+1], append([]string{after}, m.buffer.lines[m.cursor.line+1:]...)...)
		m.cursor.line++
		m.cursor.col = 0
	} else if strings.Contains(text, "\n") {
		// Multi-line paste
		lines := strings.Split(text, "\n")
		m.buffer.lines[m.cursor.line] = before + lines[0]

		// Insert middle lines
		for i := 1; i < len(lines)-1; i++ {
			m.cursor.line++
			m.buffer.lines = append(m.buffer.lines[:m.cursor.line], append([]string{lines[i]}, m.buffer.lines[m.cursor.line:]...)...)
		}

		// Last line
		if len(lines) > 1 {
			m.cursor.line++
			m.buffer.lines = append(m.buffer.lines[:m.cursor.line], append([]string{lines[len(lines)-1] + after}, m.buffer.lines[m.cursor.line:]...)...)
		}
		m.cursor.col = len(lines[len(lines)-1])
	} else {
		m.buffer.lines[m.cursor.line] = before + text + after
		m.cursor.col += len(text)
	}
	m.buffer.modified = true
}

// backspace deletes character before cursor or selected text
func (m *Model) backspace() {
	// If selection is active, delete it
	if m.selection.active {
		m.deleteSelection()
		return
	}

	if m.cursor.col > 0 {
		m.saveUndoState()
		line := m.buffer.lines[m.cursor.line]
		m.buffer.lines[m.cursor.line] = line[:m.cursor.col-1] + line[m.cursor.col:]
		m.cursor.col--
		m.buffer.modified = true
	} else if m.cursor.line > 0 {
		m.saveUndoState()
		// Join with previous line
		prevLine := m.buffer.lines[m.cursor.line-1]
		currLine := m.buffer.lines[m.cursor.line]
		m.cursor.col = len(prevLine)
		m.buffer.lines[m.cursor.line-1] = prevLine + currLine
		m.buffer.lines = append(m.buffer.lines[:m.cursor.line], m.buffer.lines[m.cursor.line+1:]...)
		m.cursor.line--
		m.buffer.modified = true
	}
}

// delete deletes character after cursor
func (m *Model) delete() {
	line := m.buffer.lines[m.cursor.line]
	if m.cursor.col < len(line) {
		m.saveUndoState()
		m.buffer.lines[m.cursor.line] = line[:m.cursor.col] + line[m.cursor.col+1:]
		m.buffer.modified = true
	} else if m.cursor.line < len(m.buffer.lines)-1 {
		m.saveUndoState()
		// Join with next line
		nextLine := m.buffer.lines[m.cursor.line+1]
		m.buffer.lines[m.cursor.line] = line + nextLine
		m.buffer.lines = append(m.buffer.lines[:m.cursor.line+1], m.buffer.lines[m.cursor.line+2:]...)
		m.buffer.modified = true
	}
}

// moveCursor moves the cursor by the given delta
func (m *Model) moveCursor(dLine, dCol int) {
	newLine := m.cursor.line + dLine
	newCol := m.cursor.col + dCol

	if newLine < 0 {
		newLine = 0
	}
	if newLine >= len(m.buffer.lines) {
		newLine = len(m.buffer.lines) - 1
	}

	lineLen := len(m.buffer.lines[newLine])
	if newCol < 0 {
		newCol = 0
	}
	if newCol > lineLen {
		newCol = lineLen
	}

	m.cursor.line = newLine
	m.cursor.col = newCol
}

// adjustViewport scrolls to keep cursor visible
func (m *Model) adjustViewport() {
	// Vertical scrolling
	if m.cursor.line < m.viewport.topLine {
		m.viewport.topLine = m.cursor.line
	}
	if m.cursor.line >= m.viewport.topLine+m.height {
		m.viewport.topLine = m.cursor.line - m.height + 1
	}

	// Horizontal scrolling
	if m.cursor.col < m.viewport.leftCol {
		m.viewport.leftCol = m.cursor.col
	}
	if m.cursor.col >= m.viewport.leftCol+m.width {
		m.viewport.leftCol = m.cursor.col - m.width + 1
	}
}

// saveUndoState saves current state for undo
func (m *Model) saveUndoState() {
	state := BufferState{lines: make([]string, len(m.buffer.lines))}
	copy(state.lines, m.buffer.lines)
	m.buffer.undoStack = append(m.buffer.undoStack, state)
	// Clear redo stack on new edit
	m.buffer.redoStack = nil
}

// Undo reverts the last change
func (m *Model) Undo() {
	if len(m.buffer.undoStack) > 0 {
		// Save current state to redo
		state := BufferState{lines: make([]string, len(m.buffer.lines))}
		copy(state.lines, m.buffer.lines)
		m.buffer.redoStack = append(m.buffer.redoStack, state)

		// Pop and restore
		lastIdx := len(m.buffer.undoStack) - 1
		state = m.buffer.undoStack[lastIdx]
		m.buffer.undoStack = m.buffer.undoStack[:lastIdx]
		m.buffer.lines = state.lines
	}
}

// Redo reapplies a previously undone change
func (m *Model) Redo() {
	if len(m.buffer.redoStack) > 0 {
		// Save current state to undo
		state := BufferState{lines: make([]string, len(m.buffer.lines))}
		copy(state.lines, m.buffer.lines)
		m.buffer.undoStack = append(m.buffer.undoStack, state)

		// Pop and restore
		lastIdx := len(m.buffer.redoStack) - 1
		state = m.buffer.redoStack[lastIdx]
		m.buffer.redoStack = m.buffer.redoStack[:lastIdx]
		m.buffer.lines = state.lines
	}
}

// GetFilePath returns the current file path
func (m *Model) GetFilePath() string {
	return m.filepath
}

// SaveAs saves the buffer to a new file path
func (m *Model) SaveAs(path string) tea.Cmd {
	return func() tea.Msg {
		content := strings.Join(m.buffer.lines, "\n")
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			return StatusMsg("Error saving: " + err.Error())
		}

		m.filepath = path
		m.buffer.modified = false
		return StatusMsg("Saved: " + path)
	}
}
func (m *Model) Save() tea.Cmd {
	return func() tea.Msg {
		if m.filepath == "" {
			return StatusMsg("No file to save")
		}

		content := strings.Join(m.buffer.lines, "\n")
		err := os.WriteFile(m.filepath, []byte(content), 0644)
		if err != nil {
			return StatusMsg("Error saving: " + err.Error())
		}

		m.buffer.modified = false
		return StatusMsg("Saved: " + m.filepath)
	}
}

// SaveWithAdmin saves the file using sudo (for editing protected files)
func (m *Model) SaveWithAdmin() tea.Cmd {
	return func() tea.Msg {
		if m.filepath == "" {
			return StatusMsg("No file to save")
		}

		content := strings.Join(m.buffer.lines, "\n")

		// Try pkexec first, then fall back to sudo
		var cmd *exec.Cmd
		if _, err := exec.LookPath("pkexec"); err == nil {
			cmd = exec.Command("pkexec", "tee", m.filepath)
		} else {
			cmd = exec.Command("sudo", "tee", m.filepath)
		}

		cmd.Stdin = strings.NewReader(content)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return StatusMsg("Error saving with admin: " + err.Error() + " - " + string(output))
		}

		m.buffer.modified = false
		return StatusMsg("Saved (admin): " + m.filepath)
	}
}

// Copy copies the current line or selection to clipboard
func (m *Model) Copy() {
	var text string
	if m.selection.active {
		text = m.getSelectedText()
	} else {
		// Copy current line
		if m.cursor.line < len(m.buffer.lines) {
			text = m.buffer.lines[m.cursor.line]
		}
	}
	m.clipboard.Copy(text)
}

// Cut cuts the current line or selection to clipboard
func (m *Model) Cut() {
	m.Copy()
	if m.selection.active {
		m.deleteSelection()
	} else {
		// Cut current line
		m.saveUndoState()
		m.buffer.lines = append(m.buffer.lines[:m.cursor.line], m.buffer.lines[m.cursor.line+1:]...)
		if len(m.buffer.lines) == 0 {
			m.buffer.lines = []string{""}
		}
		if m.cursor.line >= len(m.buffer.lines) {
			m.cursor.line = len(m.buffer.lines) - 1
		}
		m.cursor.col = 0
		m.buffer.modified = true
	}
}

// ToggleLineNumbers toggles line number display
func (m *Model) ToggleLineNumbers() {
	m.showLineNumbers = !m.showLineNumbers
}

// Paste inserts clipboard content at cursor
func (m *Model) Paste() {
	text := m.clipboard.Paste()
	if text == "" {
		return
	}
	m.saveUndoState()
	m.insertText(text)
}

// SelectAll selects all text in the buffer
func (m *Model) SelectAll() {
	if len(m.buffer.lines) == 0 {
		return
	}
	m.selection.active = true
	m.selection.startLine = 0
	m.selection.startCol = 0
	m.selection.endLine = len(m.buffer.lines) - 1
	m.selection.endCol = len(m.buffer.lines[len(m.buffer.lines)-1])
	m.cursor.line = m.selection.endLine
	m.cursor.col = m.selection.endCol
}

// getSelectedText returns the currently selected text
func (m *Model) getSelectedText() string {
	if !m.selection.active {
		return ""
	}

	var parts []string
	startLine, startCol := m.selection.startLine, m.selection.startCol
	endLine, endCol := m.selection.endLine, m.selection.endCol

	// Normalize selection
	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}

	for i := startLine; i <= endLine && i < len(m.buffer.lines); i++ {
		line := m.buffer.lines[i]
		if i == startLine && i == endLine {
			// Single line selection
			if startCol < len(line) {
				end := endCol
				if end > len(line) {
					end = len(line)
				}
				parts = append(parts, line[startCol:end])
			}
		} else if i == startLine {
			// First line
			if startCol < len(line) {
				parts = append(parts, line[startCol:])
			}
		} else if i == endLine {
			// Last line
			end := endCol
			if end > len(line) {
				end = len(line)
			}
			parts = append(parts, line[:end])
		} else {
			// Middle line
			parts = append(parts, line)
		}
	}

	return strings.Join(parts, "\n")
}

// deleteSelection deletes the selected text
func (m *Model) deleteSelection() {
	if !m.selection.active {
		return
	}

	m.saveUndoState()
	startLine, startCol := m.selection.startLine, m.selection.startCol
	endLine, endCol := m.selection.endLine, m.selection.endCol

	// Normalize selection
	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}

	if startLine == endLine {
		// Single line
		line := m.buffer.lines[startLine]
		before := line[:startCol]
		after := ""
		if endCol < len(line) {
			after = line[endCol:]
		}
		m.buffer.lines[startLine] = before + after
	} else {
		// Multi-line
		firstLine := m.buffer.lines[startLine]
		lastLine := m.buffer.lines[endLine]

		before := firstLine[:startCol]
		after := ""
		if endCol < len(lastLine) {
			after = lastLine[endCol:]
		}

		m.buffer.lines[startLine] = before + after
		m.buffer.lines = append(m.buffer.lines[:startLine+1], m.buffer.lines[endLine+1:]...)
	}

	m.cursor.line = startLine
	m.cursor.col = startCol
	m.selection.active = false
	m.buffer.modified = true
}
