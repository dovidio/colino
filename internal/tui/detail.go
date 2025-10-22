package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type detailPage struct {
	width        int
	height       int
	viewport     viewport.Model
	selectedItem *articleDetail
}

func (m detailPage) Init() tea.Cmd {
	return nil
}

func (m detailPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, func() tea.Msg { return goToTableMsg{} }
		case "k":
			m.viewport.ScrollUp(1)
			return m, nil
		case "j":
			m.viewport.ScrollDown(1)
			return m, nil
		case "g":
			m.viewport.GotoTop()
			return m, nil
		case "G":
			m.viewport.GotoBottom()
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width - 4
		m.height = msg.Height - 4
		if m.selectedItem != nil {
			m.viewport = setupViewport(m.width, m.height, m.selectedItem)
		}

		return m, nil
	case goToDetailMsg:
		m.selectedItem = msg.item
		m.viewport = setupViewport(m.width, m.height, m.selectedItem)

		return m, nil
	}

	return m, nil
}

func (m detailPage) View() string {
	if m.selectedItem == nil {
		return "No item selected"
	}

	// Extract title
	title := extractTitle(m.selectedItem.metadata)
	if title == "" {
		title = "No title"
	}

	// Define colors
	lightBlue := lightBlue()
	darkBlue := darkBlue()
	borderColor := darkBlue

	// Border styling
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(borderColor)

	// Title styling
	titleStyle := lipgloss.NewStyle().
		Foreground(darkBlue).
		Bold(true).
		Align(lipgloss.Left).
		MarginBottom(1).
		Width(m.width - 8)

	// URL styling
	urlStyle := lipgloss.NewStyle().
		Foreground(lightBlue).
		Italic(true).
		MarginBottom(1).
		Width(m.width - 8)

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
	if m.selectedItem.url != "" {
		urlRendered = urlStyle.Render("URL: " + m.selectedItem.url)
	} else {
		urlRendered = urlStyle.Render("URL: Not available")
	}

	// Render metadata
	author := m.selectedItem.authorUsername
	if author == "" {
		author = "Unknown author"
	}
	metadataRendered := metadataStyle.Render(fmt.Sprintf("Author: %s • Date: %s",
		author, m.selectedItem.createdAt.Format("2006-01-02 15:04")))

	// Calculate viewport height (leave space for title, URL, metadata, scroll info, help)
	viewportHeight := m.height - 10
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

	return pageLayout(titleRendered, borderStyle.Render(content))
}

func setupViewport(width, height int, selectedItem *articleDetail) viewport.Model {
	// Calculate content width (account for borders and padding)
	contentWidth := width
	if contentWidth < 20 {
		contentWidth = 20
	}

	// Check content type to handle PDFs specially
	contentType := extractContentType(selectedItem.metadata)
	var renderedContent string

	if strings.Contains(contentType, "pdf") || strings.Contains(contentType, "application/pdf") {
		renderedContent = handlePdf(selectedItem.content)
	} else {
		// Render markdown content using Glow
		renderedContent = renderMarkdown(selectedItem.content, contentWidth)
	}

	// Calculate viewport height (leave space for title, URL, metadata, scroll info, help)
	viewportHeight := height - 10
	if viewportHeight < 5 {
		viewportHeight = 5
	}

	// Initialize viewport with rendered content
	viewport := viewport.New(contentWidth, viewportHeight)
	viewport.SetContent(renderedContent)

	return viewport
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
