package explorer

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StatusMsg is sent to report explorer status
type StatusMsg string

// RefreshCompleteMsg is sent when explorer refresh is done
type RefreshCompleteMsg struct {
	ItemCount int
}

// FileSelectedMsg is sent when a file is selected
type FileSelectedMsg struct {
	Path string
}

// TreeNode represents a file or directory in the explorer
type TreeNode struct {
	Path     string
	Name     string
	IsDir    bool
	Expanded bool
	Children []*TreeNode
	Parent   *TreeNode
}

// Model is the file explorer state
type Model struct {
	width        int
	height       int
	root         *TreeNode
	cursor       int         // Index of currently selected node in flat list
	flat         []*TreeNode // Flattened view of visible nodes
	scrollOffset int         // First visible item index for scrolling
}

// New creates a new file explorer model
func New() (*Model, tea.Cmd) {
	m := &Model{
		cursor:       0,
		scrollOffset: 0,
	}
	cwd, fileCount := m.loadWorkingDirectory()
	return m, func() tea.Msg {
		return StatusMsg(fmt.Sprintf("Loaded %s (%d items)", cwd, fileCount))
	}
}

// Init initializes the explorer
func (m *Model) Init() tea.Cmd {
	return nil
}

// SetSize updates the explorer dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update processes messages
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.flat)-1 {
				m.cursor++
			}
		case "enter", " ":
			if m.cursor < len(m.flat) {
				node := m.flat[m.cursor]
				if node.IsDir {
					node.Expanded = !node.Expanded
					m.refreshFlat()
				} else {
					return m, func() tea.Msg {
						return FileSelectedMsg{Path: node.Path}
					}
				}
			}
		}

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			// Scroll up - move cursor and scroll offset up
			if m.cursor > 0 {
				m.cursor--
			}
			if m.cursor < m.scrollOffset {
				m.scrollOffset = m.cursor
			}
		case tea.MouseButtonWheelDown:
			// Scroll down - move cursor and scroll offset down
			if m.cursor < len(m.flat)-1 {
				m.cursor++
			}
			if m.cursor >= m.scrollOffset+m.height {
				m.scrollOffset = m.cursor - m.height + 1
			}
		}
	}

	return m, nil
}

// View renders the file explorer
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	// If no items to show, display error/info message
	if len(m.flat) == 0 {
		msg := "No files to display"
		if m.root == nil {
			msg = "Error: Cannot access directory"
		}
		style := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Background(lipgloss.Color("#1e1e1e")).
			Foreground(lipgloss.Color("#858585"))
		return style.Render(msg)
	}

	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Background(lipgloss.Color("#1e1e1e"))

	cursorStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#264f78"))

	var lines []string
	startIdx := m.scrollOffset
	if startIdx < 0 {
		startIdx = 0
	}
	if startIdx >= len(m.flat) {
		startIdx = 0
	}
	for i := startIdx; i < len(m.flat); i++ {
		node := m.flat[i]
		line := m.renderNode(node)
		if i == m.cursor {
			line = cursorStyle.Render(line)
		}
		lines = append(lines, line)
		if len(lines) >= m.height {
			break
		}
	}

	// Fill remaining space
	for len(lines) < m.height {
		lines = append(lines, "")
	}

	content := ""
	for _, line := range lines {
		content += line + "\n"
	}

	return style.Render(content)
}

// renderNode renders a single tree node
func (m *Model) renderNode(node *TreeNode) string {
	prefix := ""
	if node.Parent != nil {
		prefix = "  "
	}

	if node.IsDir {
		if node.Expanded {
			prefix += "▼ "
		} else {
			prefix += "▶ "
		}
	} else {
		prefix += "  "
	}

	name := node.Name
	// Truncate if too long
	maxLen := m.width - len(prefix) - 1
	if len(name) > maxLen && maxLen > 3 {
		name = name[:maxLen-3] + "..."
	}

	return prefix + name
}

// loadWorkingDirectory loads the current directory structure
func (m *Model) loadWorkingDirectory() (string, int) {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	m.root = m.buildTree(cwd, nil)
	if m.root != nil {
		m.root.Expanded = true // Auto-expand root
	}
	m.refreshFlat()
	return cwd, len(m.flat)
}

// buildTree recursively builds the file tree
func (m *Model) buildTree(path string, parent *TreeNode) *TreeNode {
	info, err := os.Stat(path)
	if err != nil {
		log.Printf("Error accessing directory %s: %v", path, err)
		return nil
	}

	node := &TreeNode{
		Path:   path,
		Name:   info.Name(),
		IsDir:  info.IsDir(),
		Parent: parent,
	}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			log.Printf("Error reading directory %s: %v", path, err)
			return node
		}

		// Sort: directories first, then files
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].IsDir() != entries[j].IsDir() {
				return entries[i].IsDir()
			}
			return entries[i].Name() < entries[j].Name()
		})

		for _, entry := range entries {
			// Skip hidden files (starting with .)
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			if child := m.buildTree(filepath.Join(path, entry.Name()), node); child != nil {
				node.Children = append(node.Children, child)
			}
		}
	}

	return node
}

// refreshFlat updates the flattened view based on expansion state
func (m *Model) refreshFlat() {
	m.flat = nil
	if m.root != nil {
		m.flatten(m.root, 0)
	}
	// Clamp cursor
	if m.cursor >= len(m.flat) {
		m.cursor = len(m.flat) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// flatten recursively adds visible nodes to flat list
func (m *Model) flatten(node *TreeNode, depth int) {
	m.flat = append(m.flat, node)
	if node.Expanded {
		for _, child := range node.Children {
			m.flatten(child, depth+1)
		}
	}
}

// Refresh reloads the directory structure
func (m *Model) Refresh() tea.Cmd {
	return func() tea.Msg {
		cwd, itemCount := m.loadWorkingDirectory()
		_ = cwd // cwd not used in message but useful for debugging
		return RefreshCompleteMsg{ItemCount: itemCount}
	}
}

// GetRootPath returns the current root directory path
func (m *Model) GetRootPath() string {
	if m.root != nil {
		return m.root.Path
	}
	return ""
}

// ChangeRoot changes the root directory of the explorer
func (m *Model) ChangeRoot(path string) tea.Cmd {
	return func() tea.Msg {
		m.root = m.buildTree(path, nil)
		if m.root != nil {
			m.root.Expanded = true
		}
		m.refreshFlat()
		return StatusMsg(fmt.Sprintf("Changed folder: %s (%d items)", path, len(m.flat)))
	}
}
