// Package tracker turns a stream of focus events into stored sessions.
package tracker

import (
	"context"
	"time"

	"monique/internal/domain"
)

const heartbeatInterval = 30 * time.Second

type Tracker struct {
	collector domain.Collector
	repo      domain.Repository
	current   *domain.FocusEvent
}

func New(collector domain.Collector, repo domain.Repository) *Tracker {
	return &Tracker{collector: collector, repo: repo}
}

// Run recovers any dangling session from a previous crash, then consumes
// focus events until ctx is cancelled, at which point it closes the
// currently-open session gracefully.
func (t *Tracker) Run(ctx context.Context) error {
	if err := t.repo.RecoverDangling(ctx); err != nil {
		return err
	}

	events, err := t.collector.Events(ctx)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return t.repo.CloseOpenSession(context.Background(), time.Now().Unix())
			}
			// Hyprland sometimes fires activewindowv2 twice for the same
			// focus change; skip re-opening a session for the same window.
			if t.current != nil && *t.current == ev {
				continue
			}
			if err := t.repo.OpenSession(ctx, ev, time.Now().Unix()); err != nil {
				return err
			}
			t.current = &ev
		case <-ticker.C:
			if err := t.repo.Heartbeat(ctx, time.Now().Unix()); err != nil {
				return err
			}
		case <-ctx.Done():
			return t.repo.CloseOpenSession(context.Background(), time.Now().Unix())
		}
	}
}
