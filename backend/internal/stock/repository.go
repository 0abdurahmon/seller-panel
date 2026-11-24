package stock

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Business errors. The handler maps them to HTTP status codes.
var (
	ErrValidation    = errors.New("validation failed")
	ErrNotFound      = errors.New("product not found")
	ErrArchived      = errors.New("product is archived")
	ErrStockChanged  = errors.New("stock was changed by someone else")
	ErrBelowReserved = errors.New("stock cannot be lower than reserved")
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

// MemStore is an in-memory Repository for unit tests, with the same rules.
type MemStore struct {
	mu    sync.Mutex
	items map[memKey]*memItem
	now   func() time.Time
}

type memKey struct{ shopID, productID int64 }

type memItem struct {
	stock, reserved int
	archived        bool
	updatedAt       time.Time
}

// NewMemStore returns an empty in-memory store.
func NewMemStore() *MemStore {
	return &MemStore{items: make(map[memKey]*memItem), now: time.Now}
}

// Put adds or replaces a product's stock. Tests use it to set up data.
func (s *MemStore) Put(shopID, productID int64, stock, reserved int, archived bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[memKey{shopID, productID}] = &memItem{stock: stock, reserved: reserved, archived: archived, updatedAt: s.now()}
}

// SetStock implements Repository.
func (s *MemStore) SetStock(_ context.Context, shopID, productID int64, stock, expected int) (Row, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.items[memKey{shopID, productID}]
	if !ok {
		return Row{}, ErrNotFound
	}
	current := Row{Stock: it.stock, Reserved: it.reserved, UpdatedAt: it.updatedAt}
	switch {
	case it.archived:
		return Row{}, ErrArchived
	case it.stock != expected:
		return current, ErrStockChanged
	case stock < it.reserved:
		return current, ErrBelowReserved
	}
	it.stock = stock
	it.updatedAt = s.now()
	return Row{Stock: it.stock, Reserved: it.reserved, UpdatedAt: it.updatedAt}, nil
}
