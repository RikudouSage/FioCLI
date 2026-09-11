package tui

import "github.com/charmbracelet/lipgloss"

var (
	primaryColor = lipgloss.AdaptiveColor{Light: "#075E54", Dark: "#5EE1B7"}
	mutedColor   = lipgloss.AdaptiveColor{Light: "#667085", Dark: "#8B95A5"}
	faintColor   = lipgloss.AdaptiveColor{Light: "#98A2B3", Dark: "#667085"}
	pendingColor = lipgloss.AdaptiveColor{Light: "#9A6700", Dark: "#E3B341"}
	inColor      = lipgloss.AdaptiveColor{Light: "#08783E", Dark: "#57D68D"}
	outColor     = lipgloss.AdaptiveColor{Light: "#B42318", Dark: "#FF7B72"}

	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
	labelStyle   = lipgloss.NewStyle().Foreground(mutedColor)
	helpStyle    = lipgloss.NewStyle().Foreground(mutedColor)
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(primaryColor).MarginTop(1)
	cardStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(mutedColor).Padding(0, 2)
)
