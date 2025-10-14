package tui

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"golino/internal/colinodb"
	"golino/internal/config"
)

type viewMode int

const (
	tableView viewMode = iota
	detailView
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
	viewMode     viewMode
	selectedItem *colinodb.Content
	viewport     viewport.Model
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
		viewMode:    tableView,
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
			if m.viewMode == detailView {
				// Return to table view on 'q' when in detail view
				m.viewMode = tableView
				m.selectedItem = nil
				m.viewport = viewport.Model{}
				return m, nil
			}
			return m, tea.Quit
		case "esc":
			if m.viewMode == detailView {
				// Return to table view on escape
				m.viewMode = tableView
				m.selectedItem = nil
				m.viewport = viewport.Model{}
				return m, nil
			}
		case " ":
			if m.viewMode == tableView && len(m.items) > 0 {
				// Enter detail view
				globalCursor := m.currentPage*m.pageSize + m.cursor
				if globalCursor < len(m.items) {
					m.viewMode = detailView
					m.selectedItem = &m.items[globalCursor]
					m.setupViewport()
				}
				return m, nil
			}
		}

		// Table view navigation
		if m.viewMode == tableView {
			switch msg.String() {
			case "k":
				if m.cursor > 0 {
					m.cursor--
				} else if m.currentPage > 0 {
					// Move to previous page
					m.currentPage--
					m.cursor = m.pageSize - 1
				}
				m.updateTableRows()
				return m, nil
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
				return m, nil
			case "g":
				m.currentPage = 0
				m.cursor = 0
				m.updateTableRows()
				return m, nil
			case "G":
				m.currentPage = m.totalPages - 1
				lastPageItems := len(m.items) % m.pageSize
				if lastPageItems == 0 {
					lastPageItems = m.pageSize
				}
				m.cursor = lastPageItems - 1
				m.updateTableRows()
				return m, nil
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
		}

		// Detail view navigation - use viewport controls with vim motions only
		if m.viewMode == detailView {
			switch msg.String() {
			case "k":
				m.viewport.LineUp(1)
				return m, nil
			case "j":
				m.viewport.LineDown(1)
				return m, nil
			case "g":
				m.viewport.GotoTop()
				return m, nil
			case "G":
				m.viewport.GotoBottom()
				return m, nil
			}
		}
	case tea.WindowSizeMsg:
		m.tableWidth = msg.Width
		m.tableHeight = msg.Height - 4
		m.configureTable(msg.Width, msg.Height-4) // Leave room for borders/title

		// Update viewport if in detail view
		if m.viewMode == detailView {
			m.setupViewport()
		}

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

	if m.viewMode == detailView {
		return m.renderDetailView()
	}

	return m.renderTableView()
}

func (m model) renderTableView() string {
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true).
		Align(lipgloss.Center).
		MarginBottom(1).
		Render("All Articles")

	// Wrap table in blue border
	tableContainer := m.table.Render()

	// Help info
	helpInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Align(lipgloss.Center).
		Render("j/k: move • l/h: page • g/G: home/end • Space: view details • q: quit")

	return lipgloss.JoinVertical(lipgloss.Left, title, tableContainer, helpInfo)
}

func (m model) renderDetailView() string {
	if m.selectedItem == nil {
		return "No item selected"
	}

	// Extract title
	title := extractTitle(m.selectedItem.Metadata)
	if title == "" {
		title = "No title"
	}

	// Define colors
	lightBlue := lipgloss.Color("#87CEEB")
	darkBlue := lipgloss.Color("#4682B4")
	borderColor := lipgloss.Color("#6B7280")

	// Border styling
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width(m.tableWidth - 4)

	// Title styling
	titleStyle := lipgloss.NewStyle().
		Foreground(darkBlue).
		Bold(true).
		Align(lipgloss.Center).
		MarginBottom(1).
		Width(m.tableWidth - 8)

	// URL styling
	urlStyle := lipgloss.NewStyle().
		Foreground(lightBlue).
		Italic(true).
		MarginBottom(1).
		Width(m.tableWidth - 8)

	// Metadata styling
	metadataStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		MarginBottom(1)

	// Scroll position info styling
	scrollStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Align(lipgloss.Center).
		Bold(true)

	// Render title and URL
	titleRendered := titleStyle.Render(title)

	var urlRendered string
	if m.selectedItem.URL.Valid && m.selectedItem.URL.String != "" {
		urlRendered = urlStyle.Render("URL: " + m.selectedItem.URL.String)
	} else {
		urlRendered = urlStyle.Render("URL: Not available")
	}

	// Render metadata
	author := m.selectedItem.AuthorUsername
	if author == "" {
		author = "Unknown author"
	}
	metadataRendered := metadataStyle.Render(fmt.Sprintf("Author: %s • Date: %s",
		author, m.selectedItem.CreatedAt.Format("2006-01-02 15:04")))

	// Calculate viewport height (leave space for title, URL, metadata, scroll info, help)
	viewportHeight := m.tableHeight - 10
	if viewportHeight < 5 {
		viewportHeight = 5
	}

	// Scroll position info using viewport
	scrollPercent := m.viewport.ScrollPercent()
	if scrollPercent < 0 {
		scrollPercent = 0
	} else if scrollPercent > 1 {
		scrollPercent = 1
	}
	scrollPosition := int(scrollPercent * 100)
	scrollInfo := fmt.Sprintf("Scroll: %d%%", scrollPosition)
	scrollRendered := scrollStyle.Render(scrollInfo)

	// Help info
	helpInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Align(lipgloss.Center).
		MarginTop(1).
		Render("j/k: scroll • g/G: top/bottom • esc/q: back")

	// Combine all elements except viewport content
	headerContent := lipgloss.JoinVertical(lipgloss.Left,
		titleRendered,
		urlRendered,
		metadataRendered)

	// Combine everything
	content := lipgloss.JoinVertical(lipgloss.Left,
		headerContent,
		m.viewport.View(),
		scrollRendered,
		helpInfo)

	return borderStyle.Render(content)
}

