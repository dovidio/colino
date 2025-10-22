package tui

import (
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/charmbracelet/lipgloss/table"
	"golino/internal/colinodb"
)

type tablePage struct {
	items []colinodb.Content
	table *table.Table

	// TODO: some of these properties maybe don't need to stay in the model
	ready        bool
	cursor       int
	currentPage  int
	totalPages   int
	tableWidth   int
	tableHeight  int
	urlWidth     int
	titleWidth   int
	authorWidth  int
	dateWidth    int
	previewWidth int
	pageSize     int
}

func TablePage(items []colinodb.Content, cursor int, pageSize int, currentPage int) tablePage {
	return tablePage{
		items:       items,
		cursor:      cursor,
		pageSize:    pageSize,
		currentPage: currentPage,
	}
}

func (m tablePage) Init() tea.Cmd {
	return nil
}

func (m tablePage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case " ":
			if len(m.items) > 0 {
				globalCursor := m.currentPage*m.pageSize + m.cursor
				if globalCursor < len(m.items) {
					selectedItem := &m.items[globalCursor]
					return m, func() tea.Msg { return goToDetailMsg{item: contentToArticleDetail(selectedItem)} }
				}
				return m, nil
			}
		case "2":
			return m, func() tea.Msg { return goToSearchMsg{} }
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
				return m, tea.ClearScreen // Force screen refresh to fix border rendering
			}
			return m, nil
		case "h": // Previous page
			if m.currentPage > 0 {
				m.currentPage--
				m.cursor = 0
				m.updateTableRows()
				return m, tea.ClearScreen // Force screen refresh to fix border rendering
			}
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.tableWidth = msg.Width - 2
		m.tableHeight = msg.Height
		m.configureTable(msg.Width, msg.Height-4) // Leave room for borders/title
		m.ready = true

		return m, tea.ClearScreen
	}

	return m, nil
}

func (m tablePage) View() string {
	return m.renderTableView()
}

func (m tablePage) renderTableView() string {
	if !m.ready {
		return "...Loading"
	}

	if len(m.items) == 0 {
		return "No content found in the database"
	}

	menu := renderMenu(0, m.tableWidth)

	// Wrap table in blue border
	tableContainer := m.table.Render()

	// Help info
	helpInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Align(lipgloss.Center).
		Render("j/k: move • l/h: page • g/G: home/end • Space: view details • q: quit")

	return pageLayout("Articles", lipgloss.JoinVertical(lipgloss.Left, menu, tableContainer, helpInfo))
}

func (m *tablePage) updateTableRows() {
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
		title := extractTitleM(item.Metadata)
		if title == "" {
			title = "No title"
		}

		author := item.AuthorUsername
		if author == "" {
			author = "Unknown author"
		}

		// Extract clean preview from markdown content
		contentType := extractContentTypeM(item.Metadata)
		preview := extractPreview(item.Content, m.previewWidth, contentType)
		strings.ReplaceAll(preview, "\n", " ")

		row := []string{
			truncateString(title, m.titleWidth),
			truncateString(author, m.authorWidth),
			truncateString(item.CreatedAt.Format("2006-01-02"), m.dateWidth),
			truncateString(strings.ReplaceAll(preview, "\n", ""), m.previewWidth),
		}
		rows = append(rows, row)
	}

	// Ensure cursor is within valid bounds for the current page
	itemsOnCurrentPage := len(rows)
	if itemsOnCurrentPage > 0 {
		if m.cursor >= itemsOnCurrentPage {
			m.cursor = itemsOnCurrentPage - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
	}

	// Define colors
	lightBlue := lightBlue()
	darkBlue := darkBlue()

	borderStyle := lipgloss.NewStyle().Foreground(darkBlue)

	headerStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true).
		Foreground(darkBlue).
		Align(lipgloss.Center)

	newTable := table.New().
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
				return lipgloss.NewStyle().
					Padding(0, 1).
					Background(lightBlue).
					Foreground(lipgloss.Color("0"))
			}
			return lipgloss.NewStyle().Padding(0, 1)
		})

	m.table = newTable
}

// configureTable sets up the table with dynamic column widths based on available space
func (m *tablePage) configureTable(width, height int) {
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

// extractPreview extracts a clean preview from markdown content
func extractPreview(content string, maxLength int, contentType string) string {
	// Handle PDF content type specially
	if strings.Contains(contentType, "pdf") || strings.Contains(contentType, "application/pdf") {
		return handlePdf(content)[:10]
	}

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
	paragraphs := strings.SplitSeq(strings.TrimSpace(preview), "\n\n")
	for paragraph := range paragraphs {
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

	// Fallback: returns only the first few lines
	lines := strings.SplitSeq(strings.TrimSpace(preview), "\n")
	for line := range lines {
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

// handlePdf
func handlePdf(content string) string {
	return "PDF document - content preview not available"
}
