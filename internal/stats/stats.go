// Package stats computes aggregate views over stored sessions.
package stats

import (
	"context"
	"time"

	"monique/internal/domain"
)

type Store struct {
	repo domain.Repository
}

func New(repo domain.Repository) *Store {
	return &Store{repo: repo}
}

// Today returns total focused time per app since midnight, including time
// accrued by the currently-open session.
func (s *Store) Today(ctx context.Context) ([]domain.AppTotal, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return s.repo.TimeByApp(ctx, startOfDay.Unix(), now.Unix())
}
