package diff

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ReviewMode indicates the current state of diff review
type ReviewMode int

const (
	ReviewModeNone ReviewMode = iota
	ReviewModeActive
)

// ReviewAction represents an action taken on a hunk
type ReviewAction int

const (
	ActionAccept ReviewAction = iota
	ActionReject
	ActionSkip
)

// HunkReview tracks the review state of a hunk
type HunkReview struct {
	Hunk   Hunk
	Action ReviewAction
	Index  int
}

// ReviewModel is the diff review UI state
type ReviewModel struct {
	width     int
	height    int
	mode      ReviewMode
	diff      *Diff
	hunks     []HunkReview
	cursor    int // Current hunk index
	scroll    int // Scroll position within current hunk
	completed bool
}

// NewReviewModel creates a new diff review model
func NewReviewModel() *ReviewModel {
	return &ReviewModel{
		mode:   ReviewModeNone,
		hunks:  []HunkReview{},
		cursor: 0,
		scroll: 0,
	}
}

// SetSize updates the review panel dimensions
func (m *ReviewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// ShowDiff starts reviewing a diff
func (m *ReviewModel) ShowDiff(diff *Diff) {
	m.mode = ReviewModeActive
	m.diff = diff
	m.hunks = make([]HunkReview, len(diff.Hunks))
	for i, hunk := range diff.Hunks {
		m.hunks[i] = HunkReview{
			Hunk:   hunk,
			Action: ActionSkip,
			Index:  i,
		}
	}
	m.cursor = 0
	m.scroll = 0
	m.completed = false
}

// IsActive returns true if diff review mode is active
func (m *ReviewModel) IsActive() bool {
	return m.mode == ReviewModeActive
}

// Deactivate exits review mode
func (m *ReviewModel) Deactivate() {
	m.mode = ReviewModeNone
}

// Init initializes the review model
func (m *ReviewModel) Init() tea.Cmd {
	return nil
}

// Update processes messages for diff review
func (m *ReviewModel) Update(msg tea.Msg) (*ReviewModel, tea.Cmd) {
	if m.mode != ReviewModeActive {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "enter":
			// Accept current hunk
			if m.cursor < len(m.hunks) {
				m.hunks[m.cursor].Action = ActionAccept
				m.nextHunk()
			}

		case "n", "backspace":
			// Reject current hunk
			if m.cursor < len(m.hunks) {
				m.hunks[m.cursor].Action = ActionReject
				m.nextHunk()
			}

		case "a":
			// Accept all remaining
			for i := m.cursor; i < len(m.hunks); i++ {
				m.hunks[i].Action = ActionAccept
			}
			m.completed = true

		case "r":
			// Reject all remaining
			for i := m.cursor; i < len(m.hunks); i++ {
				m.hunks[i].Action = ActionReject
			}
			m.completed = true

		case "q", "esc":
			// Exit review mode
			m.mode = ReviewModeNone

		case "up", "k":
			if m.scroll > 0 {
				m.scroll--
			}

		case "down", "j":
			m.scroll++

		case "tab":
			m.nextHunk()

		case "shift+tab":
			m.prevHunk()
		}
	}

	return m, nil
}

// nextHunk moves to the next hunk
func (m *ReviewModel) nextHunk() {
	if m.cursor < len(m.hunks)-1 {
		m.cursor++
		m.scroll = 0
	} else {
		m.completed = true
	}
}

// prevHunk moves to the previous hunk
func (m *ReviewModel) prevHunk() {
	if m.cursor > 0 {
		m.cursor--
		m.scroll = 0
	}
}

// View renders the diff review UI
func (m *ReviewModel) View() string {
	if m.mode != ReviewModeActive || m.diff == nil {
		return ""
	}

	// Header
	headerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#264f78")).
		Foreground(lipgloss.Color("#ffffff")).
		Bold(true).
		Width(m.width)

	header := headerStyle.Render(" DIFF REVIEW MODE ")

	// Progress
	progressStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#858585")).
		Width(m.width)
	progress := progressStyle.Render(
		fmt.Sprintf("Hunk %d/%d | (y) Accept (n) Reject (a) Accept All (r) Reject All (q) Quit",
			m.cursor+1, len(m.hunks)))

	// Current hunk display
	var hunkContent []string
	if m.cursor < len(m.hunks) {
		hunk := m.hunks[m.cursor]
		hunkContent = m.renderHunk(hunk)
	} else if m.completed {
		hunkContent = []string{"All hunks reviewed. Press 'q' to exit."}
	}

	// Join everything
	contentHeight := m.height - 3 // header + progress + padding
	content := strings.Join(hunkContent, "\n")
	if len(hunkContent) > contentHeight {
		// Scroll within hunk
		end := m.scroll + contentHeight
		if end > len(hunkContent) {
			end = len(hunkContent)
		}
		if m.scroll < len(hunkContent) {
			content = strings.Join(hunkContent[m.scroll:end], "\n")
		}
	}

	contentStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(contentHeight).
		Background(lipgloss.Color("#1e1e1e"))

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		progress,
		contentStyle.Render(content),
	)
}

// renderHunk renders a single hunk with highlighting
func (m *ReviewModel) renderHunk(review HunkReview) []string {
	var lines []string

	// Hunk header
	hunkHeader := fmt.Sprintf("@@ -%d,%d +%d,%d @@",
		review.Hunk.OldStart+1, review.Hunk.OldCount,
		review.Hunk.NewStart+1, review.Hunk.NewCount)
	
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#569cd6")).
		Background(lipgloss.Color("#252526"))
	lines = append(lines, headerStyle.Render(hunkHeader))

	// Hunk lines
	for _, change := range review.Hunk.Lines {
		lines = append(lines, FormatDiffLine(change, m.width))
	}

	return lines
}

// GetAcceptedHunks returns all hunks marked for acceptance
func (m *ReviewModel) GetAcceptedHunks() []Hunk {
	var accepted []Hunk
	for _, review := range m.hunks {
		if review.Action == ActionAccept {
			accepted = append(accepted, review.Hunk)
		}
	}
	return accepted
}

// GetRejectedHunks returns all hunks marked for rejection
func (m *ReviewModel) GetRejectedHunks() []Hunk {
	var rejected []Hunk
	for _, review := range m.hunks {
		if review.Action == ActionReject {
			rejected = append(rejected, review.Hunk)
		}
	}
	return rejected
}

// ApplyReview applies the reviewed changes to original content
func ApplyReview(original []string, diff *Diff, accepted []Hunk) []string {
	result := make([]string, len(original))
	copy(result, original)

	// Apply accepted hunks in reverse order to maintain line indices
	for i := len(accepted) - 1; i >= 0; i-- {
		result = ApplyHunk(result, accepted[i])
	}

	return result
}
