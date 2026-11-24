package stock

import (
	"context"
	"errors"
	"fmt"
)

// MaxStock is the largest quantity a seller can enter, to catch typos.
const MaxStock = 1_000_000

// Service holds the stock rules.
type Service struct {
	repo              Repository
	lowStockThreshold int
}

// NewService builds the service; lowStockThreshold comes from LOW_STOCK_THRESHOLD.
func NewService(repo Repository, lowStockThreshold int) (*Service, error) {
	if lowStockThreshold < 0 {
		return nil, fmt.Errorf("LOW_STOCK_THRESHOLD must not be negative, got %d", lowStockThreshold)
	}
	return &Service{repo: repo, lowStockThreshold: lowStockThreshold}, nil
}

// SetStock sets stock if it still equals expected; on conflict returns the current level.
func (s *Service) SetStock(ctx context.Context, shopID, productID int64, stock, expected int) (Level, error) {
	if productID <= 0 {
		return Level{}, ErrNotFound
	}
	if stock < 0 || stock > MaxStock {
		return Level{}, fmt.Errorf("%w: stock must be between 0 and %d", ErrValidation, MaxStock)
	}
	if expected < 0 || expected > MaxStock {
		return Level{}, fmt.Errorf("%w: expected_stock must be between 0 and %d", ErrValidation, MaxStock)
	}
	row, err := s.repo.SetStock(ctx, shopID, productID, stock, expected)
	switch {
	case errors.Is(err, ErrStockChanged), errors.Is(err, ErrBelowReserved):
		return s.level(productID, row), err
	case err != nil:
		return Level{}, err
	}
	return s.level(productID, row), nil
}

// Label returns the stock label for an available quantity using the configured threshold.
func (s *Service) Label(available int) Label {
	return LabelFor(available, s.lowStockThreshold)
}

func (s *Service) level(productID int64, row Row) Level {
	available := row.Stock - row.Reserved
	return Level{
		ProductID: productID,
		Stock:     row.Stock,
		Reserved:  row.Reserved,
		Available: available,
		Label:     s.Label(available),
		UpdatedAt: row.UpdatedAt.UTC(),
	}
}
