package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorAccent = lipgloss.Color("#7D56F4")
	colorMuted  = lipgloss.Color("240")
	colorOK     = lipgloss.Color("42")
	colorErr    = lipgloss.Color("196")

	sidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(0, 1)

	sidebarSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("229")).
				Background(colorAccent).
				Bold(true)

	mainStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(0, 1)

	mainFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorAccent).
				Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Background(lipgloss.Color("235")).
			Padding(0, 1)

	methodStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)

	okStyle  = lipgloss.NewStyle().Bold(true).Foreground(colorOK)
	errStyle = lipgloss.NewStyle().Bold(true).Foreground(colorErr)

	dimStyle = lipgloss.NewStyle().Foreground(colorMuted)

	paramsBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(0, 1)

	paramNameStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	paramRowFocusedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("229")).
				Background(colorAccent).
				Bold(true)
)
