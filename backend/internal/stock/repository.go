package stock

import (
	"context"
	"time"
)

// Row is the stock part of a product as stored.
type Row struct {
	Stock     int
	Reserved  int
	UpdatedAt time.Time
}

// Repository stores stock; every method is scoped by shopID.
type Repository interface {
	// SetStock errors in order: NotFound, Archived, StockChanged, BelowReserved (last two return current).
	SetStock(ctx context.Context, shopID, productID int64, stock, expected int) (Row, error)
}
