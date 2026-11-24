package balance

import "context"

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// Service reads the shop's balance. Entries are written by the order feature.
type Service struct {
	repo Repository
}

// NewService returns the balance service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Summary returns the shop's totals.
func (s *Service) Summary(ctx context.Context, shopID int64) (Summary, error) {
	return s.repo.Summary(ctx, shopID)
}

// Entries returns one page of history; page and pageSize are clamped like product lists.
func (s *Service) Entries(ctx context.Context, shopID int64, page, pageSize int) (Page, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	items, total, err := s.repo.Entries(ctx, shopID, pageSize, (page-1)*pageSize)
	if err != nil {
		return Page{}, err
	}
	if items == nil {
		items = []Entry{}
	}
	return Page{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}
