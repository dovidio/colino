package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"golino/internal/colinodb"
	"golino/internal/config"
)

type viewMode int

const (
	tableView viewMode = iota
	searchView
	detailView
)

// Navigation messages
type goToDetailMsg struct {
	item *articleDetail
}
type goToSearchMsg struct{}
type goToTableMsg struct{}

type rootPage struct {
	viewMode   viewMode
	detailPage detailPage
	tablePage  tablePage
	searchPage searchPage
	width      int
	height     int
	err        error
}

type articleDetail struct {
	content        string
	metadata       string
	url            string
	createdAt      time.Time
	authorUsername string
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

	tablePage := TablePage(
		rows,
		0,
		10,
		0,
	)

	m := rootPage{
		tablePage: tablePage,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func (m rootPage) Init() tea.Cmd {
	return nil
}

func (m rootPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.viewMode {
	case tableView:
		m.tablePage, cmd = update[tablePage](m.tablePage, msg)
	case detailView:
		m.detailPage, cmd = update[detailPage](m.detailPage, msg)
	case searchView:
		m.searchPage, cmd = update[searchPage](m.searchPage, msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
	case goToSearchMsg:
		m.viewMode = searchView
		m.searchPage, cmd = update[searchPage](m.searchPage, msg)
	case goToTableMsg:
		m.viewMode = tableView
	case goToDetailMsg:
		m.viewMode = detailView
		m.detailPage, cmd = update[detailPage](m.detailPage, msg)
	case tea.WindowSizeMsg:
		var cmds []tea.Cmd

		m.tablePage, cmd = update[tablePage](m.tablePage, msg)
		cmds = append(cmds, cmd)

		m.detailPage, cmd = update[detailPage](m.detailPage, msg)
		cmds = append(cmds, cmd)

		m.searchPage, cmd = update[searchPage](m.searchPage, msg)
		cmds = append(cmds, cmd)

		m.width = msg.Width - 4
		m.height = msg.Height - 4

		return m, tea.Batch(cmds...)
	}

	return m, cmd
}

func (m rootPage) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}

	switch m.viewMode {
	case detailView:
		return m.detailPage.View()
	case searchView:
		return m.searchPage.View()
	case tableView:
		return m.tablePage.View()
	default:
		return "Unknown View"
	}
}

func update[T any](model tea.Model, msg tea.Msg) (T, tea.Cmd) {
	newModel, cmd := model.Update(msg)
	return newModel.(T), cmd
}
