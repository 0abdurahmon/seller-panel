package stock

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0abdurahmon/seller-panel/backend/internal/database"
)

// Needs TEST_DATABASE_URL and the products table; runs in its own database.
func newPgRepository(t *testing.T) (*PgRepository, *pgxpool.Pool) {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	url := ownDatabase(t, base)
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
	return NewPgRepository(pool), pool
}

// ownDatabase returns a separate "<name>_stock" database; other packages' tests truncate tables.
func ownDatabase(t *testing.T, base string) string {
	t.Helper()
	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("TEST_DATABASE_URL: %v", err)
	}
	name := strings.TrimPrefix(u.Path, "/") + "_stock"
	if !regexp.MustCompile(`^[A-Za-z0-9_]+$`).MatchString(name) {
		t.Fatalf("unexpected database name %q", name)
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize())
	var pgErr *pgconn.PgError
	// 42P04: already exists; 23505: created concurrently by another run.
	if err != nil && !(errors.As(err, &pgErr) && (pgErr.Code == "42P04" || pgErr.Code == "23505")) {
		t.Fatalf("create test database: %v", err)
	}
	u.Path = "/" + name
	return u.String()
}

// seedShop creates a seller and shop with a unique email; returns shop id.
func seedShop(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	ctx := context.Background()
	email := fmt.Sprintf("stock-%d@example.com", time.Now().UnixNano())
	var sellerID, shopID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO sellers (email, password_hash) VALUES ($1, 'x') RETURNING id`, email,
	).Scan(&sellerID); err != nil {
		t.Fatalf("seed seller: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO shops (seller_id, name) VALUES ($1, 'Stock test shop') RETURNING id`, sellerID,
	).Scan(&shopID); err != nil {
		t.Fatalf("seed shop: %v", err)
	}
	return shopID
}

// seedProduct creates one product in the shop and returns its id.
func seedProduct(t *testing.T, pool *pgxpool.Pool, shopID int64, stock, reserved int, status string) int64 {
	t.Helper()
	ctx := context.Background()
	suffix := time.Now().UnixNano()
	var categoryID, productID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO categories (name) VALUES ($1) RETURNING id`, fmt.Sprintf("Stock test %d", suffix),
	).Scan(&categoryID); err != nil {
		t.Fatalf("seed category: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO products (shop_id, category_id, title, price, sku, stock, reserved, status)
		 VALUES ($1, $2, 'Phone', 100000, $3, $4, $5, $6) RETURNING id`,
		shopID, categoryID, fmt.Sprintf("SKU-%d", suffix), stock, reserved, status,
	).Scan(&productID); err != nil {
		t.Fatalf("seed product: %v", err)
	}
	return productID
}

func readStock(t *testing.T, pool *pgxpool.Pool, productID int64) int {
	t.Helper()
	var stock int
	if err := pool.QueryRow(context.Background(),
		`SELECT stock FROM products WHERE id = $1`, productID,
	).Scan(&stock); err != nil {
		t.Fatal(err)
	}
	return stock
}

func TestPgSetStockRules(t *testing.T) {
	store, pool := newPgRepository(t)
	ctx := context.Background()
	shop := seedShop(t, pool)
	otherShop := seedShop(t, pool)
	product := seedProduct(t, pool, shop, 10, 4, "active")
	archived := seedProduct(t, pool, shop, 10, 0, "archived")

	row, err := store.SetStock(ctx, shop, product, 25, 10)
	if err != nil || row.Stock != 25 || row.Reserved != 4 {
		t.Fatalf("SetStock = %+v, %v; want stock 25 reserved 4", row, err)
	}
	if _, err := store.SetStock(ctx, otherShop, product, 1, 25); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other shop: err = %v, want ErrNotFound", err)
	}
	if _, err := store.SetStock(ctx, shop, 1<<40, 1, 0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: err = %v, want ErrNotFound", err)
	}
	if _, err := store.SetStock(ctx, shop, archived, 1, 10); !errors.Is(err, ErrArchived) {
		t.Fatalf("archived: err = %v, want ErrArchived", err)
	}
	cur, err := store.SetStock(ctx, shop, product, 3, 25)
	if !errors.Is(err, ErrBelowReserved) || cur.Stock != 25 || cur.Reserved != 4 {
		t.Fatalf("below reserved = %+v, %v; want ErrBelowReserved with current 25 / 4", cur, err)
	}
	if row, err := store.SetStock(ctx, shop, product, 4, 25); err != nil || row.Stock != 4 {
		t.Fatalf("equal to reserved = %+v, %v; want allowed", row, err)
	}
	if got := readStock(t, pool, product); got != 4 {
		t.Fatalf("stored stock %d, want 4", got)
	}
	if got := readStock(t, pool, archived); got != 10 {
		t.Fatalf("archived product's stock changed to %d", got)
	}
}

