package tui

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"golino/internal/colinodb"
	"golino/internal/config"
)

type model struct {
	items        []colinodb.Content
	table        *table.Table
	cursor       int
	pageSize     int
	currentPage  int
	totalPages   int
	tableWidth   int
	tableHeight  int
	urlWidth     int
	titleWidth   int
	authorWidth  int
	dateWidth    int
	previewWidth int
	ready        bool
	err          error
}

func Run(ctx context.Context) error {
	dbPath, err := config.LoadDBPath()
	if err != nil {
		return err
	}

	db, err := colinodb.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed opening the Colino database: %w", err)
	}
	defer db.Close()

	// Fetch all data from database for full pagination
	rows, err := colinodb.GetSince(ctx, db, time.Time{}, "", 0) // Empty time to get all data
	if err != nil {
		return fmt.Errorf("query failed while reading from the Colino database: %w", err)
	}

	m := model{
		items:       rows,
		cursor:      0,
		pageSize:    10, // Default page size, will be updated on resize
		currentPage: 0,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "k":
			if m.cursor > 0 {
				m.cursor--
			} else if m.currentPage > 0 {
				// Move to previous page
				m.currentPage--
				m.cursor = m.pageSize - 1
			}
			m.updateTableRows()
			return m, tea.ClearScreen
		case "j":
			itemsOnCurrentPage := min(m.pageSize, len(m.items)-m.currentPage*m.pageSize)
			if m.cursor < itemsOnCurrentPage-1 {
				m.cursor++
			} else if m.currentPage < m.totalPages-1 {
				// Move to next page
				m.currentPage++
				m.cursor = 0
			}
			m.updateTableRows()
			return m, tea.ClearScreen
		case "g":
			m.currentPage = 0
			m.cursor = 0
			m.updateTableRows()
			return m, tea.ClearScreen
		case "G":
			m.currentPage = m.totalPages - 1
			lastPageItems := len(m.items) % m.pageSize
			if lastPageItems == 0 {
				lastPageItems = m.pageSize
			}
			m.cursor = lastPageItems - 1
			m.updateTableRows()
			return m, tea.ClearScreen
		case "l": // Next page
			if m.currentPage < m.totalPages-1 {
				m.currentPage++
				m.cursor = 0
				m.updateTableRows()
			}
			return m, nil
		case "h": // Previous page
			if m.currentPage > 0 {
				m.currentPage--
				m.cursor = 0
				m.updateTableRows()
			}
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.tableWidth = msg.Width
		m.tableHeight = msg.Height - 4
		m.configureTable(msg.Width, msg.Height-4) // Leave room for borders/title
		m.ready = true
		return m, tea.ClearScreen
	}

	return m, nil
}

func (m model) View() string {
	if !m.ready {
		return "Loading..."
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}

	if len(m.items) == 0 {
		return "No content found in the database."
	}

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true).
		Align(lipgloss.Center).
		MarginBottom(1).
		Render("All Articles")

	// Define colors
	// lightBlue := lipgloss.Color("#87CEEB")
	// Wrap table in blue border
	tableContainer := m.table.Render()

	// Help info
	helpInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Align(lipgloss.Center).
		Render("j/k: move • l/h: page • g/G: home/end • q: quit")

	return lipgloss.JoinVertical(lipgloss.Left, title, tableContainer, helpInfo)
}

