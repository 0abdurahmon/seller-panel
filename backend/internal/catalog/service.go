package catalog

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context) (ListResponse, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return ListResponse{}, err
	}
	if items == nil {
		items = []Category{}
	}
	return ListResponse{Items: items}, nil
}
