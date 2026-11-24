package dashboard

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0abdurahmon/seller-panel/backend/internal/database/dbtest"
)

func newPgRepo(t *testing.T) (*PgRepository, *pgxpool.Pool) {
	t.Helper()
	pool := dbtest.NewPool(t, "dashboard")
	return NewPgRepository(pool), pool
}

func uniqueName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func seedShop(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	ctx := context.Background()
	var sellerID, shopID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO sellers (email, password_hash) VALUES ($1, 'x') RETURNING id`, uniqueName("dashboard")+"@example.com").Scan(&sellerID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO shops (seller_id, name) VALUES ($1, 'Dashboard shop') RETURNING id`, sellerID).Scan(&shopID); err != nil {
		t.Fatal(err)
	}
	return shopID
}

func seedProduct(t *testing.T, pool *pgxpool.Pool, shopID int64, title string) int64 {
	t.Helper()
	ctx := context.Background()
	var categoryID, productID int64
	if err := pool.QueryRow(ctx, `INSERT INTO categories (name) VALUES ($1) RETURNING id`, uniqueName("category")).Scan(&categoryID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO products (shop_id, category_id, title, price, sku, stock, status)
		 VALUES ($1, $2, $3, 1000, $4, 100, 'active') RETURNING id`,
		shopID, categoryID, title, uniqueName("SKU")).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	return productID
}

type line struct {
	productID int64
	quantity  int64
	price     int64
}

// seedOrder inserts an order with exact times; delivered orders keep the time they were delivered.
func seedOrder(t *testing.T, pool *pgxpool.Pool, shopID int64, status string, createdAt, changedAt time.Time, lines ...line) int64 {
	t.Helper()
	ctx := context.Background()
	var total int64
	for _, l := range lines {
		total += l.price * l.quantity
	}
	if total == 0 {
		total = 1000
	}
	var reason any
	if status == "cancelled" {
		reason = "seeded"
	}
	var id int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO orders (shop_id, status, buyer_name, buyer_phone, delivery_address, total, cancel_reason, created_at, status_changed_at)
		 VALUES ($1, $2::order_status, 'Buyer', '+998901234567', 'Tashkent', $3, $4, $5, $6) RETURNING id`,
		shopID, status, total, reason, createdAt, changedAt).Scan(&id); err != nil {
		t.Fatal(err)
	}
	for _, l := range lines {
		if _, err := pool.Exec(ctx,
			`INSERT INTO order_items (order_id, product_id, title_snapshot, price_snapshot, quantity) VALUES ($1, $2, 'snapshot', $3, $4)`,
			id, l.productID, l.price, l.quantity); err != nil {
			t.Fatal(err)
		}
	}
	return id
}

func pgWindow(t *testing.T) Window {
	t.Helper()
	return newTestService(t, nil).window(now)
}

