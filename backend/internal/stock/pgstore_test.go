package stock

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0abdurahmon/seller-panel/backend/internal/database"
)

// These tests need a real PostgreSQL with the products table. Set
// TEST_DATABASE_URL to run them; without it, or before the products
// migration exists, they are skipped.
func newPgPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	if err := database.Migrate(url); err != nil {
		t.Fatal(err)
	}
	pool, err := database.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	var exists bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass('public.products') IS NOT NULL`).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Skip("products table does not exist yet (products migration not merged)")
	}
	if _, err := pool.Exec(ctx, `TRUNCATE products, shops, sellers RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}
	return pool
}

// seedProduct creates a seller, shop, category and one product and returns
// the shop and product ids.
func seedProduct(t *testing.T, pool *pgxpool.Pool, email string, stock int, status string) (int64, int64) {
	t.Helper()
	ctx := context.Background()
	insert := func(dest *int64, sql string, args ...any) {
		t.Helper()
		if err := pool.QueryRow(ctx, sql, args...).Scan(dest); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	var sellerID, shopID, categoryID, productID int64
	insert(&sellerID, `INSERT INTO sellers (email, password_hash) VALUES ($1, 'x') RETURNING id`, email)
	insert(&shopID, `INSERT INTO shops (seller_id, name) VALUES ($1, 'Shop') RETURNING id`, sellerID)
	insert(&categoryID, `INSERT INTO categories (name) VALUES ($1) RETURNING id`, "Stock test "+email)
	insert(&productID,
		`INSERT INTO products (shop_id, category_id, title, price, sku, stock, status)
		 VALUES ($1, $2, 'Phone', 100000, 'SKU-1', $3, $4) RETURNING id`,
		shopID, categoryID, stock, status)
	return shopID, productID
}

func TestPgSetStockRules(t *testing.T) {
	pool := newPgPool(t)
	store := NewPgStore(pool)
	ctx := context.Background()
	shop, product := seedProduct(t, pool, "a@example.com", 10, "active")
	otherShop, _ := seedProduct(t, pool, "b@example.com", 10, "active")
	archivedShop, archived := seedProduct(t, pool, "c@example.com", 10, "archived")

	row, err := store.SetStock(ctx, shop, product, 25)
	if err != nil || row.Stock != 25 || row.Reserved != 0 {
		t.Fatalf("SetStock = %+v, %v; want stock 25", row, err)
	}
	if _, err := store.SetStock(ctx, otherShop, product, 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other shop: err = %v, want ErrNotFound", err)
	}
	if _, err := store.SetStock(ctx, shop, 999999, 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: err = %v, want ErrNotFound", err)
	}
	if _, err := store.SetStock(ctx, archivedShop, archived, 1); !errors.Is(err, ErrArchived) {
		t.Fatalf("archived: err = %v, want ErrArchived", err)
	}

	if err := store.Reserve(ctx, pool, shop, product, 7); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetStock(ctx, shop, product, 6); !errors.Is(err, ErrBelowReserved) {
		t.Fatalf("below reserved: err = %v, want ErrBelowReserved", err)
	}
	if row, err := store.SetStock(ctx, shop, product, 7); err != nil || row.Reserved != 7 {
		t.Fatalf("equal to reserved = %+v, %v; want allowed", row, err)
	}
}

func TestPgReserveReleaseCommit(t *testing.T) {
	pool := newPgPool(t)
	store := NewPgStore(pool)
	ctx := context.Background()
	shop, product := seedProduct(t, pool, "a@example.com", 10, "active")

	read := func() (stock, reserved int) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT stock, reserved FROM products WHERE id = $1`, product).Scan(&stock, &reserved); err != nil {
			t.Fatal(err)
		}
		return
	}

	// The example from docs/DATA_MODEL.md section 4.
	if err := store.Reserve(ctx, pool, shop, product, 6); err != nil {
		t.Fatal(err)
	}
	if err := store.Reserve(ctx, pool, shop, product, 5); !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("err = %v, want ErrInsufficientStock", err)
	}
	if s, r := read(); s != 10 || r != 6 {
		t.Fatalf("after refused reserve: stock %d reserved %d, want 10 / 6", s, r)
	}
	if err := store.Release(ctx, pool, shop, product, 6); err != nil {
		t.Fatal(err)
	}
	if err := store.Reserve(ctx, pool, shop, product, 5); err != nil {
		t.Fatal(err)
	}
	if err := store.Commit(ctx, pool, shop, product, 5); err != nil {
		t.Fatal(err)
	}
	if s, r := read(); s != 5 || r != 0 {
		t.Fatalf("after commit: stock %d reserved %d, want 5 / 0", s, r)
	}

	if err := store.Release(ctx, pool, shop, product, 1); err == nil {
		t.Fatal("release of unreserved units accepted")
	}
	if err := store.Reserve(ctx, pool, shop, product, 0); !errors.Is(err, ErrValidation) {
		t.Fatalf("zero qty: err = %v, want ErrValidation", err)
	}
}

func TestPgConcurrentReservesNeverOversell(t *testing.T) {
	pool := newPgPool(t)
	store := NewPgStore(pool)
	ctx := context.Background()
	shop, product := seedProduct(t, pool, "a@example.com", 10, "active")

	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for range 6 {
		wg.Go(func() {
			if err := store.Reserve(ctx, pool, shop, product, 2); err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		})
	}
	wg.Wait()

	var reserved int
	if err := pool.QueryRow(ctx, `SELECT reserved FROM products WHERE id = $1`, product).Scan(&reserved); err != nil {
		t.Fatal(err)
	}
	if ok != 5 || reserved != 10 {
		t.Fatalf("successful reserves = %d, reserved = %d; want 5 and 10", ok, reserved)
	}
}
