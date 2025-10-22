package tui

import (
	"github.com/charmbracelet/lipgloss"
	"strconv"
	"strings"
)

type menuItem struct {
	label string
}

func pageLayout(pageTitle string, content string) string {
	return lipgloss.NewStyle().
		Padding(0, 1).
		Render(lipgloss.JoinVertical(lipgloss.Left, content))
}

func renderMenu(activeItem int, width int) string {
	divider := strings.Repeat("─", max(0, width))

	items := []menuItem{
		{
			label: "Table",
		},
		{
			label: "Search",
		},
	}

	styledItems := []string{}
	for index, item := range items {
		var style lipgloss.Style
		content := item.label + " [" + strconv.Itoa(index+1) + "]"
		if activeItem == index {
			style = lipgloss.NewStyle().Foreground(lipgloss.Color(strconv.Itoa(15))).Underline(true)
		} else {
			style = lipgloss.NewStyle().Foreground(lipgloss.Color(strconv.Itoa(8)))
		}

		fullContent := style.Render(content)
		if index != len(items)-1 {
			fullContent = fullContent + " | "
		}

		styledItems = append(styledItems, fullContent)
	}

	menu := lipgloss.JoinHorizontal(lipgloss.Left, styledItems...)

	return lipgloss.JoinVertical(lipgloss.Left, menu, divider)
}
