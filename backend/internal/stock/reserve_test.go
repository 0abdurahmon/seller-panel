package stock

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestReserveRejectsBadQuantityWithoutTouchingDB(t *testing.T) {
	ctx := context.Background()
	ops := map[string]func(context.Context, Execer, int64, int64, int) error{
		"reserve": Reserve, "release": Release, "commit": Commit,
	}
	for name, op := range ops {
		for _, qty := range []int{0, -1, MaxStock + 1} {
			// A nil Execer would panic if the query ran: validation must come first.
			if err := op(ctx, nil, 1, 1, qty); !errors.Is(err, ErrValidation) {
				t.Errorf("%s(qty %d): err = %v, want ErrValidation", name, qty, err)
			}
		}
	}
}

func readStockReserved(t *testing.T, store *PgStore, productID int64) (int, int) {
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
	store, pool := newPgStore(t)
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

	if err := Reserve(ctx, pool, shop, product, 6); err != nil {
		t.Fatal(err)
	}
	check("order A accepted", 10, 6)
	if err := Reserve(ctx, pool, shop, product, 5); !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("order B: err = %v, want ErrInsufficientStock", err)
	}
	check("order B refused", 10, 6)
	if err := Release(ctx, pool, shop, product, 6); err != nil {
		t.Fatal(err)
	}
	check("order A cancelled", 10, 0)
	if err := Reserve(ctx, pool, shop, product, 5); err != nil {
		t.Fatal(err)
	}
	if err := Commit(ctx, pool, shop, product, 5); err != nil {
		t.Fatal(err)
	}
	check("order B delivered", 5, 0)

	if err := Release(ctx, pool, shop, product, 1); !errors.Is(err, ErrReservationMismatch) {
		t.Fatalf("release unreserved: err = %v, want ErrReservationMismatch", err)
	}
	if err := Commit(ctx, pool, shop, product, 1); !errors.Is(err, ErrReservationMismatch) {
		t.Fatalf("commit unreserved: err = %v, want ErrReservationMismatch", err)
	}
	if err := Reserve(ctx, pool, otherShop, product, 1); !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("other shop: err = %v, want refused", err)
	}
	check("refusals changed nothing", 5, 0)
}

// Inside a rolled-back order transaction nothing stays reserved.
func TestPgReserveInsideTransactionRollsBack(t *testing.T) {
	store, pool := newPgStore(t)
	ctx := context.Background()
	shop := seedShop(t, pool)
	product := seedProduct(t, pool, shop, 10, 0, "active")

	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if err := Reserve(ctx, tx, shop, product, 4); err != nil {
			return err
		}
		return Reserve(ctx, tx, shop, product, 7) // second item does not fit
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
	store, pool := newPgStore(t)
	ctx := context.Background()
	shop := seedShop(t, pool)
	product := seedProduct(t, pool, shop, 10, 0, "active")

	var wg sync.WaitGroup
	var mu sync.Mutex
	ok, refused := 0, 0
	for range 6 {
		wg.Go(func() {
			err := Reserve(ctx, pool, shop, product, 2)
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
