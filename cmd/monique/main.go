// Command monique tracks focused windows (via Hyprland) into SQLite and
// shows a live Bubble Tea view of where the time went.
package main

import (
	"context"
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
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
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

	switch os.Args[1] {
	case "track":
		runTrack(db)
	case "ui":
		runUI(db)
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: monique <track|ui>")
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

func runUI(db *storage.SQLite) {
	m := tui.New(stats.New(db))
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "monique ui:", err)
		os.Exit(1)
	}
}
