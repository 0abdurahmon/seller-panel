package stock

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Business errors. The handler maps them to HTTP status codes.
var (
	ErrValidation        = errors.New("validation failed")
	ErrNotFound          = errors.New("product not found")
	ErrArchived          = errors.New("product is archived")
	ErrBelowReserved     = errors.New("stock cannot be lower than reserved")
	ErrInsufficientStock = errors.New("insufficient stock")
)

// Row is the stock part of a product as stored.
type Row struct {
	Stock     int
	Reserved  int
	UpdatedAt time.Time
}

// Repository is the persistence the service needs. Every method is scoped by
// shopID: a product of another shop behaves exactly like a missing one.
type Repository interface {
	// SetStock sets the stock of an active or draft product. It returns
	// ErrNotFound, ErrArchived or ErrBelowReserved when the change is refused.
	SetStock(ctx context.Context, shopID, productID int64, stock int) (Row, error)
}

// MemStore keeps products' stock in memory. It is used in unit tests and
// follows the same rules as the PostgreSQL store.
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
func (s *MemStore) SetStock(_ context.Context, shopID, productID int64, stock int) (Row, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.items[memKey{shopID, productID}]
	switch {
	case !ok:
		return Row{}, ErrNotFound
	case it.archived:
		return Row{}, ErrArchived
	case stock < it.reserved:
		return Row{}, ErrBelowReserved
	}
	it.stock = stock
	it.updatedAt = s.now()
	return Row{Stock: it.stock, Reserved: it.reserved, UpdatedAt: it.updatedAt}, nil
}
