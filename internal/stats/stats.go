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

// Recent returns the latest focus events (newest first) for the live log.
func (s *Store) Recent(ctx context.Context, limit int) ([]domain.EventRow, error) {
	return s.repo.RecentEvents(ctx, time.Now().Unix(), limit)
}

// LastDays returns total focused time per day for the last n days (today
// included), one entry per day even when nothing was tracked that day.
func (s *Store) LastDays(ctx context.Context, n int) ([]domain.DayTotal, error) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(n - 1))
	rows, err := s.repo.TimeByDay(ctx, start.Unix(), now.Unix())
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
