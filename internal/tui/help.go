package tui

import (
	"github.com/charmbracelet/lipgloss"
)

func helpBar(items []string) string {
	var content string

	for index, item := range items {
		content = content + item
		if index != len(items)-1 {
			content = content + " • "
		}
	}

	helpBar := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Align(lipgloss.Center).
		Render(content)

	return helpBar
}
