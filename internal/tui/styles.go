package tui

import "github.com/charmbracelet/lipgloss"

// All lipgloss styles for the TUI live here so layout tweaks don't
// require hunting through model.go.
var (
	borderColor = lipgloss.Color("240")

	tableBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(borderColor)

	previewHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("252")).
				Padding(0, 1)

	previewBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(borderColor).
				Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 1)

	filterPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Padding(0, 1)
)
