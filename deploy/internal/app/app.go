package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"rift/internal/diff"
	"rift/internal/editor"
	"rift/internal/explorer"
	"rift/internal/keymap"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// handleKeyPress processes global keybindings
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle save-as mode first
	if m.saveAsActive {
		return m.handleSaveAsInput(msg)
	}

	// Handle open-folder mode
	if m.openFolderActive {
		return m.handleOpenFolderInput(msg)
	}

	// Check if diff reviewer is active - it has priority
	if m.reviewer.IsActive() {
		var cmd tea.Cmd
		m.reviewer, cmd = m.reviewer.Update(msg)
		if !m.reviewer.IsActive() {
			// Reviewer was closed
			m.statusMsg = "Diff review completed"
		}
		return m, cmd
	}

	km := keymap.DefaultGlobalKeyMap()

	switch {
	case keymap.Matches(msg, km.Quit):
		return m, tea.Quit

	case keymap.Matches(msg, km.ToggleExplorer):
		m.showExplorer = !m.showExplorer
		m.updateChildSizes()
		if !m.showExplorer && m.focus == FocusExplorer {
			m.focus = FocusEditor
		}
		return m, nil

	case keymap.Matches(msg, km.Save):
		// If no file is open, enter save-as mode with explorer root as default
		if m.editor.GetFilePath() == "" {
			m.saveAsActive = true
			rootPath := m.explorer.GetRootPath()
			if rootPath == "" {
				rootPath, _ = os.Getwd()
			}
			m.saveAsPath = filepath.Join(rootPath, "untitled.rift")
			return m, nil
		}
		return m, m.editor.Save()

	case keymap.Matches(msg, km.Undo):
		m.editor.Undo()
		return m, nil

	case keymap.Matches(msg, km.Redo):
		m.editor.Redo()
		return m, nil

	case keymap.Matches(msg, km.Copy):
		m.editor.Copy()
		m.statusMsg = "Copied to clipboard"
		return m, nil

	case keymap.Matches(msg, km.Cut):
		m.editor.Cut()
		m.statusMsg = "Cut to clipboard"
		return m, nil

	case keymap.Matches(msg, km.Paste):
		m.editor.Paste()
		m.statusMsg = "Pasted from clipboard"
		return m, nil

	case keymap.Matches(msg, km.SelectAll):
		m.editor.SelectAll()
		m.statusMsg = "Selected all"
		return m, nil

	case keymap.Matches(msg, km.ToggleLineNumbers):
		m.editor.ToggleLineNumbers()
		m.statusMsg = "Toggled line numbers"
		return m, nil

	case keymap.Matches(msg, km.NewFile):
		// Create new empty file
		m.editor = editor.New()
		m.statusMsg = "New file created (Ctrl+S to save)"
		m.focus = FocusEditor
		return m, nil

	case keymap.Matches(msg, km.OpenFolder):
		// Open folder prompt
		m.openFolderActive = true
		m.openFolderPath = m.explorer.GetRootPath()
		if m.openFolderPath == "" {
			cwd, _ := os.Getwd()
			m.openFolderPath = cwd
		}
		return m, nil

	case keymap.Matches(msg, km.ShowHelp):
		// Show help message overriding current status
		m.statusMsg = "Rift Editor - Ctrl+B: Toggle Explorer | Ctrl+S: Save | Ctrl+Q: Quit | Ctrl+G: Line Numbers | F1: Help"
		return m, nil
	}

	// Pass to focused component
	var cmd tea.Cmd
	switch m.focus {
	case FocusExplorer:
		m.explorer, cmd = m.explorer.Update(msg)
	case FocusEditor:
		m.editor, cmd = m.editor.Update(msg)
	}

	return m, cmd
}

// handleMouseEvent routes mouse events to components
func (m *Model) handleMouseEvent(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Determine which component was clicked
	if m.showExplorer && msg.X < m.explorerWidth {
		m.focus = FocusExplorer
		// Adjust X to be relative to explorer
		adjMsg := msg
		adjMsg.X = msg.X
		var cmd tea.Cmd
		m.explorer, cmd = m.explorer.Update(adjMsg)
		return m, cmd
	} else {
		m.focus = FocusEditor
		// Adjust X to be relative to editor
		adjMsg := msg
		if m.showExplorer {
			adjMsg.X = msg.X - m.explorerWidth
		}
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(adjMsg)
		return m, cmd
	}
}

