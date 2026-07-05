// Package collector polls the OS for the currently-focused window and emits
// a FocusEvent whenever it can read one. The per-OS logic lives in
// active_<goos>.go behind build tags; this file is OS-agnostic.
package collector

import (
	"context"
	"time"

	"monique/internal/domain"
)

const pollInterval = time.Second

type Poll struct{}

func New() *Poll {
	return &Poll{}
}

// Events polls the foreground window every pollInterval and sends a FocusEvent
// each time one is read. Consecutive identical events are de-duplicated
// downstream by the tracker, so this emits unconditionally on every tick.
func (p *Poll) Events(ctx context.Context) (<-chan domain.FocusEvent, error) {
	out := make(chan domain.FocusEvent)

	go func() {
		defer close(out)

		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		emit := func() {
			ev, ok, err := activeWindow(ctx)
			if err != nil || !ok {
				return // nothing focused, or transient failure: skip this tick
			}
			select {
			case out <- ev:
			case <-ctx.Done():
			}
		}

		emit() // capture whatever is focused at startup
		for {
			select {
			case <-ticker.C:
				emit()
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}
