package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	helpStyle  = lipgloss.NewStyle().Faint(true).Padding(1, 1, 0, 1)
	barStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("250")) // darker white, matches the scroll highlight
)

// liveTableStyles renders the selected (scroll-highlighted) row as a
// darker-white bar instead of the default bright purple.
func liveTableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("235")). // dark text
		Background(lipgloss.Color("250")). // darker white
		Bold(false)
	return s
}
