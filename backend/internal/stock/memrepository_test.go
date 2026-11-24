package stock

import (
	"context"
	"sync"
	"time"
)

type memRepository struct {
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

func newMemRepository() *memRepository {
	return &memRepository{items: make(map[memKey]*memItem), now: time.Now}
}

func (s *memRepository) Put(shopID, productID int64, stock, reserved int, archived bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[memKey{shopID, productID}] = &memItem{stock: stock, reserved: reserved, archived: archived, updatedAt: s.now()}
}

func (s *memRepository) SetStock(_ context.Context, shopID, productID int64, stock, expected int) (Row, error) {
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

func (s *memRepository) row(shopID, productID int64) (Row, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.items[memKey{shopID, productID}]
	if !ok {
		return Row{}, false
	}
	return Row{Stock: it.stock, Reserved: it.reserved, UpdatedAt: it.updatedAt}, true
}
