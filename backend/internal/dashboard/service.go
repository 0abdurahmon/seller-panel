package dashboard

import (
	"context"
	"errors"
	"time"
)

// Service builds the dashboard summary. It only reads; orders are written by the order feature.
type Service struct {
	repo Repository
	loc  *time.Location
	now  func() time.Time
}

// NewService returns the dashboard service; loc is the business time zone that defines "today".
func NewService(repo Repository, loc *time.Location) (*Service, error) {
	if loc == nil {
		return nil, errors.New("dashboard: the business time zone is required")
	}
	return &Service{repo: repo, loc: loc, now: time.Now}, nil
}

// Summary returns the home page numbers of the shop as of now.
func (s *Service) Summary(ctx context.Context, shopID int64) (Summary, error) {
	res, err := s.repo.Summary(ctx, shopID, s.window(s.now()))
	if err != nil {
		return Summary{}, err
	}
	if res.TopProducts == nil {
		res.TopProducts = []TopProduct{}
	}
	return res, nil
}

// window turns "now" into calendar days of the business time zone: today, today and the 6 days
// before it, today and the 29 days before it. Dates are added in the zone, so a day is a
// calendar day even when its length changes.
func (s *Service) window(now time.Time) Window {
	local := now.In(s.loc)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, s.loc)
	return Window{
		TodayStart: today.UTC(),
		TodayEnd:   today.AddDate(0, 0, 1).UTC(),
		From7D:     today.AddDate(0, 0, -6).UTC(),
		From30D:    today.AddDate(0, 0, -29).UTC(),
	}
}
