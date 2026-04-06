package keymap

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Model holds keybinding state
type Model struct{}

// New creates a new keymap model
func New() *Model {
	return &Model{}
}

// Init initializes the keymap
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update processes messages
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	return m, nil
}

// GlobalKeyMap defines all global shortcuts
type GlobalKeyMap struct {
	Save              []string
	Quit              []string
	NewFile           []string
	OpenFolder        []string
	ToggleExplorer    []string
	Undo              []string
	Redo              []string
	Copy              []string
	Cut               []string
	Paste             []string
	Find              []string
	CommandPalette    []string
	QuickOpen         []string
	SelectAll         []string
	ToggleLineNumbers []string
	ShowHelp          []string
}

// DefaultGlobalKeyMap returns the default keybindings
func DefaultGlobalKeyMap() GlobalKeyMap {
	return GlobalKeyMap{
		Save:              []string{"ctrl+s"},
		Quit:              []string{"ctrl+q"},
		NewFile:           []string{"ctrl+n"},
		OpenFolder:        []string{"ctrl+o"},
		ToggleExplorer:    []string{"ctrl+b"},
		Undo:              []string{"ctrl+z"},
		Redo:              []string{"ctrl+shift+z"},
		Copy:              []string{"ctrl+c"},
		Cut:               []string{"ctrl+x"},
		Paste:             []string{"ctrl+v"},
		Find:              []string{"ctrl+f"},
		CommandPalette:    []string{"ctrl+shift+p"},
		QuickOpen:         []string{"ctrl+p"},
		SelectAll:         []string{"ctrl+a"},
		ToggleLineNumbers: []string{"ctrl+g"},
		ShowHelp:          []string{"f1"},
	}
}

// Matches checks if a key message matches any of the given keys
func Matches(msg tea.KeyMsg, keys []string) bool {
	for _, key := range keys {
		if msg.String() == key {
			return true
		}
	}
	return false
}