// FocusTarget indicates which component has focus
type FocusTarget int

const (
	FocusExplorer FocusTarget = iota
	FocusEditor
)

// Model is the main application state
type Model struct {
	// Dimensions
	width  int
	height int

	// Layout
	explorerWidth int
	showExplorer  bool
	focus         FocusTarget

	// Sub-components
	explorer *explorer.Model
	editor   *editor.Model
	keymap   *keymap.Model
	reviewer *diff.ReviewModel

	// State
	err       error
	statusMsg string

	// Save-as prompt
	saveAsActive bool
	saveAsPath   string

	// Open folder prompt
	openFolderActive bool
	openFolderPath   string
}

// New creates a new application model with optional path argument
func New(startPath string) (*Model, tea.Cmd) {
	explorerModel, explorerCmd := explorer.New()

	// Determine if path is file or folder
	if startPath != "" {
		info, err := os.Stat(startPath)
		if err == nil {
			if info.IsDir() {
				// Change explorer root to this folder
				return &Model{
					explorerWidth: 30,
					showExplorer:  true,
					focus:         FocusExplorer,
					explorer:      explorerModel,
					editor:        editor.New(),
					keymap:        keymap.New(),
					reviewer:      diff.NewReviewModel(),
				}, tea.Batch(explorerCmd, explorerModel.ChangeRoot(startPath))
			} else {
				// Open the file in editor - set explorer to file's directory and open file
				return &Model{
						explorerWidth: 30,
						showExplorer:  true,
						focus:         FocusEditor,
						explorer:      explorerModel,
						editor:        editor.New(),
						keymap:        keymap.New(),
						reviewer:      diff.NewReviewModel(),
					}, tea.Batch(
						explorerCmd,
						explorerModel.ChangeRoot(filepath.Dir(startPath)),
						func() tea.Msg {
							// This will be handled in Update as FileSelectedMsg
							return explorer.FileSelectedMsg{Path: startPath}
						},
					)
			}
		}
	}

	return &Model{
		explorerWidth: 30,
		showExplorer:  true,
		focus:         FocusExplorer,
		explorer:      explorerModel,
		editor:        editor.New(),
		keymap:        keymap.New(),
		reviewer:      diff.NewReviewModel(),
	}, explorerCmd
}

// Init initializes the application
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.explorer.Init(),
		m.editor.Init(),
		m.reviewer.Init(),
		tea.EnableMouseCellMotion,
	)
}

// Update handles all messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateChildSizes()

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.MouseMsg:
		return m.handleMouseEvent(msg)

	case explorer.FileSelectedMsg:
		m.focus = FocusEditor
		cmds = append(cmds, m.editor.OpenFile(msg.Path))
		m.statusMsg = fmt.Sprintf("Opened: %s", msg.Path)

	case explorer.StatusMsg:
		m.statusMsg = string(msg)

	case editor.StatusMsg:
		m.statusMsg = string(msg)
		// Refresh explorer when a file is saved (new files appear in explorer)
		if strings.HasPrefix(string(msg), "Saved: ") {
			cmds = append(cmds, m.explorer.Refresh())
		}

	case explorer.RefreshCompleteMsg:
		m.statusMsg = fmt.Sprintf("Explorer refreshed (%d items)", msg.ItemCount)
	}

	// Update focused component
	var cmd tea.Cmd
	switch m.focus {
	case FocusExplorer:
		m.explorer, cmd = m.explorer.Update(msg)
		cmds = append(cmds, cmd)
	case FocusEditor:
		m.editor, cmd = m.editor.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Always update keymap for global shortcuts
	m.keymap, cmd = m.keymap.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the application
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Main content area
	var content string
	if m.showExplorer {
		explorerView := m.explorer.View()
		editorView := m.editor.View()
		content = lipgloss.JoinHorizontal(lipgloss.Top, explorerView, editorView)
	} else {
		content = m.editor.View()
	}

	// Status bar at bottom
	statusBar := m.renderStatusBar()
	mainContent := lipgloss.JoinVertical(lipgloss.Left, content, statusBar)

	// If save-as is active, show prompt
	if m.saveAsActive {
		saveAsPrompt := m.renderSaveAs()
		return lipgloss.JoinVertical(lipgloss.Left, mainContent, saveAsPrompt)
	}

	// If open-folder is active, show prompt
	if m.openFolderActive {
		openFolderPrompt := m.renderOpenFolder()
		return lipgloss.JoinVertical(lipgloss.Left, mainContent, openFolderPrompt)
	}

	// If diff reviewer is active, render it as overlay
	if m.reviewer.IsActive() {
		return lipgloss.JoinVertical(lipgloss.Left, mainContent, m.reviewer.View())
	}

	return mainContent
}

