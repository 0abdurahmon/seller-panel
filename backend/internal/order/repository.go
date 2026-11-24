package order

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Errors returned by the repository. The service maps them to business errors.
var (
	ErrNotFound          = errors.New("order not found")
	ErrProductNotFound   = errors.New("product not found")
	ErrInsufficientStock = errors.New("not enough available stock")
	ErrInvalidTransition = errors.New("transition not allowed")
	ErrValidation        = errors.New("validation failed")
)

// Balance entry types, from docs/DOMAIN.md.
const (
	EntrySale       = "sale"
	EntryCommission = "commission"
)

// BalanceEntry is one money movement for the shop.
type BalanceEntry struct {
	ShopID  int64
	OrderID int64
	Type    string
	Amount  int64
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

// MemRepository is an in-memory Repository for unit tests. InTx restores the
// previous state when fn fails, like a rollback.
type MemRepository struct {
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

// NewMemRepository returns an empty in-memory repository.
func NewMemRepository() *MemRepository {
	return &MemRepository{orders: map[int64]Order{}, products: map[int64]*memProduct{}}
}

// AddOrder stores an order for tests.
func (m *MemRepository) AddOrder(o Order) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orders[o.ID] = o
}

// AddProduct stores a product's stock for tests.
func (m *MemRepository) AddProduct(id, shopID int64, stock, reserved int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.products[id] = &memProduct{shopID: shopID, stock: stock, reserved: reserved}
}

// Product returns the stock and reserved count of a product for tests.
func (m *MemRepository) Product(id int64) (stock, reserved int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := m.products[id]
	return p.stock, p.reserved
}

// Balance returns the balance entries written so far.
func (m *MemRepository) Balance() []BalanceEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]BalanceEntry(nil), m.balance...)
}

func (m *MemRepository) List(_ context.Context, shopID int64, f Filter) (Page[Summary], error) {
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

func (m *MemRepository) Get(_ context.Context, shopID, orderID int64) (Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.orders[orderID]
	if !ok || o.ShopID != shopID {
		return Order{}, ErrNotFound
	}
	return o, nil
}

func (m *MemRepository) InTx(_ context.Context, fn func(tx TxRepository) error) error {
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

// memTx is the transaction view. It is only used while MemRepository.mu is held by InTx.
type memTx struct {
	m *MemRepository
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
