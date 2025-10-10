package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Table represents a scrollable table component
type Table struct {
	width     int
	height    int
	cursor    int
	offset    int
	rowStyle  lipgloss.Style
	headerStyle lipgloss.Style
	borderStyle lipgloss.Style
}

// NewTable creates a new table with default styling
func NewTable() *Table {
	lightBlue := lipgloss.Color("#87CEEB")
	darkBlue := lipgloss.Color("#4682B4")

	return &Table{
		width:  80,
		height: 20,
		cursor: 0,
		offset: 0,
		headerStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(darkBlue).
			Align(lipgloss.Center),
		rowStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		borderStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lightBlue).
			Padding(0, 1),
	}
}

// SetWidth sets the table width
func (t *Table) SetWidth(width int) {
	t.width = width
}

// SetHeight sets the table height
func (t *Table) SetHeight(height int) {
	t.height = height
}

// MoveUp moves the cursor up by n rows
func (t *Table) MoveUp(n int) {
	t.cursor -= n
	if t.cursor < 0 {
		t.cursor = 0
	}
	t.updateOffset()
}

// MoveDown moves the cursor down by n rows
func (t *Table) MoveDown(n int) {
	t.cursor += n
	if t.cursor >= t.maxCursor() {
		t.cursor = t.maxCursor()
	}
	t.updateOffset()
}

// GotoTop moves cursor to the top
func (t *Table) GotoTop() {
	t.cursor = 0
	t.offset = 0
}

// GotoBottom moves cursor to the bottom
func (t *Table) GotoBottom() {
	t.cursor = t.maxCursor()
	t.updateOffset()
}

// maxCursor returns the maximum cursor position
func (t *Table) maxCursor() int {
	return max(0, t.height-1)
}

// updateOffset adjusts the view offset based on cursor position
func (t *Table) updateOffset() {
	if t.cursor < t.offset {
		t.offset = t.cursor
	} else if t.cursor >= t.offset+t.height {
		t.offset = t.cursor - t.height + 1
	}
}

// View renders the table as a string
func (t *Table) View(rows []contentRow) string {
	if len(rows) == 0 {
		return "No data to display"
	}

	// Define column widths based on total width
	idWidth := 8
	titleWidth := t.width*3/10 - 2 // 30% of width
	authorWidth := t.width*2/10 - 2 // 20% of width
	sourceWidth := 10
	dateWidth := 10
	previewWidth := t.width - idWidth - titleWidth - authorWidth - sourceWidth - dateWidth - 12 // Account for padding

	// Create header
	header := t.createHeader(idWidth, titleWidth, authorWidth, sourceWidth, dateWidth, previewWidth)

	// Create visible rows
	visibleRows := t.getVisibleRows(rows, idWidth, titleWidth, authorWidth, sourceWidth, dateWidth, previewWidth)

	// Combine header and rows
	content := lipgloss.JoinVertical(lipgloss.Left, header, visibleRows)

	// Apply border
	return t.borderStyle.Render(content)
}

// createHeader creates the table header
func (t *Table) createHeader(idWidth, titleWidth, authorWidth, sourceWidth, dateWidth, previewWidth int) string {
	cells := []string{
		t.headerStyle.Render(lipgloss.PlaceHorizontal(idWidth, lipgloss.Center, "ID")),
		t.headerStyle.Render(lipgloss.PlaceHorizontal(titleWidth, lipgloss.Center, "Title")),
		t.headerStyle.Render(lipgloss.PlaceHorizontal(authorWidth, lipgloss.Center, "Author")),
		t.headerStyle.Render(lipgloss.PlaceHorizontal(sourceWidth, lipgloss.Center, "Source")),
		t.headerStyle.Render(lipgloss.PlaceHorizontal(dateWidth, lipgloss.Center, "Date")),
		t.headerStyle.Render(lipgloss.PlaceHorizontal(previewWidth, lipgloss.Center, "Preview")),
	}

	separator := t.headerStyle.Render(strings.Repeat("─", t.width-4))

	return lipgloss.JoinHorizontal(lipgloss.Left, cells...) + "\n" + separator
}

// getVisibleRows returns the visible rows with styling
func (t *Table) getVisibleRows(rows []contentRow, idWidth, titleWidth, authorWidth, sourceWidth, dateWidth, previewWidth int) string {
	var renderedRows []string

	start := t.offset
	end := min(start+t.height, len(rows))

	if start >= len(rows) {
		start = len(rows) - 1
		if start < 0 {
			start = 0
		}
		end = len(rows)
	}

	for i := start; i < end; i++ {
		row := rows[i]
		style := t.rowStyle

		// Highlight cursor row
		if i == t.cursor {
			style = style.Copy().Background(lipgloss.Color("#87CEEB")).Foreground(lipgloss.Color("0"))
		}

		// Truncate content to fit column width
		title := truncate(row.Title, titleWidth)
		author := truncate(row.Author, authorWidth)
		source := truncate(row.Source, sourceWidth)
		preview := truncate(row.Preview, previewWidth)

		cells := []string{
			style.Render(lipgloss.PlaceHorizontal(idWidth, lipgloss.Left, row.ID)),
			style.Render(lipgloss.PlaceHorizontal(titleWidth, lipgloss.Left, title)),
			style.Render(lipgloss.PlaceHorizontal(authorWidth, lipgloss.Left, author)),
			style.Render(lipgloss.PlaceHorizontal(sourceWidth, lipgloss.Left, source)),
			style.Render(lipgloss.PlaceHorizontal(dateWidth, lipgloss.Left, row.Date)),
			style.Render(lipgloss.PlaceHorizontal(previewWidth, lipgloss.Left, preview)),
		}

		renderedRows = append(renderedRows, lipgloss.JoinHorizontal(lipgloss.Left, cells...))
	}

	return lipgloss.JoinVertical(lipgloss.Left, renderedRows...)
}

// truncate truncates a string to fit within width, adding "..." if needed
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if len(s) <= width {
		return s
	}
	if width <= 3 {
		return s[:width]
	}
	return s[:width-3] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}