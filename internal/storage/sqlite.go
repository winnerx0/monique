// Package storage implements domain.Repository on top of SQLite.
package storage

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite"

	"monique/internal/domain"
)

//go:embed schema.sql
var schema string

type SQLite struct {
	db *sql.DB
}

// Open opens (creating if needed) the SQLite database at path in WAL mode
// and applies the schema. Safe to call from both the tracker and the UI.
func Open(path string) (*SQLite, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;`); err != nil {
		return nil, fmt.Errorf("set pragmas: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &SQLite{db: db}, nil
}

func (s *SQLite) Close() error {
	return s.db.Close()
}

func (s *SQLite) OpenSession(ctx context.Context, ev domain.FocusEvent, at int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`UPDATE focus_sessions SET ended_at = ?, duration_seconds = ? - started_at WHERE ended_at IS NULL`,
		at, at); err != nil {
		return fmt.Errorf("close previous session: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO focus_sessions (app_class, title, pid, started_at, last_seen_at) VALUES (?, ?, ?, ?, ?)`,
		ev.AppClass, ev.Title, ev.PID, at, at); err != nil {
		return fmt.Errorf("insert session: %w", err)
	}

	return tx.Commit()
}

func (s *SQLite) Heartbeat(ctx context.Context, at int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE focus_sessions SET last_seen_at = ? WHERE ended_at IS NULL`, at)
	return err
}

func (s *SQLite) CloseOpenSession(ctx context.Context, at int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE focus_sessions SET ended_at = ?, duration_seconds = ? - started_at WHERE ended_at IS NULL`,
		at, at)
	return err
}

func (s *SQLite) RecoverDangling(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE focus_sessions SET ended_at = last_seen_at, duration_seconds = last_seen_at - started_at WHERE ended_at IS NULL`)
	return err
}

func (s *SQLite) TimeByApp(ctx context.Context, since int64, now int64) ([]domain.AppTotal, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT app_class, title, SUM(COALESCE(duration_seconds, ? - started_at)) AS total
		FROM focus_sessions
		WHERE started_at >= ?
		GROUP BY app_class, title
		ORDER BY total DESC
	`, now, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.AppTotal
	for rows.Next() {
		var t domain.AppTotal
		if err := rows.Scan(&t.AppClass, &t.Title, &t.DurationSeconds); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *SQLite) TimeByDay(ctx context.Context, since int64, now int64) ([]domain.DayTotal, error) {
	// Per day: the total across all apps, plus the single most-used app that
	// day and its own time. per_app aggregates each app's daily time; day_top
	// picks the busiest app per day; we join it back to the day total.
	rows, err := s.db.QueryContext(ctx, `
		WITH per_app AS (
			SELECT date(started_at, 'unixepoch', 'localtime') AS day,
			       app_class,
			       SUM(COALESCE(duration_seconds, ? - started_at)) AS app_total
			FROM focus_sessions
			WHERE started_at >= ?
			GROUP BY day, app_class
		),
		day_top AS (
			SELECT day, app_class AS top_app, app_total AS top_total,
			       ROW_NUMBER() OVER (PARTITION BY day ORDER BY app_total DESC) AS rn
			FROM per_app
		)
		SELECT p.day, SUM(p.app_total) AS total,
		       COALESCE(t.top_app, '') AS top_app,
		       COALESCE(t.top_total, 0) AS top_total
		FROM per_app p
		LEFT JOIN day_top t ON t.day = p.day AND t.rn = 1
		GROUP BY p.day
		ORDER BY p.day
	`, now, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.DayTotal
	for rows.Next() {
		var t domain.DayTotal
		if err := rows.Scan(&t.Date, &t.DurationSeconds, &t.TopApp, &t.TopAppSeconds); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