// configureTable sets up the table with dynamic column widths based on available space
func (m *model) configureTable(width, height int) {
	if len(m.items) == 0 {
		return
	}

	// Calculate page size (leave space for header, borders, and pagination info)
	m.pageSize = max(5, height-6) // At least 5 rows, leave space for header, borders, and pagination info
	m.totalPages = (len(m.items) + m.pageSize - 1) / m.pageSize

	// Ensure current page is valid
	if m.currentPage >= m.totalPages {
		m.currentPage = m.totalPages - 1
	}
	if m.currentPage < 0 {
		m.currentPage = 0
	}

	// Calculate cursor position within current page
	globalCursor := m.currentPage*m.pageSize + m.cursor
	if globalCursor >= len(m.items) {
		globalCursor = len(m.items) - 1
		m.currentPage = globalCursor / m.pageSize
		m.cursor = globalCursor % m.pageSize
	}

	// Calculate dynamic column widths (4 columns now)
	m.dateWidth = 10
	// Account for borders and padding more accurately: 2 chars left border + 2 chars right border + 3 chars padding per column * 4 columns
	borderPaddingWidth := 4 + (3 * 4) // 4 for borders, 12 for padding = 16 total
	remainingWidth := width - m.dateWidth - borderPaddingWidth

	m.titleWidth = remainingWidth * 35 / 100   // Increased from 30%
	m.authorWidth = remainingWidth * 25 / 100  // Same as before
	m.previewWidth = remainingWidth * 40 / 100 // Increased from 45%

	// Ensure minimum widths
	if m.titleWidth < 20 {
		m.titleWidth = 20 // Increased minimum
	}
	if m.authorWidth < 12 {
		m.authorWidth = 12
	}
	if m.previewWidth < 25 {
		m.previewWidth = 25 // Increased minimum
	}

	// Calculate total used width after ensuring minimums
	totalUsedWidth := m.titleWidth + m.authorWidth + m.dateWidth + m.previewWidth + borderPaddingWidth

	// If we have unused space, distribute it proportionally to use full width
	if totalUsedWidth < width {
		unusedWidth := width - totalUsedWidth
		m.titleWidth += unusedWidth * 35 / 100
		m.authorWidth += unusedWidth * 25 / 100
		m.previewWidth += unusedWidth * 40 / 100
	}

	m.updateTableRows()
}

// updateTableRows updates only the table rows without recalculating layout
func (m *model) updateTableRows() {
	if len(m.items) == 0 {
		return
	}

	// Prepare headers
	headers := []string{
		truncateString("Title", m.titleWidth),
		truncateString("Author", m.authorWidth),
		truncateString("Date", m.dateWidth),
		truncateString("Preview", m.previewWidth),
	}

	// Prepare rows - only show current page
	var rows [][]string
	startIdx := m.currentPage * m.pageSize
	endIdx := min(startIdx+m.pageSize, len(m.items))

	for i := startIdx; i < endIdx; i++ {
		item := m.items[i]
		title := extractTitle(item.Metadata)
		if title == "" {
			title = "No title"
		}

		author := item.AuthorUsername
		if author == "" {
			author = "Unknown author"
		}

		preview := item.Content
		// Split by newlines and take the first non-empty line
		lines := strings.Split(strings.TrimSpace(preview), "\n")
		if len(lines) > 0 {
			preview = lines[0]
		}
		// Truncate if still too long
		previewLimit := m.previewWidth - 3
		if len(preview) > previewLimit && previewLimit > 0 {
			preview = preview[:previewLimit] + "..."
		}

		row := []string{
			truncateString(title, m.titleWidth),
			truncateString(author, m.authorWidth),
			truncateString(item.CreatedAt.Format("2006-01-02"), m.dateWidth),
			truncateString(preview, m.previewWidth),
		}
		rows = append(rows, row)
	}

	// Define colors
	lightBlue := lipgloss.Color("#87CEEB")
	darkBlue := lipgloss.Color("#4682B4")

	borderStyle := lipgloss.NewStyle().Foreground(darkBlue)

	// Create base styles
	baseStyle := lipgloss.NewStyle().
		Padding(0, 1)

	headerStyle := baseStyle.Copy().
		Bold(true).
		Foreground(darkBlue).
		Align(lipgloss.Center)

	// Create table with explicit width to use full horizontal space
	m.table = table.New().
		Width(m.tableWidth).
		Border(lipgloss.ThickBorder()).
		BorderStyle(borderStyle).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == -1 { // Header row
				return headerStyle
			}
			if row == m.cursor { // Selected row
				return baseStyle.Copy().
					Background(lightBlue).
					Foreground(lipgloss.Color("0"))
			}
			return baseStyle
		})
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func extractTitle(metadata sql.NullString) string {
	if !metadata.Valid {
		return ""
	}

	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(metadata.String), &meta); err != nil {
		return ""
	}

	if title, ok := meta["entry_title"].(string); ok {
		return title
	}

	return ""
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
