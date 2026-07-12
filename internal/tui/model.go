// Package tui is the Bubble Tea viewer for tracked focus time.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"monique/internal/domain"
	"monique/internal/stats"
)

const (
	refreshInterval = time.Second
	barMaxWidth     = 40
)

type view int

const (
	viewLive view = iota
	viewWeek
)

type refreshMsg struct {
	events []domain.EventRow
	week   []domain.DayTotal
	err    error
}

type tickMsg struct{}

type Model struct {
	stats    *stats.Store
	table    table.Model
	viewport viewport.Model // scrolls the week chart (up/down keys)
	view     view
	days     int  // chart range: 7 or 30
	weekJump bool // jump chart to today (bottom) on next content set
	week     []domain.DayTotal
	err      error
}

func New(stats *stats.Store) Model {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "Started", Width: 8},
			{Title: "App", Width: 14},
			{Title: "Window", Width: 40},
			{Title: "For", Width: 10},
		}),
		table.WithFocused(false),
	)
	return Model{stats: stats, table: t, days: 7, viewport: viewport.New(0, 0)}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.refresh(), tick())
}

// refresh fetches data for both views in one command; the queries are cheap
// and it keeps the update logic to a single message type.
func (m Model) refresh() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		events, err := m.stats.Recent(ctx, 100)
		if err != nil {
			return refreshMsg{err: err}
		}
		week, err := m.stats.LastDays(ctx, m.days)
		return refreshMsg{events: events, week: week, err: err}
	}
}

func tick() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "a":
			if m.view == viewWeek { // already on the chart: toggle range
				if m.days == 7 {
					m.days = 30
				} else {
					m.days = 7
				}
			}
			m.view = viewWeek
			m.weekJump = true // land on today once the new content is set
			return m, m.refresh()
		case "l":
			m.view = viewLive
			return m, nil
		}
	case tea.WindowSizeMsg:
		// Reserve rows for the title, help line and a little padding.
		m.viewport.Width = msg.Width
		if h := msg.Height - 4; h > 0 {
			m.viewport.Height = h
		}
	case tickMsg:
		return m, tea.Batch(m.refresh(), tick())
	case refreshMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		rows := make([]table.Row, 0, len(msg.events))
		for _, e := range msg.events {
			started := time.Unix(e.StartedAt, 0).Format("15:04:05")
			dur := formatDuration(e.DurationSeconds)
			if e.Open {
				started = "▶ " + time.Unix(e.StartedAt, 0).Format("15:04")
			}
			rows = append(rows, table.Row{started, e.AppClass, e.Title, dur})
		}
		m.table.SetRows(rows)
		m.week = msg.week
		if m.view == viewWeek {
			m.viewport.SetContent(m.weekView())
			if m.weekJump { // preserve manual scroll on plain refreshes
				m.viewport.GotoBottom()
				m.weekJump = false
			}
		}
		return m, nil
	}

	// Remaining messages (arrow keys, page up/down) scroll the chart.
	if m.view == viewWeek {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) View() string {
	var body string
	switch m.view {
	case viewWeek:
		body = titleStyle.Render(fmt.Sprintf("monique — last %d days", m.days)) + "\n" + m.viewport.View()
	default:
		body = titleStyle.Render("monique — live activity") + "\n" + m.table.View()
	}
	if m.err != nil {
		body += "\n" + m.err.Error()
	}
	return body + helpStyle.Render("a: chart (again: 7/30 days) · ↑/↓ scroll · l: live · q: quit")
}

func (m Model) weekView() string {
	var max int64
	for _, d := range m.week {
		if d.DurationSeconds > max {
			max = d.DurationSeconds
		}
	}
	var b strings.Builder
	for _, d := range m.week {
		day, _ := time.Parse("2006-01-02", d.Date)
		width := 0
		if max > 0 {
			width = int(d.DurationSeconds * barMaxWidth / max)
		}
		if width == 0 && d.DurationSeconds > 0 {
			width = 1 // visible sliver for non-zero days
		}
		top := ""
		if d.TopApp != "" {
			top = fmt.Sprintf("  %s (%s)", d.TopApp, formatDuration(d.TopAppSeconds))
		}
		fmt.Fprintf(&b, " %s  %s %s%s\n",
			day.Format("Mon 02"),
			barStyle.Render(strings.Repeat("█", width)),
			formatDuration(d.DurationSeconds),
			top)
	}
	return b.String()
}

func formatDuration(seconds int64) string {
	d := time.Duration(seconds) * time.Second
	h := int(d.Hours())
	mnt := int(d.Minutes()) % 60
	sec := int(d.Seconds()) % 60
	return fmt.Sprintf("%dh%02dm%02ds", h, mnt, sec)
}