func TestPgSetStockStaleExpected(t *testing.T) {
	store, pool := newPgRepository(t)
	ctx := context.Background()
	shop := seedShop(t, pool)
	product := seedProduct(t, pool, shop, 10, 0, "draft")

	if row, err := store.SetStock(ctx, shop, product, 15, 10); err != nil || row.Stock != 15 {
		t.Fatalf("matching expected = %+v, %v; want stock 15", row, err)
	}
	cur, err := store.SetStock(ctx, shop, product, 20, 10) // still thinks it is 10
	if !errors.Is(err, ErrStockChanged) || cur.Stock != 15 {
		t.Fatalf("stale expected = %+v, %v; want ErrStockChanged with current 15", cur, err)
	}
	if got := readStock(t, pool, product); got != 15 {
		t.Fatalf("stored stock %d, want 15 (a refused change must not write)", got)
	}
}

// Eight tabs that saw 10 save at once: exactly one wins.
func TestPgConcurrentEditsDoNotOverwriteEachOther(t *testing.T) {
	store, pool := newPgRepository(t)
	ctx := context.Background()
	shop := seedShop(t, pool)
	product := seedProduct(t, pool, shop, 10, 0, "active")

	const tabs = 8
	var wg sync.WaitGroup
	var mu sync.Mutex
	var winners []int
	changed := 0
	for i := range tabs {
		wg.Go(func() {
			_, err := store.SetStock(ctx, shop, product, 100+i, 10)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				winners = append(winners, 100+i)
			case errors.Is(err, ErrStockChanged):
				changed++
			default:
				t.Errorf("tab %d: unexpected error %v", i, err)
			}
		})
	}
	wg.Wait()

	if len(winners) != 1 || changed != tabs-1 {
		t.Fatalf("winners = %v, stock_changed = %d; want exactly 1 winner", winners, changed)
	}
	if got := readStock(t, pool, product); got != winners[0] {
		t.Fatalf("stored stock %d, want the winner's %d", got, winners[0])
	}
}

// orderStock is the kind of interface the orders feature declares (ARCHITECTURE rule 4).
type orderStock interface {
	Reserve(ctx context.Context, db database.DBTX, shopID, productID int64, qty int) error
	Release(ctx context.Context, db database.DBTX, shopID, productID int64, qty int) error
	Commit(ctx context.Context, db database.DBTX, shopID, productID int64, qty int) error
}

var _ orderStock = (*PgRepository)(nil)

func TestReserveRejectsBadQuantityWithoutTouchingDB(t *testing.T) {
	ctx := context.Background()
	s := &PgRepository{}
	ops := map[string]func(context.Context, database.DBTX, int64, int64, int) error{
		"reserve": s.Reserve, "release": s.Release, "commit": s.Commit,
	}
	for name, op := range ops {
		for _, qty := range []int{0, -1, MaxStock + 1} {
			// A nil DBTX would panic if the query ran: validation must come first.
			if err := op(ctx, nil, 1, 1, qty); !errors.Is(err, ErrValidation) {
				t.Errorf("%s(qty %d): err = %v, want ErrValidation", name, qty, err)
			}
		}
	}
}

