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
	"github.com/charmbracelet/lipgloss"

	"monique/internal/domain"
)

const (
	refreshInterval = time.Second
	barMaxWidth     = 40
	recentLimit     = 100
)

// Eighth-block ramp: bar widths encode sub-cell precision so small daily
// differences stay visible instead of rounding to nothing.
var eighths = []rune{'▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}

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
	repo     domain.Repository
	table    table.Model
	viewport viewport.Model // scrolls the week chart (up/down keys)
	view     view
	days     int  // chart range: 7 or 30
	weekJump bool // jump chart to today (bottom) on next content set
	width    int
	week     []domain.DayTotal
	err      error
}

// New builds the model. When chart is true it opens on the weekly chart
// instead of the live activity log.
func New(repo domain.Repository, chart bool) Model {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "STARTED", Width: 9},
			{Title: "APP", Width: 14},
			{Title: "WINDOW", Width: 40},
			{Title: "FOR", Width: 10},
		}),
		table.WithFocused(true), // focused so ↑/↓/PgUp/PgDn scroll the log
		table.WithHeight(20),
		table.WithStyles(liveTableStyles()),
	)
	m := Model{repo: repo, table: t, days: 7, viewport: viewport.New(0, 0)}
	if chart {
		m.view = viewWeek
		m.weekJump = true // land on today once the first data arrives
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.refresh(), tick())
}

// refresh fetches data for both views in one command; the queries are cheap
// and it keeps the update logic to a single message type.
func (m Model) refresh() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		now := time.Now()
		events, err := m.repo.RecentEvents(ctx, now.Unix(), recentLimit)
		if err != nil {
			return refreshMsg{err: err}
		}
		week, err := lastDays(ctx, m.repo, m.days, now)
		return refreshMsg{events: events, week: week, err: err}
	}
}

// lastDays returns total focused time per day for the last n days (today
// included), one entry per day even when nothing was tracked that day.
func lastDays(ctx context.Context, repo domain.Repository, n int, now time.Time) ([]domain.DayTotal, error) {
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(n - 1))
	rows, err := repo.TimeByDay(ctx, start.Unix(), now.Unix())
	if err != nil {
		return nil, err
	}
	byDate := make(map[string]domain.DayTotal, len(rows))
	for _, r := range rows {
		byDate[r.Date] = r
	}
	out := make([]domain.DayTotal, n)
	for i := range out {
		d := start.AddDate(0, 0, i).Format("2006-01-02")
		day := byDate[d] // zero value for days with no data
		day.Date = d
		out[i] = day
	}
	return out, nil
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
		case "c":
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
		m.width = msg.Width
		// Reserve rows for the header (title + rule), footer help, and padding.
		if h := msg.Height - 6; h > 0 {
			m.viewport.Width = msg.Width
			m.viewport.Height = h
			m.table.SetHeight(h)
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
			started := " " + time.Unix(e.StartedAt, 0).Format("15:04:05")
			if e.Open {
				started = liveDotStyle.Render("● ") + time.Unix(e.StartedAt, 0).Format("15:04:05")
			}
			rows = append(rows, table.Row{started, e.AppClass, e.Title, formatDuration(e.DurationSeconds)})
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

	// Remaining messages (arrow keys, page up/down) scroll the active view.
	var cmd tea.Cmd
	if m.view == viewWeek {
		m.viewport, cmd = m.viewport.Update(msg)
	} else {
		m.table, cmd = m.table.Update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	var (
		label, subtitle, body string
	)
	switch m.view {
	case viewWeek:
		label, subtitle = "week", fmt.Sprintf("last %d days", m.days)
		body = m.viewport.View()
	default:
		label = "live"
		body = m.table.View()
	}
	out := m.header(label, subtitle) + "\n" + body
	if m.err != nil {
		out += "\n" + errStyle.Render(m.err.Error())
	}
	return out + "\n" + m.help()
}

// header lays out the top strip: brand · view label, then a rule, with the
// current day's total pushed to the right so the numbers are visible from
// either view without extra keystrokes.
func (m Model) header(label, subtitle string) string {
	left := brandStyle.Render("monique") + eyebrowStyle.Render("· "+label)
	if subtitle != "" {
		left += "  " + subtitleStyle.Render(subtitle)
	}
	right := m.todayLine()

	width := m.width
	if width == 0 {
		width = 80
	}
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	top := left + strings.Repeat(" ", gap) + right
	rule := ruleStyle.Render(strings.Repeat("─", width))
	return top + "\n" + rule
}

func (m Model) todayLine() string {
	if len(m.week) == 0 {
		return ""
	}
	today := m.week[len(m.week)-1]
	if today.DurationSeconds == 0 {
		return mutedStyle.Render("today  ") + accentStyle.Render("—")
	}
	s := mutedStyle.Render("today  ") + accentStyle.Render(formatDuration(today.DurationSeconds))
	if today.TopApp != "" {
		s += mutedStyle.Render("   top  ") + accentStyle.Render(today.TopApp) +
			mutedStyle.Render(" "+formatDuration(today.TopAppSeconds))
	}
	return s + " "
}

func (m Model) help() string {
	k := helpKeyStyle.Render
	parts := []string{
		k("l") + " live",
		k("c") + " chart (again: 7/30)",
		k("↑↓") + " scroll",
		k("q") + " quit",
	}
	return helpStyle.Render(strings.Join(parts, "   "))
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
		bar := renderBar(d.DurationSeconds, max)
		top := ""
		if d.TopApp != "" {
			top = mutedStyle.Render(fmt.Sprintf("   %s %s", d.TopApp, formatDuration(d.TopAppSeconds)))
		}
		fmt.Fprintf(&b, " %s  %s  %s%s\n",
			mutedStyle.Render(day.Format("Mon 02")),
			bar,
			accentStyle.Render(formatDuration(d.DurationSeconds)),
			top)
	}
	return b.String()
}

// renderBar draws a bar of width proportional to value/max with eighth-block
// precision, so a day that's 3% of the peak still gets a visible sliver
// rather than rounding to blank.
func renderBar(value, max int64) string {
	if max == 0 || value == 0 {
		return barStyle.Render(strings.Repeat(" ", barMaxWidth))
	}
	// Total width in eighths.
	units := value * int64(barMaxWidth) * 8 / max
	if units == 0 {
		units = 1 // one eighth minimum for any non-zero day
	}
	full := int(units / 8)
	rem := int(units % 8)
	var b strings.Builder
	b.WriteString(strings.Repeat("█", full))
	if rem > 0 && full < barMaxWidth {
		b.WriteRune(eighths[rem-1])
		full++
	}
	if full < barMaxWidth {
		b.WriteString(strings.Repeat(" ", barMaxWidth-full))
	}
	return barStyle.Render(b.String())
}

func formatDuration(seconds int64) string {
	d := time.Duration(seconds) * time.Second
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
