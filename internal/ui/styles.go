package ui

import "github.com/charmbracelet/lipgloss"

// tokyo-night 寄りの配色
var (
	colAccent = lipgloss.Color("#7aa2f7")
	colGreen  = lipgloss.Color("#9ece6a")
	colYellow = lipgloss.Color("#e0af68")
	colRed    = lipgloss.Color("#f7768e")
	colPurple = lipgloss.Color("#bb9af7")
	colSubtle = lipgloss.AdaptiveColor{Light: "#9699a3", Dark: "#565f89"}
	colText   = lipgloss.AdaptiveColor{Light: "#343b58", Dark: "#c0caf5"}
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#1a1b26")).
			Background(colAccent).
			Padding(0, 1)

	subtleStyle = lipgloss.NewStyle().Foreground(colSubtle)
	textStyle   = lipgloss.NewStyle().Foreground(colText)
	headerStyle = lipgloss.NewStyle().Foreground(colSubtle).Bold(true)

	cursorStyle   = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	selectedStyle = lipgloss.NewStyle().Foreground(colPurple).Bold(true)

	okStyle   = lipgloss.NewStyle().Foreground(colGreen)
	warnStyle = lipgloss.NewStyle().Foreground(colYellow)
	errStyle  = lipgloss.NewStyle().Foreground(colRed)
	skipStyle = lipgloss.NewStyle().Foreground(colSubtle)
	infoStyle = lipgloss.NewStyle().Foreground(colAccent)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colSubtle).
			Padding(0, 1)

	confirmBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colRed).
			Padding(0, 1)

	keyStyle  = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	descStyle = lipgloss.NewStyle().Foreground(colSubtle)

	diffAddStyle  = lipgloss.NewStyle().Foreground(colGreen)
	diffDelStyle  = lipgloss.NewStyle().Foreground(colRed)
	diffHunkStyle = lipgloss.NewStyle().Foreground(colPurple)
	diffFileStyle = lipgloss.NewStyle().Foreground(colText).Bold(true)

	tabStyle = lipgloss.NewStyle().
			Foreground(colSubtle).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(colSubtle)

	tabActiveStyle = lipgloss.NewStyle().
			Foreground(colAccent).
			Bold(true).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(colAccent)
)