func readStockReserved(t *testing.T, store *PgRepository, productID int64) (int, int) {
	t.Helper()
	var stock, reserved int
	if err := store.pool.QueryRow(context.Background(),
		`SELECT stock, reserved FROM products WHERE id = $1`, productID,
	).Scan(&stock, &reserved); err != nil {
		t.Fatal(err)
	}
	return stock, reserved
}

// The example from DATA_MODEL.md section 4, step by step.
func TestPgReserveReleaseCommit(t *testing.T) {
	store, pool := newPgRepository(t)
	ctx := context.Background()
	shop := seedShop(t, pool)
	otherShop := seedShop(t, pool)
	product := seedProduct(t, pool, shop, 10, 0, "active")

	check := func(step string, wantStock, wantReserved int) {
		t.Helper()
		if s, r := readStockReserved(t, store, product); s != wantStock || r != wantReserved {
			t.Fatalf("%s: stock %d reserved %d, want %d / %d", step, s, r, wantStock, wantReserved)
		}
	}

	if err := store.Reserve(ctx, pool, shop, product, 6); err != nil {
		t.Fatal(err)
	}
	check("order A accepted", 10, 6)
	if err := store.Reserve(ctx, pool, shop, product, 5); !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("order B: err = %v, want ErrInsufficientStock", err)
	}
	check("order B refused", 10, 6)
	if err := store.Release(ctx, pool, shop, product, 6); err != nil {
		t.Fatal(err)
	}
	check("order A cancelled", 10, 0)
	if err := store.Reserve(ctx, pool, shop, product, 5); err != nil {
		t.Fatal(err)
	}
	if err := store.Commit(ctx, pool, shop, product, 5); err != nil {
		t.Fatal(err)
	}
	check("order B delivered", 5, 0)

	if err := store.Release(ctx, pool, shop, product, 1); !errors.Is(err, ErrReservationMismatch) {
		t.Fatalf("release unreserved: err = %v, want ErrReservationMismatch", err)
	}
	if err := store.Commit(ctx, pool, shop, product, 1); !errors.Is(err, ErrReservationMismatch) {
		t.Fatalf("commit unreserved: err = %v, want ErrReservationMismatch", err)
	}
	if err := store.Reserve(ctx, pool, otherShop, product, 1); !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("other shop: err = %v, want refused", err)
	}
	check("refusals changed nothing", 5, 0)
}

// Inside a rolled-back order transaction nothing stays reserved.
func TestPgReserveInsideTransactionRollsBack(t *testing.T) {
	store, pool := newPgRepository(t)
	ctx := context.Background()
	shop := seedShop(t, pool)
	product := seedProduct(t, pool, shop, 10, 0, "active")

	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if err := store.Reserve(ctx, tx, shop, product, 4); err != nil {
			return err
		}
		return store.Reserve(ctx, tx, shop, product, 7) // second item does not fit
	})
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("err = %v, want ErrInsufficientStock", err)
	}
	if _, r := readStockReserved(t, store, product); r != 0 {
		t.Fatalf("reserved %d after rollback, want 0", r)
	}
}

// Six orders of 2 against stock 10 at once: exactly five fit.
func TestPgConcurrentReservesNeverOversell(t *testing.T) {
	store, pool := newPgRepository(t)
	ctx := context.Background()
	shop := seedShop(t, pool)
	product := seedProduct(t, pool, shop, 10, 0, "active")

	var wg sync.WaitGroup
	var mu sync.Mutex
	ok, refused := 0, 0
	for range 6 {
		wg.Go(func() {
			err := store.Reserve(ctx, pool, shop, product, 2)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				ok++
			case errors.Is(err, ErrInsufficientStock):
				refused++
			default:
				t.Errorf("unexpected error %v", err)
			}
		})
	}
	wg.Wait()
	if _, r := readStockReserved(t, store, product); ok != 5 || refused != 1 || r != 10 {
		t.Fatalf("ok %d refused %d reserved %d; want 5, 1, 10", ok, refused, r)
	}
}
