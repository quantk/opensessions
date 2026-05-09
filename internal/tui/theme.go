package tui

import "github.com/charmbracelet/lipgloss"

const (
	nord0  = "#2E3440"
	nord1  = "#3B4252"
	nord2  = "#434C5E"
	nord3  = "#4C566A"
	nord4  = "#D8DEE9"
	nord5  = "#E5E9F0"
	nord6  = "#ECEFF4"
	nord7  = "#8FBCBB"
	nord8  = "#88C0D0"
	nord9  = "#81A1C1"
	nord10 = "#5E81AC"
	nord11 = "#BF616A"
	nord12 = "#D08770"
	nord13 = "#EBCB8B"
	nord14 = "#A3BE8C"
	nord15 = "#B48EAD"
)

var (
	titleStyle          = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(nord8))
	modeStyle           = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(nord15))
	accentStyle         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(nord8))
	selectedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color(nord6)).Background(lipgloss.Color(nord1))
	activeToolStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color(nord6)).Background(lipgloss.Color(nord2))
	userStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color(nord14))
	assistantStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color(nord9))
	toolStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color(nord12))
	dimStyle            = lipgloss.NewStyle().Foreground(lipgloss.Color(nord3))
	warnStyle           = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(nord13))
	errorStyle          = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(nord11))
	successStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color(nord14))
	focusRailStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(nord8))
	focusRailQuietStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(nord3))
	sourceBadgeStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(nord6)).Background(lipgloss.Color(nord2))
	detailHeadingStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(nord8))
	detailLabelStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(nord7))
)

func mdNord(color string) *string {
	return &color
}
