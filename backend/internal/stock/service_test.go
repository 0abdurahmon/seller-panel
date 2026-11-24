package stock

import (
	"context"
	"errors"
	"testing"
)

func newService(t *testing.T) (*Service, *memRepository) {
	t.Helper()
	store := newMemRepository()
	return NewService(store), store
}

func stockOf(t *testing.T, store *memRepository, shopID, productID int64) int {
	t.Helper()
	row, ok := store.row(shopID, productID)
	if !ok {
		t.Fatalf("product %d of shop %d is missing", productID, shopID)
	}
	return row.Stock
}

func TestSetStockChangesTheStock(t *testing.T) {
	svc, store := newService(t)
	store.Put(1, 10, 20, 3, false)
	ctx := context.Background()

	for _, c := range []struct{ stock, expected int }{{50, 20}, {8, 50}, {3, 8}} {
		if err := svc.SetStock(ctx, 1, 10, c.stock, c.expected); err != nil {
			t.Fatalf("SetStock(%d): %v", c.stock, err)
		}
		if got := stockOf(t, store, 1, 10); got != c.stock {
			t.Fatalf("stock = %d, want %d", got, c.stock)
		}
	}
	if row, _ := store.row(1, 10); row.Reserved != 3 {
		t.Fatalf("reserved = %d, a stock change must not touch it", row.Reserved)
	}
}

func TestSetStockRules(t *testing.T) {
	svc, store := newService(t)
	store.Put(1, 10, 20, 4, false) // active, 4 reserved
	store.Put(1, 11, 7, 0, true)   // archived
	store.Put(2, 12, 9, 0, false)  // belongs to shop 2
	ctx := context.Background()

	cases := []struct {
		name            string
		productID       int64
		stock, expected int
		want            error
	}{
		{"negative", 10, -1, 20, ErrValidation},
		{"above max", 10, MaxStock + 1, 20, ErrValidation},
		{"negative expected", 10, 5, -1, ErrValidation},
		{"expected above max", 10, 5, MaxStock + 1, ErrValidation},
		{"below reserved", 10, 3, 20, ErrBelowReserved},
		{"archived", 11, 5, 7, ErrArchived},
		{"missing", 99, 5, 0, ErrNotFound},
		{"bad id", 0, 5, 0, ErrNotFound},
		{"other shop", 12, 5, 9, ErrNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := svc.SetStock(ctx, 1, c.productID, c.stock, c.expected)
			if !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
		})
	}

	// Equal to reserved, the max and zero are allowed edges.
	if err := svc.SetStock(ctx, 1, 10, 4, 20); err != nil {
		t.Fatalf("equal to reserved: %v", err)
	}
	if err := svc.SetStock(ctx, 1, 10, MaxStock, 4); err != nil {
		t.Fatalf("max: %v", err)
	}
	store.Put(1, 13, 5, 0, false)
	if err := svc.SetStock(ctx, 1, 13, 0, 5); err != nil || stockOf(t, store, 1, 13) != 0 {
		t.Fatalf("zero: err %v, stock %d", err, stockOf(t, store, 1, 13))
	}
}

func TestSetStockBelowReservedKeepsTheStock(t *testing.T) {
	svc, store := newService(t)
	store.Put(1, 10, 20, 4, false)
	ctx := context.Background()

	if err := svc.SetStock(ctx, 1, 10, 2, 20); !errors.Is(err, ErrBelowReserved) {
		t.Fatalf("err = %v, want ErrBelowReserved", err)
	}
	if got := stockOf(t, store, 1, 10); got != 20 {
		t.Fatalf("stock = %d, a refused request must not change it", got)
	}
}

func TestSetStockStaleExpectedIsRefused(t *testing.T) {
	svc, store := newService(t)
	store.Put(1, 10, 20, 4, false)
	ctx := context.Background()

	// Tab 1 saw 20 and saves 25.
	if err := svc.SetStock(ctx, 1, 10, 25, 20); err != nil || stockOf(t, store, 1, 10) != 25 {
		t.Fatalf("tab 1: err %v, stock %d; want stock 25", err, stockOf(t, store, 1, 10))
	}
	// Tab 2 also saw 20; its save must not overwrite 25.
	if err := svc.SetStock(ctx, 1, 10, 30, 20); !errors.Is(err, ErrStockChanged) {
		t.Fatalf("tab 2: err = %v, want ErrStockChanged", err)
	}
	if got := stockOf(t, store, 1, 10); got != 25 {
		t.Fatalf("stock = %d, want 25 kept", got)
	}
	// Stale and below reserved at once: stock_changed wins.
	if err := svc.SetStock(ctx, 1, 10, 1, 20); !errors.Is(err, ErrStockChanged) {
		t.Fatalf("stale + below reserved: err = %v, want ErrStockChanged", err)
	}
	// With the fresh value tab 2 succeeds.
	if err := svc.SetStock(ctx, 1, 10, 30, 25); err != nil || stockOf(t, store, 1, 10) != 30 {
		t.Fatalf("tab 2 retry: err %v, stock %d; want stock 30", err, stockOf(t, store, 1, 10))
	}
}
