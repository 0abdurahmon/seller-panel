package migrations_test

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

var ctx = context.Background()

type fixture struct {
	pool    *pgxpool.Pool
	shop    int64
	product int64
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	pool := dbtest.NewPool(t, "schema")
	n := time.Now().UnixNano()
	var seller, shop, category, product int64
	scan := func(dest *int64, sql string, args ...any) {
		t.Helper()
		if err := pool.QueryRow(ctx, sql, args...).Scan(dest); err != nil {
			t.Fatal(err)
		}
	}
	scan(&seller, `INSERT INTO sellers (email, password_hash) VALUES ($1, 'x') RETURNING id`, fmt.Sprintf("schema-%d@example.com", n))
	scan(&shop, `INSERT INTO shops (seller_id, name) VALUES ($1, 'Schema shop') RETURNING id`, seller)
	scan(&category, `INSERT INTO categories (name) VALUES ($1) RETURNING id`, fmt.Sprintf("schema-%d", n))
	scan(&product,
		`INSERT INTO products (shop_id, category_id, title, price, sku, stock) VALUES ($1, $2, 'Phone', 1000, $3, 10) RETURNING id`,
		shop, category, fmt.Sprintf("SCHEMA-%d", n))
	return fixture{pool: pool, shop: shop, product: product}
}

func (f fixture) order(status, cancelReason, buyer string, total int64) (int64, error) {
	var id int64
	err := f.pool.QueryRow(ctx,
		`INSERT INTO orders (shop_id, status, buyer_name, buyer_phone, delivery_address, total, cancel_reason)
		 VALUES ($1, $2::order_status, $3, '+998901234567', 'Tashkent', $4, NULLIF($5, '')) RETURNING id`,
		f.shop, status, buyer, total, cancelReason).Scan(&id)
	return id, err
}

func (f fixture) mustOrder(t *testing.T) int64 {
	t.Helper()
	id, err := f.order("new", "", "Buyer", 5000)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func wantConstraint(t *testing.T, err error, name string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.ConstraintName != name {
		t.Fatalf("err = %v, want constraint %s", err, name)
	}
}

func TestOrderDefaultsAndNumberSequence(t *testing.T) {
	f := newFixture(t)
	first, second := f.mustOrder(t), f.mustOrder(t)

	var status string
	var n1, n2 int64
	if err := f.pool.QueryRow(ctx, `SELECT status::text, number FROM orders WHERE id = $1`, first).Scan(&status, &n1); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT number FROM orders WHERE id = $1`, second).Scan(&n2); err != nil {
		t.Fatal(err)
	}
	if status != "new" || n1 < 100000 || n2 != n1+1 {
		t.Fatalf("status %q, numbers %d and %d; want new, at least 100000 and consecutive", status, n1, n2)
	}
}

func TestOrderConstraints(t *testing.T) {
	f := newFixture(t)
	cases := []struct {
		name       string
		status     string
		reason     string
		buyer      string
		total      int64
		constraint string
	}{
		{"blank buyer name", "new", "", "   ", 100, "orders_buyer_name_check"},
		{"zero total", "new", "", "Buyer", 0, "orders_total_check"},
		{"negative total", "new", "", "Buyer", -5, "orders_total_check"},
		{"cancelled without a reason", "cancelled", "", "Buyer", 100, "orders_cancel_reason_matches_status"},
		{"cancelled with a blank reason", "cancelled", "  ", "Buyer", 100, "orders_cancel_reason_matches_status"},
		{"reason on an order that is not cancelled", "accepted", "changed my mind", "Buyer", 100, "orders_cancel_reason_matches_status"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := f.order(c.status, c.reason, c.buyer, c.total)
			wantConstraint(t, err, c.constraint)
		})
	}
	if _, err := f.order("cancelled", "out of stock", "Buyer", 100); err != nil {
		t.Fatalf("a cancelled order with a reason must be accepted: %v", err)
	}
	_, err := f.pool.Exec(ctx,
		`INSERT INTO orders (shop_id, buyer_name, buyer_phone, delivery_address, total, status) VALUES ($1, 'B', 'p', 'a', 1, 'shipped')`, f.shop)
	if err == nil {
		t.Fatal("an unknown status was accepted")
	}
}

func TestOrderItemConstraints(t *testing.T) {
	f := newFixture(t)
	order := f.mustOrder(t)
	add := func(orderID, productID, price int64, qty int) error {
		_, err := f.pool.Exec(ctx,
			`INSERT INTO order_items (order_id, product_id, title_snapshot, price_snapshot, quantity) VALUES ($1, $2, 'Phone', $3, $4)`,
			orderID, productID, price, qty)
		return err
	}

	if err := add(order, f.product, 1000, 2); err != nil {
		t.Fatal(err)
	}
	wantConstraint(t, add(order, f.product, 1000, 1), "order_items_order_product_key")
	other := f.mustOrder(t)
	wantConstraint(t, add(other, f.product, 1000, 0), "order_items_quantity_check")
	wantConstraint(t, add(other, f.product, 1000, -1), "order_items_quantity_check")
	wantConstraint(t, add(other, f.product, 0, 1), "order_items_price_check")
	wantConstraint(t, add(order+1_000_000, f.product, 1000, 1), "order_items_order_id_fkey")
	wantConstraint(t, add(other, f.product+1_000_000, 1000, 1), "order_items_product_id_fkey")
}

func TestOrderStatusHistory(t *testing.T) {
	f := newFixture(t)
	order := f.mustOrder(t)
	insert := func(from any, to string) error {
		_, err := f.pool.Exec(ctx,
			`INSERT INTO order_status_history (order_id, from_status, to_status) VALUES ($1, $2::order_status, $3::order_status)`, order, from, to)
		return err
	}
	if err := insert(nil, "new"); err != nil {
		t.Fatalf("the creation row has no from_status: %v", err)
	}
	if err := insert("new", "accepted"); err != nil {
		t.Fatal(err)
	}
	if err := insert("new", "shipped"); err == nil {
		t.Fatal("an unknown status was accepted in the history")
	}
}

func TestBalanceEntriesPointToRealOrders(t *testing.T) {
	f := newFixture(t)
	order := f.mustOrder(t)
	add := func(orderID any, typ string, amount int64) error {
		_, err := f.pool.Exec(ctx,
			`INSERT INTO balance_entries (shop_id, order_id, type, amount) VALUES ($1, $2, $3::balance_entry_type, $4)`, f.shop, orderID, typ, amount)
		return err
	}
	wantConstraint(t, add(order+1_000_000, "sale", 100), "balance_entries_order_id_fkey")
	if err := add(order, "sale", 5000); err != nil {
		t.Fatalf("a sale of a real order failed: %v", err)
	}
	if err := add(order, "commission", -500); err != nil {
		t.Fatal(err)
	}
	if err := add(nil, "payout", -100); err != nil {
		t.Fatalf("a payout has no order and must still work: %v", err)
	}
}