// HasError returns true if the app encountered a fatal error
func (m *Model) HasError() bool {
	return m.err != nil
}

// updateChildSizes updates dimensions of child components
func (m *Model) updateChildSizes() {
	statusHeight := 1
	availableHeight := m.height - statusHeight

	if m.showExplorer {
		m.explorer.SetSize(m.explorerWidth, availableHeight)
		m.editor.SetSize(m.width-m.explorerWidth, availableHeight)
	} else {
		m.explorer.SetSize(0, availableHeight)
		m.editor.SetSize(m.width, availableHeight)
	}

	// Reviewer takes the bottom portion of the screen
	m.reviewer.SetSize(m.width, 10)
}

// renderStatusBar renders the bottom status bar
func (m *Model) renderStatusBar() string {
	style := lipgloss.NewStyle().
		Background(lipgloss.Color("#1e1e1e")).
		Foreground(lipgloss.Color("#ffffff")).
		Width(m.width)

	content := m.statusMsg
	if content == "" {
		content = "Rift Editor - Ctrl+B: Toggle Explorer | Ctrl+S: Save | Ctrl+Q: Quit"
	}

	return style.Render(content)
}

// handleSaveAsInput handles keyboard input during save-as mode
func (m *Model) handleSaveAsInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		// Confirm save
		m.saveAsActive = false
		return m, m.editor.SaveAs(m.saveAsPath)

	case tea.KeyEsc, tea.KeyCtrlC:
		// Cancel save-as
		m.saveAsActive = false
		m.statusMsg = "Save cancelled"
		return m, nil

	case tea.KeyBackspace:
		// Delete last character
		if len(m.saveAsPath) > 0 {
			m.saveAsPath = m.saveAsPath[:len(m.saveAsPath)-1]
		}
		return m, nil

	case tea.KeyRunes:
		// Add typed character
		m.saveAsPath += string(msg.Runes)
		return m, nil
	}

	return m, nil
}

// renderSaveAs renders the save-as prompt overlay
func (m *Model) renderSaveAs() string {
	style := lipgloss.NewStyle().
		Background(lipgloss.Color("#264f78")).
		Foreground(lipgloss.Color("#ffffff")).
		Width(m.width).
		Padding(0, 1)

	label := "Save as (Confirm: Enter, Cancel: Esc): "
	return style.Render(label + m.saveAsPath)
}

// handleOpenFolderInput handles keyboard input during open-folder mode
func (m *Model) handleOpenFolderInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		// Confirm folder change
		m.openFolderActive = false
		return m, m.explorer.ChangeRoot(m.openFolderPath)

	case tea.KeyEsc, tea.KeyCtrlC:
		// Cancel open-folder
		m.openFolderActive = false
		m.statusMsg = "Open folder cancelled"
		return m, nil

	case tea.KeyBackspace:
		// Delete last character
		if len(m.openFolderPath) > 0 {
			m.openFolderPath = m.openFolderPath[:len(m.openFolderPath)-1]
		}
		return m, nil

	case tea.KeyRunes:
		// Add typed character
		m.openFolderPath += string(msg.Runes)
		return m, nil
	}

	return m, nil
}

// renderOpenFolder renders the open-folder prompt overlay
func (m *Model) renderOpenFolder() string {
	style := lipgloss.NewStyle().
		Background(lipgloss.Color("#264f78")).
		Foreground(lipgloss.Color("#ffffff")).
		Width(m.width).
		Padding(0, 1)

	label := "Open folder (Confirm: Enter, Cancel: Esc): "
	return style.Render(label + m.openFolderPath)
}
