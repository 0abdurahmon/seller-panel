package orderalert

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/database/dbtest"
)

func TestPgChatForShop(t *testing.T) {
	pool := dbtest.NewPool(t, "orderalert")
	ctx := context.Background()
	var linked, unlinked, linkedShop, unlinkedShop int64
	// The test database is reused between runs, so emails are unique per run.
	n := time.Now().UnixNano()
	chat := n%1_000_000_000 + 3_000_000 // chat ids are unique too
	if err := pool.QueryRow(ctx, `INSERT INTO sellers (email, password_hash, telegram_chat_id) VALUES ($1, 'x', $2) RETURNING id`,
		fmt.Sprintf("alert-linked-%d@example.com", n), chat).Scan(&linked); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO sellers (email, password_hash) VALUES ($1, 'x') RETURNING id`,
		fmt.Sprintf("alert-unlinked-%d@example.com", n)).Scan(&unlinked); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO shops (seller_id, name) VALUES ($1, 'Linked') RETURNING id`, linked).Scan(&linkedShop); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO shops (seller_id, name) VALUES ($1, 'Unlinked') RETURNING id`, unlinked).Scan(&unlinkedShop); err != nil {
		t.Fatal(err)
	}

	repo := NewPgRepository(pool)
	got, err := repo.ChatForShop(ctx, linkedShop)
	if err != nil || got != chat {
		t.Fatalf("linked shop: chat = %d, err = %v", got, err)
	}
	if _, err := repo.ChatForShop(ctx, unlinkedShop); !errors.Is(err, ErrNoChat) {
		t.Fatalf("unlinked shop: err = %v, want ErrNoChat", err)
	}
	if _, err := repo.ChatForShop(ctx, 999999999); !errors.Is(err, ErrNoChat) {
		t.Fatalf("missing shop: err = %v, want ErrNoChat", err)
	}
}

func TestPgLinesAreScopedByShop(t *testing.T) {
	pool := dbtest.NewPool(t, "orderalert")
	ctx := context.Background()
	n := time.Now().UnixNano()
	insert := func(sql string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := pool.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	shop := func(name string) int64 {
		seller := insert(`INSERT INTO sellers (email, password_hash) VALUES ($1, 'x') RETURNING id`,
			fmt.Sprintf("alert-lines-%s-%d@example.com", name, n))
		return insert(`INSERT INTO shops (seller_id, name) VALUES ($1, $2) RETURNING id`, seller, name)
	}
	mine, other := shop("mine"), shop("other")
	category := insert(`INSERT INTO categories (name) VALUES ($1) RETURNING id`, fmt.Sprintf("alert-%d", n))
	product := func(sku string) int64 {
		return insert(`INSERT INTO products (shop_id, category_id, title, price, sku) VALUES ($1, $2, 'Now renamed', 1, $3) RETURNING id`,
			mine, category, sku)
	}
	orderID := insert(`INSERT INTO orders (shop_id, buyer_name, buyer_phone, delivery_address, total)
		VALUES ($1, 'Aziz', '+998900000000', 'Tashkent', 40) RETURNING id`, mine)
	for _, line := range []struct {
		sku, title string
		price      int64
		quantity   int
	}{{"B", "Second product", 10, 1}, {"A", "First product", 15, 2}} {
		insert(`INSERT INTO order_items (order_id, product_id, title_snapshot, price_snapshot, quantity)
			VALUES ($1, $2, $3, $4, $5) RETURNING id`, orderID, product(line.sku), line.title, line.price, line.quantity)
	}

	repo := NewPgRepository(pool)
	got, err := repo.Lines(ctx, mine, orderID)
	want := []Line{{Title: "Second product", Quantity: 1, Price: 10}, {Title: "First product", Quantity: 2, Price: 15}}
	if err != nil || fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("lines = %v, err = %v, want the snapshots in order %v", got, err, want)
	}
	if _, err := repo.Lines(ctx, other, orderID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other shop: err = %v, want ErrNotFound", err)
	}
	if _, err := repo.Lines(ctx, mine, 999999999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing order: err = %v, want ErrNotFound", err)
	}
}
