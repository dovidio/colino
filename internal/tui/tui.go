package tui

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"golino/internal/colinodb"
	"golino/internal/config"
)

type model struct {
	items    []colinodb.Content
	table    *Table
	ready    bool
	err      error
}

type contentRow struct {
	ID      string
	Title   string
	Author  string
	Source  string
	Date    string
	Preview string
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

	since := time.Now().Add(-24 * time.Hour) // Default to last 24 hours
	rows, err := colinodb.GetSince(ctx, db, since, "", 0)
	if err != nil {
		return fmt.Errorf("query failed while reading from the Colino database: %w", err)
	}

	table := NewTable()
	m := model{
		items: rows,
		table: table,
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
		case "up", "k":
			m.table.MoveUp(1)
			return m, nil
		case "down", "j":
			m.table.MoveDown(1)
			return m, nil
		case "pgup":
			m.table.MoveUp(5)
			return m, nil
		case "pgdown":
			m.table.MoveDown(5)
			return m, nil
		case "home", "g":
			m.table.GotoTop()
			return m, nil
		case "end", "G":
			m.table.GotoBottom()
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.table.SetWidth(msg.Width)
		m.table.SetHeight(msg.Height - 4) // Leave room for borders/title
		m.ready = true
		return m, nil
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
		return "No content found in the last 24 hours."
	}

	// Convert colinodb.Content to table rows
	rows := make([]contentRow, len(m.items))
	for i, item := range m.items {
		title := extractTitle(item.Metadata)
		if title == "" {
			title = "No title"
		}

		author := item.AuthorUsername
		if author == "" {
			author = "Unknown author"
		}

		preview := item.Content
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}

		rows[i] = contentRow{
			ID:      shortenID(item.ID),
			Title:   title,
			Author:  author,
			Source:  item.Source,
			Date:    item.CreatedAt.Format("2006-01-02"),
			Preview: preview,
		}
	}

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true).
		Align(lipgloss.Center).
		MarginBottom(1).
		Render("Recent Articles")

	table := m.table.View(rows)
	return lipgloss.JoinVertical(lipgloss.Left, title, table)
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

func shortenID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}