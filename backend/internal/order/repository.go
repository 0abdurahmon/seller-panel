package order

import (
	"context"
	"time"
)

// Balance entry types, from docs/DOMAIN.md.
const (
	EntrySale       = "sale"
	EntryCommission = "commission"
)

// BalanceEntry is one money movement for the shop.
type BalanceEntry struct {
	ShopID    int64
	OrderID   int64
	Type      string
	Amount    int64
	CreatedAt time.Time
}

// Filter narrows the order list.
type Filter struct {
	Status   *Status
	Page     int
	PageSize int
}

// Repository reads orders. Every method is scoped by shopID.
type Repository interface {
	List(ctx context.Context, shopID int64, f Filter) (Page[Summary], error)
	Get(ctx context.Context, shopID, orderID int64) (Order, error)
	// InTx runs fn in one transaction. Changes are rolled back if fn returns an error.
	InTx(ctx context.Context, fn func(tx TxRepository) error) error
	// SimulatableProducts lists the shop's active products that have available stock.
	SimulatableProducts(ctx context.Context, shopID int64) ([]ProductRef, error)
	// Create stores a new order with its items and the first history row, and returns it.
	Create(ctx context.Context, shopID int64, o NewOrder, at time.Time) (Order, error)
}

// TxRepository holds the writes a transition needs. All writes happen in one transaction.
type TxRepository interface {
	// LockOrder reads the order and locks its row until the transaction ends.
	LockOrder(ctx context.Context, shopID, orderID int64) (Order, error)
	// Reserve adds qty to reserved stock. It fails with ErrInsufficientStock when available < qty.
	Reserve(ctx context.Context, shopID, productID int64, qty int) error
	// Release removes qty from reserved stock.
	Release(ctx context.Context, shopID, productID int64, qty int) error
	// Deliver takes qty out of both stock and reserved stock.
	Deliver(ctx context.Context, shopID, productID int64, qty int) error
	SetStatus(ctx context.Context, shopID, orderID int64, to Status, reason *string, at time.Time) error
	AddBalance(ctx context.Context, entries []BalanceEntry) error
}
