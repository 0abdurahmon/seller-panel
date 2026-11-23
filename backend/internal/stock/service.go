package stock

import (
	"context"
	"fmt"
)

// MaxStock is the largest quantity a seller can enter. It keeps typos such as
// an extra zero row out of the database and stays far below the integer limit.
const MaxStock = 1_000_000

// Service holds the stock rules.
type Service struct {
	repo              Repository
	lowStockThreshold int
}

// NewService builds the stock service. lowStockThreshold comes from config
// (LOW_STOCK_THRESHOLD, default 5) and must not be negative.
func NewService(repo Repository, lowStockThreshold int) (*Service, error) {
	if lowStockThreshold < 0 {
		return nil, fmt.Errorf("LOW_STOCK_THRESHOLD must not be negative, got %d", lowStockThreshold)
	}
	return &Service{repo: repo, lowStockThreshold: lowStockThreshold}, nil
}

// SetStock sets the stock of one product of the shop and returns the new level.
// The new stock must be between 0 and MaxStock and not below what is already
// reserved by accepted orders: the seller cannot take back promised goods.
func (s *Service) SetStock(ctx context.Context, shopID, productID int64, stock int) (Level, error) {
	if productID <= 0 {
		return Level{}, ErrNotFound
	}
	if stock < 0 || stock > MaxStock {
		return Level{}, fmt.Errorf("%w: stock must be between 0 and %d", ErrValidation, MaxStock)
	}
	row, err := s.repo.SetStock(ctx, shopID, productID, stock)
	if err != nil {
		return Level{}, err
	}
	return s.level(productID, row), nil
}

// Label returns the label for an available quantity using the configured
// threshold. The product list uses it so every screen shows the same label.
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
		UpdatedAt: row.UpdatedAt,
	}
}
