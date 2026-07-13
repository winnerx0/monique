package domain

import "context"

// FocusEvent is emitted whenever the focused window changes.
type FocusEvent struct {
	AppClass string
	Title    string
	PID      int
}

// Session is one continuous period a window held focus.
type Session struct {
	ID              int64
	AppClass        string
	Title           string
	PID             int
	StartedAt       int64
	LastSeenAt      int64
	EndedAt         *int64
	DurationSeconds *int64
}

// EventRow is one focus event for the live activity log: when it started,
// what got focused, how long it lasted (live for the still-open one).
type EventRow struct {
	StartedAt       int64
	AppClass        string
	Title           string
	DurationSeconds int64
	Open            bool // true for the currently-focused session
}

// DayTotal is total focused time for one calendar day, plus that day's
// most-active app.
type DayTotal struct {
	Date            string // YYYY-MM-DD, local time
	DurationSeconds int64
	TopApp          string // most-used app that day ("" if none)
	TopAppSeconds   int64  // that app's time that day
}

// Collector produces focus-change events (e.g. from the Hyprland IPC socket).
type Collector interface {
	Events(ctx context.Context) (<-chan FocusEvent, error)
}

// Repository persists and queries sessions.
type Repository interface {
	// OpenSession closes any currently-open session and inserts a new open one.
	OpenSession(ctx context.Context, ev FocusEvent, at int64) error
	// Heartbeat bumps last_seen_at on the currently-open session.
	Heartbeat(ctx context.Context, at int64) error
	// CloseOpenSession closes the currently-open session, if any, at the given time.
	CloseOpenSession(ctx context.Context, at int64) error
	// RecoverDangling closes any session left open from a previous crash,
	// using its own last_seen_at as the end time.
	RecoverDangling(ctx context.Context) error
	// DeleteBefore removes sessions that started before the given time
	// (retention pruning).
	DeleteBefore(ctx context.Context, cutoff int64) error
	// RecentEvents returns the most recent focus events, newest first, with
	// the currently-open session's duration computed live against now.
	RecentEvents(ctx context.Context, now int64, limit int) ([]EventRow, error)
	// TimeByDay returns total focused duration per local calendar day since
	// the given time, including the currently-open session.
	TimeByDay(ctx context.Context, since int64, now int64) ([]DayTotal, error)
}
