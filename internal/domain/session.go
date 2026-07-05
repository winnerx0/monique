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

// AppTotal is an aggregated total for one app/title.
type AppTotal struct {
	AppClass        string
	Title           string
	DurationSeconds int64
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
	// TimeByApp returns total focused duration per app since the given time,
	// including any time accrued by the currently-open session.
	TimeByApp(ctx context.Context, since int64, now int64) ([]AppTotal, error)
}
