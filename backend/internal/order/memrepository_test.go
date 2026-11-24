package order

import (
	"context"
	"fmt"
	"sort"
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
	title    string
	price    int64
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
	m.AddPricedProduct(id, shopID, stock, reserved, fmt.Sprintf("Product %d", id), 1000)
}

func (m *memRepository) AddPricedProduct(id, shopID int64, stock, reserved int, title string, price int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.products[id] = &memProduct{shopID: shopID, stock: stock, reserved: reserved, title: title, price: price}
}

func (m *memRepository) SimulatableProducts(_ context.Context, shopID int64) ([]ProductRef, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []ProductRef
	for id, p := range m.products {
		if p.shopID == shopID && p.stock-p.reserved > 0 {
			out = append(out, ProductRef{ID: id, Title: p.title, Price: p.price, Available: p.stock - p.reserved})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (m *memRepository) Create(_ context.Context, shopID int64, n NewOrder, at time.Time) (Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var id int64
	for existing := range m.orders {
		if existing > id {
			id = existing
		}
	}
	id++
	for _, it := range n.Items {
		if p, ok := m.products[it.ProductID]; !ok || p.shopID != shopID {
			return Order{}, ErrProductNotFound
		}
	}
	o := Order{ID: id, ShopID: shopID, Number: fmt.Sprintf("#%d", 100000+id), Status: StatusNew,
		BuyerName: n.BuyerName, BuyerPhone: n.BuyerPhone, DeliveryAddress: n.DeliveryAddress,
		Items: []Item{}, CreatedAt: at, StatusChangedAt: at}
	for i, it := range n.Items {
		o.Items = append(o.Items, Item{ID: int64(i + 1), ProductID: it.ProductID, Title: it.Title, Price: it.Price, Quantity: it.Quantity})
		o.Total += it.Price * int64(it.Quantity)
	}
	o.History = []HistoryEntry{{ToStatus: StatusNew, CreatedAt: at}}
	m.orders[id] = o
	return o, nil
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
