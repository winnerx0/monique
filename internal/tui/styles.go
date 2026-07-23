package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// Single accent (amber) reserved for "now": the live-row dot and today's
// totals. Everything else stays in grayscale so the accent actually reads
// as a marker for the present moment.
var (
	accent = lipgloss.Color("215")
	muted  = lipgloss.Color("244")
	bar    = lipgloss.Color("250")
	dark   = lipgloss.Color("235")

	brandStyle    = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	eyebrowStyle  = lipgloss.NewStyle().Foreground(muted)
	subtitleStyle = lipgloss.NewStyle().Foreground(muted).Italic(true)
	mutedStyle    = lipgloss.NewStyle().Foreground(muted)
	accentStyle   = lipgloss.NewStyle().Foreground(accent).Bold(true)
	liveDotStyle  = lipgloss.NewStyle().Foreground(accent)
	ruleStyle     = lipgloss.NewStyle().Foreground(muted).Faint(true)
	barStyle      = lipgloss.NewStyle().Foreground(bar)
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	helpStyle     = lipgloss.NewStyle().Foreground(muted).Padding(1, 1, 0, 1)
	helpKeyStyle  = lipgloss.NewStyle().Foreground(bar).Bold(true)
)

// liveTableStyles renders the selected (scroll-highlighted) row as a
// darker-white bar instead of the default bright purple.
func liveTableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		Foreground(muted).
		BorderForeground(muted).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(dark).
		Background(bar).
		Bold(false)
	return s
}
