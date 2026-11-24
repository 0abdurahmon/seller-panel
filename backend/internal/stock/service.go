package stock

import (
	"context"
	"fmt"
)

// MaxStock is the largest quantity a seller can enter, to catch typos.
const MaxStock = 1_000_000

// Service holds the stock rules.
type Service struct {
	repo Repository
}

// NewService builds the service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// SetStock sets stock if it still equals expected. The product itself is read by the caller afterwards.
func (s *Service) SetStock(ctx context.Context, shopID, productID int64, stock, expected int) error {
	if productID <= 0 {
		return ErrNotFound
	}
	if stock < 0 || stock > MaxStock {
		return fmt.Errorf("%w: stock must be between 0 and %d", ErrValidation, MaxStock)
	}
	if expected < 0 || expected > MaxStock {
		return fmt.Errorf("%w: expected_stock must be between 0 and %d", ErrValidation, MaxStock)
	}
	_, err := s.repo.SetStock(ctx, shopID, productID, stock, expected)
	return err
}
