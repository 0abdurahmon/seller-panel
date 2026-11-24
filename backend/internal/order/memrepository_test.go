package order

import (
	"context"
	"sync"
	"time"
)

type memRepository struct {
	mu       sync.Mutex
	orders   map[int64]Order
	products map[int64]*memProduct
	balance  []BalanceEntry
}

type memProduct struct {
	shopID   int64
	stock    int
	reserved int
}

func newMemRepository() *memRepository {
	return &memRepository{orders: map[int64]Order{}, products: map[int64]*memProduct{}}
}

func (m *memRepository) AddOrder(o Order) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orders[o.ID] = o
}

func (m *memRepository) AddProduct(id, shopID int64, stock, reserved int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.products[id] = &memProduct{shopID: shopID, stock: stock, reserved: reserved}
}

func (m *memRepository) Product(id int64) (stock, reserved int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := m.products[id]
	return p.stock, p.reserved
}

func (m *memRepository) Balance() []BalanceEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]BalanceEntry(nil), m.balance...)
}

func (m *memRepository) List(_ context.Context, shopID int64, f Filter) (Page[Summary], error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var all []Summary
	for _, o := range m.orders {
		if o.ShopID != shopID || (f.Status != nil && o.Status != *f.Status) {
			continue
		}
		all = append(all, Summary{ID: o.ID, Number: o.Number, Status: o.Status, BuyerName: o.BuyerName, Total: o.Total, CreatedAt: o.CreatedAt})
	}
	// Newest first; ties broken by ID.
	for i := 1; i < len(all); i++ {
		for j := i; j > 0 && newer(all[j], all[j-1]); j-- {
			all[j], all[j-1] = all[j-1], all[j]
		}
	}
	start := (f.Page - 1) * f.PageSize
	if start > len(all) {
		start = len(all)
	}
	end := start + f.PageSize
	if end > len(all) {
		end = len(all)
	}
	return Page[Summary]{Items: all[start:end], Page: f.Page, PageSize: f.PageSize, Total: len(all)}, nil
}

func newer(a, b Summary) bool {
	if !a.CreatedAt.Equal(b.CreatedAt) {
		return a.CreatedAt.After(b.CreatedAt)
	}
	return a.ID > b.ID
}

func (m *memRepository) Get(_ context.Context, shopID, orderID int64) (Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.orders[orderID]
	if !ok || o.ShopID != shopID {
		return Order{}, ErrNotFound
	}
	return o, nil
}

func (m *memRepository) InTx(_ context.Context, fn func(tx TxRepository) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Snapshot for rollback.
	orders := make(map[int64]Order, len(m.orders))
	for k, v := range m.orders {
		orders[k] = v
	}
	products := make(map[int64]memProduct, len(m.products))
	for k, v := range m.products {
		products[k] = *v
	}
	balance := append([]BalanceEntry(nil), m.balance...)

	if err := fn(&memTx{m: m}); err != nil {
		m.orders, m.balance = orders, balance
		m.products = make(map[int64]*memProduct, len(products))
		for k, v := range products {
			v := v
			m.products[k] = &v
		}
		return err
	}
	return nil
}

type memTx struct {
	m *memRepository
}

func (t *memTx) LockOrder(_ context.Context, shopID, orderID int64) (Order, error) {
	o, ok := t.m.orders[orderID]
	if !ok || o.ShopID != shopID {
		return Order{}, ErrNotFound
	}
	return o, nil
}

func (t *memTx) product(shopID, productID int64) (*memProduct, error) {
	p, ok := t.m.products[productID]
	if !ok || p.shopID != shopID {
		return nil, ErrProductNotFound
	}
	return p, nil
}

func (t *memTx) Reserve(_ context.Context, shopID, productID int64, qty int) error {
	p, err := t.product(shopID, productID)
	if err != nil {
		return err
	}
	if p.stock-p.reserved < qty {
		return ErrInsufficientStock
	}
	p.reserved += qty
	return nil
}

func (t *memTx) Release(_ context.Context, shopID, productID int64, qty int) error {
	p, err := t.product(shopID, productID)
	if err != nil {
		return err
	}
	p.reserved -= qty
	return nil
}

func (t *memTx) Deliver(_ context.Context, shopID, productID int64, qty int) error {
	p, err := t.product(shopID, productID)
	if err != nil {
		return err
	}
	p.stock -= qty
	p.reserved -= qty
	return nil
}

func (t *memTx) SetStatus(_ context.Context, shopID, orderID int64, to Status, reason *string, at time.Time) error {
	o, ok := t.m.orders[orderID]
	if !ok || o.ShopID != shopID {
		return ErrNotFound
	}
	from := o.Status
	o.History = append(o.History, HistoryEntry{FromStatus: &from, ToStatus: to, Reason: reason, CreatedAt: at})
	o.Status = to
	o.CancelReason = reason
	o.StatusChangedAt = at
	t.m.orders[orderID] = o
	return nil
}

func (t *memTx) AddBalance(_ context.Context, entries []BalanceEntry) error {
	t.m.balance = append(t.m.balance, entries...)
	return nil
}
