// Package tui is the Bubble Tea viewer for tracked focus time.
package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"monique/internal/domain"
	"monique/internal/stats"
)

const refreshInterval = time.Second

type refreshMsg struct {
	totals []domain.AppTotal
	err    error
}

type Model struct {
	stats *stats.Store
	table table.Model
	err   error
}

func New(stats *stats.Store) Model {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "App", Width: 14},
			{Title: "Window", Width: 40},
			{Title: "Time today", Width: 12},
		}),
		table.WithFocused(false),
	)
	return Model{stats: stats, table: t}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.refresh(), tick())
}

func (m Model) refresh() tea.Cmd {
	return func() tea.Msg {
		totals, err := m.stats.Today(context.Background())
		return refreshMsg{totals: totals, err: err}
	}
}

func tick() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

type tickMsg struct{}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tickMsg:
		return m, tea.Batch(m.refresh(), tick())
	case refreshMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		rows := make([]table.Row, 0, len(msg.totals))
		for _, t := range msg.totals {
			rows = append(rows, table.Row{t.AppClass, t.Title, formatDuration(t.DurationSeconds)})
		}
		m.table.SetRows(rows)
		return m, nil
	}
	return m, nil
}

func (m Model) View() string {
	view := titleStyle.Render("monique — time today") + "\n" + m.table.View()
	if m.err != nil {
		view += "\n" + m.err.Error()
	}
	return view + helpStyle.Render("q: quit")
}

func formatDuration(seconds int64) string {
	d := time.Duration(seconds) * time.Second
	h := int(d.Hours())
	mnt := int(d.Minutes()) % 60
	sec := int(d.Seconds()) % 60
	return fmt.Sprintf("%dh%02dm%02ds", h, mnt, sec)
}
