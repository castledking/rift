package diff

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ChangeType indicates the type of change
type ChangeType int

const (
	ChangeUnchanged ChangeType = iota
	ChangeAdded
	ChangeRemoved
)

// LineChange represents a single line change
type LineChange struct {
	Type    ChangeType
	OldLine int // Line number in original (0-indexed, -1 for added)
	NewLine int // Line number in new (0-indexed, -1 for removed)
	Content string
}

// Hunk represents a contiguous block of changes
type Hunk struct {
	OldStart int // Starting line in original
	OldCount int // Number of lines in original
	NewStart int // Starting line in new
	NewCount int // Number of lines in new
	Lines    []LineChange
}

// Diff represents a complete diff between two texts
type Diff struct {
	OldPath string
	NewPath string
	Hunks   []Hunk
}

// ComputeDiff calculates the diff between old and new content
func ComputeDiff(oldPath, newPath string, oldLines, newLines []string) *Diff {
	d := &Diff{
		OldPath: oldPath,
		NewPath: newPath,
	}

	// Simple diff algorithm: Myers' algorithm or simpler LCS
	// For MVP, we'll use a simple line-by-line comparison
	d.Hunks = d.computeHunks(oldLines, newLines)
	return d
}

// computeHunks finds contiguous blocks of changes
func (d *Diff) computeHunks(oldLines, newLines []string) []Hunk {
	var hunks []Hunk

	// Track positions
	oldIdx, newIdx := 0, 0

	for oldIdx < len(oldLines) || newIdx < len(newLines) {
		// Find next difference
		for oldIdx < len(oldLines) && newIdx < len(newLines) && oldLines[oldIdx] == newLines[newIdx] {
			oldIdx++
			newIdx++
		}

		if oldIdx >= len(oldLines) && newIdx >= len(newLines) {
			break
		}

		// Start of a hunk
		hunk := Hunk{
			OldStart: oldIdx,
			NewStart: newIdx,
		}

		// Collect changed lines
		for oldIdx < len(oldLines) || newIdx < len(newLines) {
			// Check if we should stop this hunk (3 lines of context match)
			matchCount := 0
			for i := 0; i < 3 && oldIdx+i < len(oldLines) && newIdx+i < len(newLines); i++ {
				if oldLines[oldIdx+i] == newLines[newIdx+i] {
					matchCount++
				} else {
					break
				}
			}

			if matchCount == 3 && len(hunk.Lines) > 0 {
				// Add context lines and stop
				for i := 0; i < 3 && oldIdx < len(oldLines); i++ {
					hunk.Lines = append(hunk.Lines, LineChange{
						Type:    ChangeUnchanged,
						OldLine: oldIdx,
						NewLine: newIdx,
						Content: oldLines[oldIdx],
					})
					oldIdx++
					newIdx++
				}
				break
			}

			// Add the next change
			if oldIdx < len(oldLines) && (newIdx >= len(newLines) || (oldIdx < len(oldLines) && newIdx < len(newLines) && oldLines[oldIdx] != newLines[newIdx])) {
				// Line removed
				hunk.Lines = append(hunk.Lines, LineChange{
					Type:    ChangeRemoved,
					OldLine: oldIdx,
					NewLine: -1,
					Content: oldLines[oldIdx],
				})
				oldIdx++
				hunk.OldCount++
			} else if newIdx < len(newLines) {
				// Line added
				hunk.Lines = append(hunk.Lines, LineChange{
					Type:    ChangeAdded,
					OldLine: -1,
					NewLine: newIdx,
					Content: newLines[newIdx],
				})
				newIdx++
				hunk.NewCount++
			}
		}

		if len(hunk.Lines) > 0 {
			hunks = append(hunks, hunk)
		}
	}

	return hunks
}

// ApplyHunk applies a single hunk to the given content
func ApplyHunk(lines []string, hunk Hunk) []string {
	// Implementation to apply a hunk
	// This is used when accepting changes
	result := make([]string, 0, len(lines)+hunk.NewCount-hunk.OldCount)

	// Copy lines before hunk
	result = append(result, lines[:hunk.OldStart]...)

	// Apply changes
	for _, change := range hunk.Lines {
		switch change.Type {
		case ChangeAdded:
			result = append(result, change.Content)
		case ChangeUnchanged:
			result = append(result, change.Content)
		case ChangeRemoved:
			// Skip removed lines
		}
	}

	// Copy lines after hunk
	result = append(result, lines[hunk.OldStart+hunk.OldCount:]...)

	return result
}

// RevertHunk removes a hunk's changes from new content
func RevertHunk(lines []string, hunk Hunk) []string {
	// Implementation to revert a hunk
	result := make([]string, 0, len(lines)+hunk.OldCount-hunk.NewCount)

	// Copy lines before hunk
	result = append(result, lines[:hunk.NewStart]...)

	// Apply changes in reverse
	for _, change := range hunk.Lines {
		switch change.Type {
		case ChangeRemoved:
			result = append(result, change.Content)
		case ChangeUnchanged:
			result = append(result, change.Content)
		case ChangeAdded:
			// Skip added lines
		}
	}

	// Copy lines after hunk
	result = append(result, lines[hunk.NewStart+hunk.NewCount:]...)

	return result
}

// FormatDiffLine formats a single diff line for display
func FormatDiffLine(change LineChange, width int) string {
	prefix := " "
	style := lipgloss.NewStyle()

	switch change.Type {
	case ChangeAdded:
		prefix = "+"
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ec9b0")).
			Background(lipgloss.Color("#0d3d3d"))
	case ChangeRemoved:
		prefix = "-"
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f44747")).
			Background(lipgloss.Color("#3d0d0d"))
	case ChangeUnchanged:
		prefix = " "
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#d4d4d4"))
	}

	content := change.Content
	if len(content) > width-2 {
		content = content[:width-5] + "..."
	}

	return style.Width(width).Render(fmt.Sprintf("%s %s", prefix, content))
}

// String returns a unified diff format representation
func (d *Diff) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("--- %s\n", d.OldPath))
	sb.WriteString(fmt.Sprintf("+++ %s\n", d.NewPath))

	for _, hunk := range d.Hunks {
		sb.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
			hunk.OldStart+1, hunk.OldCount,
			hunk.NewStart+1, hunk.NewCount))

		for _, line := range hunk.Lines {
			switch line.Type {
			case ChangeAdded:
				sb.WriteString(fmt.Sprintf("+%s\n", line.Content))
			case ChangeRemoved:
				sb.WriteString(fmt.Sprintf("-%s\n", line.Content))
			default:
				sb.WriteString(fmt.Sprintf(" %s\n", line.Content))
			}
		}
	}

	return sb.String()
}
