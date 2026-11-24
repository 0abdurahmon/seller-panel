package stock

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLabelFor(t *testing.T) {
	cases := []struct {
		available, threshold int
		want                 Label
	}{
		{0, 5, OutOfStock},
		{-1, 5, OutOfStock}, // never stored, but must not read as in stock
		{1, 5, LowStock},
		{5, 5, LowStock}, // the threshold itself is low
		{6, 5, InStock},
		{1, 0, InStock}, // threshold 0 turns the warning off
		{0, 0, OutOfStock},
	}
	for _, c := range cases {
		if got := LabelFor(c.available, c.threshold); got != c.want {
			t.Errorf("LabelFor(%d, %d) = %s, want %s", c.available, c.threshold, got, c.want)
		}
	}
}

func newService(t *testing.T) (*Service, *memRepository) {
	t.Helper()
	store := newMemRepository()
	svc, err := NewService(store, 5)
	if err != nil {
		t.Fatal(err)
	}
	return svc, store
}

func TestNewServiceRejectsNegativeThreshold(t *testing.T) {
	if _, err := NewService(newMemRepository(), -1); err == nil {
		t.Fatal("negative threshold accepted")
	}
}

func TestSetStockReturnsLevelWithLabel(t *testing.T) {
	svc, store := newService(t)
	store.Put(1, 10, 20, 3, false)
	ctx := context.Background()

	cases := []struct {
		stock, expected int
		wantAvailable   int
		wantLabel       Label
	}{
		{50, 20, 47, InStock},
		{8, 50, 5, LowStock},
		{3, 8, 0, OutOfStock}, // everything left is reserved
	}
	for _, c := range cases {
		got, err := svc.SetStock(ctx, 1, 10, c.stock, c.expected)
		if err != nil {
			t.Fatalf("SetStock(%d): %v", c.stock, err)
		}
		if got.ProductID != 10 || got.Stock != c.stock || got.Reserved != 3 ||
			got.Available != c.wantAvailable || got.Label != c.wantLabel {
			t.Fatalf("SetStock(%d) = %+v, want available %d label %s", c.stock, got, c.wantAvailable, c.wantLabel)
		}
		if got.UpdatedAt.Location() != time.UTC {
			t.Fatalf("updated_at is in %s, want UTC", got.UpdatedAt.Location())
		}
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
			_, err := svc.SetStock(ctx, 1, c.productID, c.stock, c.expected)
			if !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
		})
	}

	// Equal to reserved, the max and zero are allowed edges.
	if _, err := svc.SetStock(ctx, 1, 10, 4, 20); err != nil {
		t.Fatalf("equal to reserved: %v", err)
	}
	if _, err := svc.SetStock(ctx, 1, 10, MaxStock, 4); err != nil {
		t.Fatalf("max: %v", err)
	}
	store.Put(1, 13, 5, 0, false)
	if got, err := svc.SetStock(ctx, 1, 13, 0, 5); err != nil || got.Label != OutOfStock {
		t.Fatalf("zero = %+v, %v; want out_of_stock", got, err)
	}
}

func TestSetStockBelowReservedReturnsCurrentAndKeepsStock(t *testing.T) {
	svc, store := newService(t)
	store.Put(1, 10, 20, 4, false)
	ctx := context.Background()

	cur, err := svc.SetStock(ctx, 1, 10, 2, 20)
	if !errors.Is(err, ErrBelowReserved) {
		t.Fatalf("err = %v, want ErrBelowReserved", err)
	}
	if cur.Stock != 20 || cur.Reserved != 4 || cur.Available != 16 {
		t.Fatalf("refusal level = %+v, want current stock 20 reserved 4", cur)
	}
	if got, err := svc.SetStock(ctx, 1, 10, 20, 20); err != nil || got.Stock != 20 {
		t.Fatalf("stock changed by a refused request: %+v, %v", got, err)
	}
}

func TestSetStockStaleExpectedIsRefused(t *testing.T) {
	svc, store := newService(t)
	store.Put(1, 10, 20, 4, false)
	ctx := context.Background()

	// Tab 1 saw 20 and saves 25.
	if got, err := svc.SetStock(ctx, 1, 10, 25, 20); err != nil || got.Stock != 25 {
		t.Fatalf("tab 1 = %+v, %v; want stock 25", got, err)
	}
	// Tab 2 also saw 20; its save must not overwrite 25.
	cur, err := svc.SetStock(ctx, 1, 10, 30, 20)
	if !errors.Is(err, ErrStockChanged) {
		t.Fatalf("tab 2: err = %v, want ErrStockChanged", err)
	}
	if cur.Stock != 25 || cur.Available != 21 {
		t.Fatalf("tab 2 got level %+v, want current stock 25", cur)
	}
	// Stale and below reserved at once: stock_changed wins.
	if _, err := svc.SetStock(ctx, 1, 10, 1, 20); !errors.Is(err, ErrStockChanged) {
		t.Fatalf("stale + below reserved: err = %v, want ErrStockChanged", err)
	}
	// With the fresh value tab 2 succeeds.
	if got, err := svc.SetStock(ctx, 1, 10, 30, 25); err != nil || got.Stock != 30 {
		t.Fatalf("tab 2 retry = %+v, %v; want stock 30", got, err)
	}
}
