package balance

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0abdurahmon/seller-panel/backend/internal/database/dbtest"
)

func newPgRepo(t *testing.T) (*PgRepository, *pgxpool.Pool) {
	t.Helper()
	pool := dbtest.NewPool(t, "balance")
	return NewPgRepository(pool), pool
}

func seedShop(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	ctx := context.Background()
	var sellerID, shopID int64
	email := fmt.Sprintf("balance-%d@example.com", time.Now().UnixNano())
	if err := pool.QueryRow(ctx,
		`INSERT INTO sellers (email, password_hash) VALUES ($1, 'x') RETURNING id`, email).Scan(&sellerID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO shops (seller_id, name) VALUES ($1, 'Test shop') RETURNING id`, sellerID).Scan(&shopID); err != nil {
		t.Fatal(err)
	}
	return shopID
}

// newOrder creates a delivered order, the row a balance entry must point to.
func newOrder(t *testing.T, pool *pgxpool.Pool, shopID int64) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO orders (shop_id, status, buyer_name, buyer_phone, delivery_address, total)
		 VALUES ($1, 'delivered', 'Buyer', '+998901234567', 'Tashkent', 1000) RETURNING id`, shopID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func insertEntry(pool *pgxpool.Pool, shopID int64, orderID *int64, typ string, amount int64, at time.Time) error {
	_, err := pool.Exec(context.Background(),
		`INSERT INTO balance_entries (shop_id, order_id, type, amount, created_at) VALUES ($1, $2, $3::balance_entry_type, $4, $5)`,
		shopID, orderID, typ, amount, at)
	return err
}

func mustInsert(t *testing.T, pool *pgxpool.Pool, shopID int64, orderID *int64, typ string, amount int64, at time.Time) {
	t.Helper()
	if err := insertEntry(pool, shopID, orderID, typ, amount, at); err != nil {
		t.Fatalf("insert %s %d: %v", typ, amount, err)
	}
}

func mustDeliver(t *testing.T, pool *pgxpool.Pool, shopID, total int64, at time.Time) int64 {
	t.Helper()
	order := newOrder(t, pool, shopID)
	mustInsert(t, pool, shopID, &order, "sale", total, at)
	mustInsert(t, pool, shopID, &order, "commission", -(total * 10 / 100), at)
	return order
}

func TestPgSummary(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	shop, other, empty := seedShop(t, pool), seedShop(t, pool), seedShop(t, pool)

	mustDeliver(t, pool, shop, 250000, t0)
	mustDeliver(t, pool, shop, 12345, t0.Add(time.Hour))
	mustDeliver(t, pool, other, 999000, t0)
	mustInsert(t, pool, shop, nil, "payout", -50000, t0.Add(2*time.Hour))

	got, err := repo.Summary(ctx, shop)
	if err != nil {
		t.Fatal(err)
	}
	if want := (Summary{Available: 186111, Earned: 262345, Commission: -26234}); got != want {
		t.Fatalf("Summary = %+v, want %+v", got, want)
	}
	got, err = repo.Summary(ctx, other)
	if err != nil || got != (Summary{Available: 899100, Earned: 999000, Commission: -99900}) {
		t.Fatalf("other shop Summary = %+v, %v", got, err)
	}
	got, err = repo.Summary(ctx, empty)
	if err != nil || got != (Summary{}) {
		t.Fatalf("empty shop Summary = %+v, %v; want zeros", got, err)
	}
}

func TestPgEntriesNewestFirstWithStableOrder(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	shop, other := seedShop(t, pool), seedShop(t, pool)

	first := mustDeliver(t, pool, shop, 1000, t0) // sale and commission share one timestamp
	mustDeliver(t, pool, other, 5000, t0.Add(time.Minute))
	mustInsert(t, pool, shop, nil, "payout", -300, t0.Add(time.Hour))

	items, total, err := repo.Entries(ctx, shop, 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(items) != 3 {
		t.Fatalf("total %d, items %d; want 3 and 3 (the other shop's entries must not count)", total, len(items))
	}
	if items[0].Type != EntryPayout || items[0].OrderID != nil || items[0].Amount != -300 {
		t.Fatalf("items[0] = %+v, want the newest entry, a payout with no order", items[0])
	}
	if items[1].Type != EntryCommission || items[2].Type != EntrySale || *items[1].OrderID != first || *items[2].OrderID != first {
		t.Fatalf("items[1:] = %+v, %+v; want commission then sale of order %d (later id first)", items[1], items[2], first)
	}
	if items[0].CreatedAt.Location() != time.UTC {
		t.Fatalf("created_at is in %s, want UTC", items[0].CreatedAt.Location())
	}
}

func TestPgEntriesPagination(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	shop := seedShop(t, pool)
	for i := range 25 {
		order := newOrder(t, pool, shop)
		mustInsert(t, pool, shop, &order, "sale", int64(100+i), t0.Add(time.Duration(i)*time.Minute))
	}

	page1, total, err := repo.Entries(ctx, shop, 20, 0)
	if err != nil || total != 25 || len(page1) != 20 || page1[0].Amount != 124 {
		t.Fatalf("page 1: %d items, total %d, first amount %v, err %v; want 20, 25, 124", len(page1), total, page1[0].Amount, err)
	}
	page2, _, err := repo.Entries(ctx, shop, 20, 20)
	if err != nil || len(page2) != 5 || page2[4].Amount != 100 {
		t.Fatalf("page 2: %d items, err %v; want 5 ending with the oldest (100)", len(page2), err)
	}
	past, total, err := repo.Entries(ctx, shop, 20, 400)
	if err != nil || len(past) != 0 || total != 25 {
		t.Fatalf("past the end: %d items, total %d, err %v; want 0 and 25", len(past), total, err)
	}
}

// The database is the second line of defence: bad money rows must be refused.
func TestPgConstraintsRefuseBrokenEntries(t *testing.T) {
	_, pool := newPgRepo(t)
	shop := seedShop(t, pool)
	at := time.Now().UTC()

	cases := []struct {
		name       string
		orderID    *int64
		typ        string
		amount     int64
		constraint string
	}{
		{"sale of zero", ptr(newOrder(t, pool, shop)), "sale", 0, "balance_entries_sign"},
		{"negative sale", ptr(newOrder(t, pool, shop)), "sale", -5, "balance_entries_sign"},
		{"positive commission", ptr(newOrder(t, pool, shop)), "commission", 5, "balance_entries_sign"},
		{"positive payout", nil, "payout", 5, "balance_entries_sign"},
		{"zero payout", nil, "payout", 0, "balance_entries_sign"},
		{"sale without an order", nil, "sale", 100, "balance_entries_order_matches_type"},
		{"payout with an order", ptr(newOrder(t, pool, shop)), "payout", -100, "balance_entries_order_matches_type"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := insertEntry(pool, shop, c.orderID, c.typ, c.amount, at)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.ConstraintName != c.constraint {
				t.Fatalf("err = %v, want constraint %s", err, c.constraint)
			}
		})
	}

	// A zero commission is legal (commission percent 0), and payouts can repeat.
	order := newOrder(t, pool, shop)
	mustInsert(t, pool, shop, &order, "sale", 100, at)
	mustInsert(t, pool, shop, &order, "commission", 0, at)
	mustInsert(t, pool, shop, nil, "payout", -10, at)
	mustInsert(t, pool, shop, nil, "payout", -20, at)
}

func TestPgOneSaleAndOneCommissionPerOrder(t *testing.T) {
	_, pool := newPgRepo(t)
	shop := seedShop(t, pool)
	order := newOrder(t, pool, shop)
	at := time.Now().UTC()

	mustInsert(t, pool, shop, &order, "sale", 1000, at)
	err := insertEntry(pool, shop, &order, "sale", 1000, at)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.ConstraintName != "balance_entries_order_type_key" {
		t.Fatalf("second sale err = %v, want balance_entries_order_type_key (a second delivery must not double count)", err)
	}
	mustInsert(t, pool, shop, &order, "commission", -100, at)
}