func (m *model) setupViewport() {
	if m.selectedItem == nil {
		m.viewport = viewport.Model{}
		return
	}

	// Calculate content width (account for borders and padding)
	contentWidth := m.tableWidth - 12
	if contentWidth < 20 {
		contentWidth = 20
	}

	// Render markdown content using Glow
	renderedContent := renderMarkdown(m.selectedItem.Content, contentWidth)

	// Calculate viewport height (leave space for title, URL, metadata, scroll info, help)
	viewportHeight := m.tableHeight - 10
	if viewportHeight < 5 {
		viewportHeight = 5
	}

	// Initialize viewport with rendered content
	m.viewport = viewport.New(contentWidth, viewportHeight)
	m.viewport.SetContent(renderedContent)
}

// renderMarkdown uses Glamour to render markdown content with terminal styling
func renderMarkdown(content string, width int) string {
	if strings.TrimSpace(content) == "" {
		return "No content available"
	}

	// Create a Glamour renderer with terminal width
	r, err := glamour.NewTermRenderer(
		glamour.WithWordWrap(width),
		glamour.WithStandardStyle("dark"),
	)
	if err != nil {
		// Fallback to plain content if renderer fails
		return content
	}

	// Render the markdown
	rendered, err := r.Render(content)
	if err != nil {
		// Fallback to plain content if rendering fails
		return content
	}

	return rendered
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
// extractPreview extracts a clean preview from markdown content
func extractPreview(content string, maxLength int) string {
	if strings.TrimSpace(content) == "" {
		return "No content"
	}

	// Remove markdown formatting for cleaner preview
	preview := content

	// Remove headings (# ## ### etc)
	headingPattern := regexp.MustCompile(`^#{1,6}\s+`)
	preview = headingPattern.ReplaceAllString(preview, "")

	// Remove bold/italic markers (**text**, *text*)
	boldPattern := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	preview = boldPattern.ReplaceAllString(preview, "$1")

	italicPattern := regexp.MustCompile(`\*([^*]+)\*`)
	preview = italicPattern.ReplaceAllString(preview, "$1")

	// Remove links [text](url) -> keep text only
	linkPattern := regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)
	preview = linkPattern.ReplaceAllString(preview, "$1")

	// Remove list markers (-, 1., etc.)
	listPattern := regexp.MustCompile(`^(\s*[-*+]|\s*\d+\.)\s+`)
	preview = listPattern.ReplaceAllString(preview, "")

	// Remove code blocks and inline code
	codePattern := regexp.MustCompile("`[^`]+`")
	preview = codePattern.ReplaceAllString(preview, "code")

	codeBlockPattern := regexp.MustCompile("```[^`]*```")
	preview = codeBlockPattern.ReplaceAllString(preview, "code block")

	// Split by paragraphs and get first meaningful content
	paragraphs := strings.Split(strings.TrimSpace(preview), "\n\n")
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph != "" && !strings.HasPrefix(paragraph, "#") {
			// This is a meaningful paragraph
			if len(paragraph) <= maxLength-3 {
				return paragraph
			} else {
				return paragraph[:maxLength-3] + "..."
			}
		}
	}

	// Fallback: return first few lines
	lines := strings.Split(strings.TrimSpace(preview), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			if len(line) <= maxLength-3 {
				return line
			} else {
				return line[:maxLength-3] + "..."
			}
		}
	}

	return "No preview available"
}

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

		// Extract clean preview from markdown content
		preview := extractPreview(item.Content, m.previewWidth)

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
