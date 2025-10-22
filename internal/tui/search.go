package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	textinput "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"golino/internal/colinodb"
	"golino/internal/config"
	"golino/internal/digest"
)

type searchPage struct {
	width       int
	height      int
	err         error
	searchInput textinput.Model
}

func (m searchPage) Init() tea.Cmd {
	return nil
}

func (m searchPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			if m.searchInput.Focused() {
				return m, m.showContent()
			}
		}

		if msg.Type == tea.KeyTab {
			if !m.searchInput.Focused() {
				m.searchInput.Focus()
			}
		}
		switch msg.String() {
		case "esc":
			if m.searchInput.Focused() {
				m.searchInput.Blur()
			} else {
				return m, tea.Quit
			}
		case "1":
			if !m.searchInput.Focused() {
				return m, func() tea.Msg { return goToTableMsg{} }
			}
		default:
			updated, cmd := m.searchInput.Update(msg)
			m.searchInput = updated
			return m, cmd
		}
	case goToSearchMsg:
		if m.searchInput.Value() == "" {
			m.searchInput = initializeInput()
		}
		m.searchInput.Focus()
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func initializeInput() textinput.Model {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "https://norvig.com/21-days.html"
	input.PlaceholderStyle.Width(40)
	input.Width = 50

	return input
}

func (m *searchPage) showContent() tea.Cmd {
	if strings.TrimSpace(m.searchInput.Value()) == "" {
		m.err = errors.New("Please enter a valid url")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	content, err := getContentFromCache(ctx, m.searchInput.Value())
	if content != nil {
		return func() tea.Msg {
			return goToDetailMsg{item: contentToArticleDetail(content)}
		}
	}

	appConfig, err := config.LoadAppConfig()
	if err != nil {
		m.err = err
		return nil
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	article, err := digest.GetFreshContent(ctx, appConfig, m.searchInput.Value())
	if err != nil {
		m.err = err
		return nil
	}

	if article != nil {
		return func() tea.Msg {
			return goToDetailMsg{item: &articleDetail{content: article.Content, url: m.searchInput.Value()}}
		}
	}

	return nil
}

func getContentFromCache(ctx context.Context, url string) (*colinodb.Content, error) {
	dbPath, err := config.LoadDBPath()
	if err != nil {
		return nil, err
	}
	db, err := colinodb.Open(dbPath)
	if err != nil {
		return nil, err
	}
	content, err := colinodb.GetByURL(ctx, db, url)
	if err != nil {
		return nil, err
	}
	if content == nil || content.ID == "" {
		return nil, fmt.Errorf("No content found in cache")
	}

	return content, nil
}

func (m searchPage) View() string {
	instructions := lipgloss.NewStyle().
		MarginTop(min(m.height/4, 10)).
		MarginBottom(2).
		Render("Enter a url below to fetch content from any article/video on the internet")
	var borderColor lipgloss.Color
	borderColor = lipgloss.Color("8")
	if m.searchInput.Focused() {
		borderColor = lipgloss.Color("15")
	}

	input := lipgloss.NewStyle().
		Width(50).
		AlignHorizontal(lipgloss.Left).
		Border(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Render(m.searchInput.View())

	var helpInfo string
	if m.searchInput.Focused() {
		helpInfo = helpBar([]string{
			"Enter: browse url",
			"Esc: unfocus search input",
		})

	} else {
		helpInfo = helpBar([]string{
			"1: go to table view",
			"Tab: focus search input",
			"Esc: quit colino",
		})
	}

	var error string
	if m.err != nil {
		error = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Render(fmt.Sprintf("Error while fetching content: %v", m.err))
	}

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		renderMenu(1, m.width),
		instructions,
		input,
		error,
		lipgloss.NewStyle().MarginTop(2).Render(helpInfo),
	)

	return pageLayout("Search", content)
}