func TestPgSummaryCountsAndSumsByWindow(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	shop, other, empty := seedShop(t, pool), seedShop(t, pool), seedShop(t, pool)
	w := pgWindow(t)
	sec := time.Second

	phone := seedProduct(t, pool, shop, "Phone")
	foreign := seedProduct(t, pool, other, "Foreign")

	seedOrder(t, pool, shop, "new", w.TodayStart, w.TodayStart)
	seedOrder(t, pool, shop, "new", w.TodayStart.Add(-sec), w.TodayStart.Add(-sec))
	seedOrder(t, pool, shop, "accepted", w.TodayEnd.Add(-sec), w.TodayEnd.Add(-sec))
	seedOrder(t, pool, shop, "new", w.TodayEnd, w.TodayEnd)
	seedOrder(t, pool, shop, "packed", w.From30D, w.From30D)
	seedOrder(t, pool, shop, "handed_to_courier", w.TodayStart, w.TodayStart)
	seedOrder(t, pool, shop, "cancelled", w.TodayStart, w.TodayStart)

	seedOrder(t, pool, shop, "delivered", w.From30D, w.From7D, line{phone, 1, 1000})
	seedOrder(t, pool, shop, "delivered", w.From30D, w.From7D.Add(-sec), line{phone, 1, 200})
	seedOrder(t, pool, shop, "delivered", w.From30D, w.From30D, line{phone, 1, 30})
	seedOrder(t, pool, shop, "delivered", w.From30D, w.From30D.Add(-sec), line{phone, 1, 5})
	seedOrder(t, pool, shop, "delivered", w.TodayEnd, w.TodayEnd, line{phone, 1, 7000})
	seedOrder(t, pool, shop, "cancelled", w.From7D, w.From7D)

	seedOrder(t, pool, other, "new", w.TodayStart, w.TodayStart)
	seedOrder(t, pool, other, "delivered", w.TodayStart, w.TodayStart, line{foreign, 50, 100_000})

	got, err := repo.Summary(ctx, shop, w)
	if err != nil {
		t.Fatal(err)
	}
	want := Summary{
		TodayOrders:      4,
		WaitingForAction: 5,
		Revenue7D:        1000,
		Revenue30D:       1230,
		TopProducts:      []TopProduct{{ProductID: phone, Title: "Phone", QuantitySold: 3, Revenue: 1230}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Summary = %+v, want %+v", got, want)
	}

	got, err = repo.Summary(ctx, other, w)
	wantOther := Summary{
		TodayOrders:      2,
		WaitingForAction: 1,
		Revenue7D:        5_000_000,
		Revenue30D:       5_000_000,
		TopProducts:      []TopProduct{{ProductID: foreign, Title: "Foreign", QuantitySold: 50, Revenue: 5_000_000}},
	}
	if err != nil || !reflect.DeepEqual(got, wantOther) {
		t.Fatalf("other shop Summary = %+v, %v; want %+v", got, err, wantOther)
	}

	got, err = repo.Summary(ctx, empty, w)
	if err != nil || !reflect.DeepEqual(got, Summary{}) {
		t.Fatalf("empty shop Summary = %+v, %v; want zeros", got, err)
	}
}

func TestPgTopProductsRankAndLimit(t *testing.T) {
	repo, pool := newPgRepo(t)
	shop := seedShop(t, pool)
	w := pgWindow(t)
	delivered := w.TodayStart.Add(-48 * time.Hour)

	byQuantity := make([]int64, 8)
	for i, l := range []struct {
		title    string
		quantity int64
		price    int64
	}{
		{"seven units", 7, 10},
		{"one unit", 1, 10},
		{"tie cheap", 4, 10},
		{"tie dear", 4, 20},
		{"tie a", 3, 10},
		{"tie b", 3, 10},
		{"two units", 2, 10},
		{"nine units", 9, 10},
	} {
		byQuantity[i] = seedProduct(t, pool, shop, l.title)
		seedOrder(t, pool, shop, "delivered", delivered, delivered, line{byQuantity[i], l.quantity, l.price})
	}

	got, err := repo.Summary(context.Background(), shop, w)
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for _, p := range got.TopProducts {
		ids = append(ids, p.ProductID)
	}
	// nine, seven, the 4-unit tie by revenue, then the 3-unit tie by id; the rest is cut off.
	want := []int64{byQuantity[7], byQuantity[0], byQuantity[3], byQuantity[2], byQuantity[4]}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("top products = %v, want %v", ids, want)
	}
	if len(got.TopProducts) != 5 {
		t.Fatalf("%d top products, want 5", len(got.TopProducts))
	}
}

func TestPgTopProductsSumLinesAcrossOrdersAndKeepArchivedProducts(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	shop := seedShop(t, pool)
	w := pgWindow(t)
	delivered := w.TodayStart.Add(-time.Hour)

	phone := seedProduct(t, pool, shop, "Phone")
	old := seedProduct(t, pool, shop, "Old model")
	seedOrder(t, pool, shop, "delivered", delivered, delivered, line{phone, 2, 1000}, line{old, 1, 500})
	seedOrder(t, pool, shop, "delivered", delivered, delivered, line{phone, 3, 900})
	seedOrder(t, pool, shop, "cancelled", delivered, delivered, line{phone, 100, 1000})
	if _, err := pool.Exec(ctx, `UPDATE products SET status = 'archived', title = 'Old model (archived)' WHERE id = $1`, old); err != nil {
		t.Fatal(err)
	}

	got, err := repo.Summary(ctx, shop, w)
	if err != nil {
		t.Fatal(err)
	}
	want := []TopProduct{
		{ProductID: phone, Title: "Phone", QuantitySold: 5, Revenue: 2*1000 + 3*900},
		{ProductID: old, Title: "Old model (archived)", QuantitySold: 1, Revenue: 500},
	}
	if !reflect.DeepEqual(got.TopProducts, want) {
		t.Fatalf("top products = %+v, want %+v", got.TopProducts, want)
	}
	if got.Revenue30D != 2*1000+500+3*900 {
		t.Fatalf("revenue 30d = %d, want the order totals %d", got.Revenue30D, 2*1000+500+3*900)
	}
}
