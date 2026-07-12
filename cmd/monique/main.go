// Command monique tracks focused windows (via Hyprland) into SQLite and
// shows a live Bubble Tea view of where the time went.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"monique/internal/collector"
	"monique/internal/stats"
	"monique/internal/storage"
	"monique/internal/tracker"
	"monique/internal/tui"
)

func dbPath() (string, error) {
	home, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "monique")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "monique.db"), nil
}

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	path, err := dbPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "monique:", err)
		os.Exit(1)
	}

	db, err := storage.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "monique:", err)
		os.Exit(1)
	}
	defer db.Close()

	switch cmd {
	case "":
		runBoth(db)
	case "track":
		runTrack(db)
	case "ui":
		runUI(db, false)
	case "chart":
		runUI(db, true)
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: monique [track|ui|chart]  (no args: track + ui together)")
}

// runBoth tracks and shows the UI in one process. Don't run this alongside a
// separate `monique track` — two trackers fight over the open session.
func runBoth(db *storage.SQLite) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	trackErr := make(chan error, 1)
	go func() {
		trackErr <- tracker.New(collector.New(), db).Run(ctx)
	}()

	m := tui.New(stats.New(db), false)
	_, uiErr := tea.NewProgram(m, tea.WithAltScreen()).Run()

	cancel()          // stop the tracker; it closes the open session on its way out
	err := <-trackErr // wait so the final UPDATE lands before db.Close
	if uiErr != nil {
		fmt.Fprintln(os.Stderr, "monique ui:", uiErr)
		os.Exit(1)
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, "monique track:", err)
		os.Exit(1)
	}
}

func runTrack(db *storage.SQLite) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	t := tracker.New(collector.New(), db)
	if err := t.Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "monique track:", err)
		os.Exit(1)
	}
}

// runUI opens the viewer. chart selects the weekly chart as the start view.
func runUI(db *storage.SQLite, chart bool) {
	m := tui.New(stats.New(db), chart)
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "monique ui:", err)
		os.Exit(1)
	}
}
