package order

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0abdurahmon/seller-panel/backend/internal/database/dbtest"
)

func newPgRepo(t *testing.T) (*PgRepository, *pgxpool.Pool) {
	t.Helper()
	pool := dbtest.NewPool(t, "order")
	return NewPgRepository(pool), pool
}

func newPgService(t *testing.T, repo *PgRepository) *Service {
	t.Helper()
	svc, err := NewService(repo, 10)
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func uniqueName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func seedShop(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	ctx := context.Background()
	var sellerID, shopID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO sellers (email, password_hash) VALUES ($1, 'x') RETURNING id`, uniqueName("order")+"@example.com").Scan(&sellerID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO shops (seller_id, name) VALUES ($1, 'Order shop') RETURNING id`, sellerID).Scan(&shopID); err != nil {
		t.Fatal(err)
	}
	return shopID
}

func seedProduct(t *testing.T, pool *pgxpool.Pool, shopID int64, stock, reserved int) int64 {
	t.Helper()
	ctx := context.Background()
	var categoryID, productID int64
	if err := pool.QueryRow(ctx, `INSERT INTO categories (name) VALUES ($1) RETURNING id`, uniqueName("category")).Scan(&categoryID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO products (shop_id, category_id, title, price, sku, stock, reserved, status)
		 VALUES ($1, $2, 'Phone', 1000, $3, $4, $5, 'active') RETURNING id`,
		shopID, categoryID, uniqueName("SKU"), stock, reserved).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	return productID
}

type seedItem struct {
	productID int64
	quantity  int
	price     int64
}

func seedOrder(t *testing.T, pool *pgxpool.Pool, shopID int64, status Status, createdAt time.Time, items ...seedItem) int64 {
	t.Helper()
	ctx := context.Background()
	var total int64
	for _, it := range items {
		total += it.price * int64(it.quantity)
	}
	if total == 0 {
		total = 1000
	}
	var reason any
	if status == StatusCancelled {
		reason = "seeded"
	}
	var id int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO orders (shop_id, status, buyer_name, buyer_phone, delivery_address, total, cancel_reason, created_at, status_changed_at)
		 VALUES ($1, $2::order_status, 'Buyer', '+998901234567', 'Tashkent', $3, $4, $5, $5) RETURNING id`,
		shopID, string(status), total, reason, createdAt).Scan(&id); err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if _, err := pool.Exec(ctx,
			`INSERT INTO order_items (order_id, product_id, title_snapshot, price_snapshot, quantity) VALUES ($1, $2, 'Phone', $3, $4)`,
			id, it.productID, it.price, it.quantity); err != nil {
			t.Fatal(err)
		}
	}
	return id
}

func productStock(t *testing.T, pool *pgxpool.Pool, productID int64) (stock, reserved int) {
	t.Helper()
	if err := pool.QueryRow(context.Background(),
		`SELECT stock, reserved FROM products WHERE id = $1`, productID).Scan(&stock, &reserved); err != nil {
		t.Fatal(err)
	}
	return stock, reserved
}

func count(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestPgGetListAndOrderNumber(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	shop := seedShop(t, pool)
	first, second := seedProduct(t, pool, shop, 10, 0), seedProduct(t, pool, shop, 10, 0)
	id := seedOrder(t, pool, shop, StatusNew, time.Now(), seedItem{second, 1, 500}, seedItem{first, 2, 1000})

	o, err := repo.Get(ctx, shop, id)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(o.Number, "#") || len(o.Number) < 7 || o.Status != StatusNew || o.Total != 2500 {
		t.Fatalf("order = %+v", o)
	}
	if len(o.Items) != 2 || o.Items[0].ProductID != second || o.Items[1].ProductID != first {
		t.Fatalf("Get items must follow insertion order, got %+v", o.Items)
	}
	if o.History == nil || len(o.History) != 0 || o.CreatedAt.Location() != time.UTC {
		t.Fatalf("history %v, created_at zone %v", o.History, o.CreatedAt.Location())
	}

	page, err := repo.List(ctx, shop, Filter{Page: 1, PageSize: 20})
	if err != nil || page.Total != 1 || len(page.Items) != 1 || !strings.HasPrefix(page.Items[0].Number, "#") {
		t.Fatalf("list = %+v, %v", page, err)
	}
}

func TestPgLockOrderReturnsItemsByProductID(t *testing.T) {
	repo, pool := newPgRepo(t)
	shop := seedShop(t, pool)
	low, high := seedProduct(t, pool, shop, 10, 0), seedProduct(t, pool, shop, 10, 0)
	id := seedOrder(t, pool, shop, StatusNew, time.Now(), seedItem{high, 1, 100}, seedItem{low, 1, 100})

	err := repo.InTx(context.Background(), func(tx TxRepository) error {
		o, err := tx.LockOrder(context.Background(), shop, id)
		if err != nil {
			return err
		}
		if len(o.Items) != 2 || o.Items[0].ProductID != low || o.Items[1].ProductID != high {
			t.Errorf("locked items = %+v, want ascending product_id so two orders always lock in the same order", o.Items)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPgEverythingIsScopedByShop(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	shopA, shopB := seedShop(t, pool), seedShop(t, pool)
	product := seedProduct(t, pool, shopA, 10, 0)
	id := seedOrder(t, pool, shopA, StatusNew, time.Now(), seedItem{product, 1, 100})

	if _, err := repo.Get(ctx, shopB, id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get across shops: %v, want ErrNotFound", err)
	}
	page, err := repo.List(ctx, shopB, Filter{Page: 1, PageSize: 20})
	if err != nil || page.Total != 0 || len(page.Items) != 0 {
		t.Fatalf("shop B list = %+v, %v", page, err)
	}
	err = repo.InTx(ctx, func(tx TxRepository) error {
		if _, err := tx.LockOrder(ctx, shopB, id); !errors.Is(err, ErrNotFound) {
			t.Errorf("LockOrder across shops: %v, want ErrNotFound", err)
		}
		if err := tx.Reserve(ctx, shopB, product, 1); !errors.Is(err, ErrProductNotFound) {
			t.Errorf("Reserve across shops: %v, want ErrProductNotFound", err)
		}
		if err := tx.SetStatus(ctx, shopB, id, StatusAccepted, nil, time.Now()); !errors.Is(err, ErrNotFound) {
			t.Errorf("SetStatus across shops: %v, want ErrNotFound", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, reserved := productStock(t, pool, product); reserved != 0 {
		t.Fatalf("another shop's calls reserved %d units", reserved)
	}
}

func TestPgTransitionsWriteStockHistoryAndBalance(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	svc := newPgService(t, repo)
	shop := seedShop(t, pool)
	product := seedProduct(t, pool, shop, 10, 0)
	id := seedOrder(t, pool, shop, StatusNew, time.Now(), seedItem{product, 3, 1005})

	step := func(to Status) Order {
		t.Helper()
		o, err := svc.Transition(ctx, shop, id, TransitionRequest{To: to})
		if err != nil {
			t.Fatalf("to %s: %v", to, err)
		}
		return o
	}
	step(StatusAccepted)
	if stock, reserved := productStock(t, pool, product); stock != 10 || reserved != 3 {
		t.Fatalf("after accept stock/reserved = %d/%d, want 10/3", stock, reserved)
	}
	step(StatusPacked)
	step(StatusHandedToCourier)
	o := step(StatusDelivered)
	if stock, reserved := productStock(t, pool, product); stock != 7 || reserved != 0 {
		t.Fatalf("after delivery stock/reserved = %d/%d, want 7/0", stock, reserved)
	}
	if len(o.History) != 4 {
		t.Fatalf("returned history = %+v", o.History)
	}

	got, err := repo.Get(ctx, shop, id)
	if err != nil {
		t.Fatal(err)
	}
	wantPath := []Status{StatusAccepted, StatusPacked, StatusHandedToCourier, StatusDelivered}
	if len(got.History) != 4 {
		t.Fatalf("stored history = %+v", got.History)
	}
	for i, h := range got.History {
		if h.ToStatus != wantPath[i] {
			t.Errorf("history[%d].to = %s, want %s", i, h.ToStatus, wantPath[i])
		}
		wantFrom := StatusNew
		if i > 0 {
			wantFrom = wantPath[i-1]
		}
		if h.FromStatus == nil || *h.FromStatus != wantFrom {
			t.Errorf("history[%d].from = %v, want %s", i, h.FromStatus, wantFrom)
		}
	}

	var sale, commission int64
	if err := pool.QueryRow(ctx,
		`SELECT coalesce(sum(amount) FILTER (WHERE type = 'sale'), 0), coalesce(sum(amount) FILTER (WHERE type = 'commission'), 0)
		   FROM balance_entries WHERE order_id = $1`, id).Scan(&sale, &commission); err != nil {
		t.Fatal(err)
	}
	if sale != 3015 || commission != -301 {
		t.Fatalf("balance sale/commission = %d/%d, want 3015/-301 (10%% of 3015 rounds down)", sale, commission)
	}
}

func TestPgCancelReleasesAndKeepsReason(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	svc := newPgService(t, repo)
	shop := seedShop(t, pool)
	product := seedProduct(t, pool, shop, 10, 4)
	id := seedOrder(t, pool, shop, StatusAccepted, time.Now(), seedItem{product, 4, 100})

	o, err := svc.Transition(ctx, shop, id, TransitionRequest{To: StatusCancelled, Reason: "buyer asked"})
	if err != nil {
		t.Fatal(err)
	}
	if o.CancelReason == nil || *o.CancelReason != "buyer asked" {
		t.Fatalf("cancel reason = %v", o.CancelReason)
	}
	if _, reserved := productStock(t, pool, product); reserved != 0 {
		t.Fatalf("reserved = %d after cancel, want 0", reserved)
	}
	if n := count(t, pool, `SELECT count(*) FROM balance_entries WHERE order_id = $1`, id); n != 0 {
		t.Fatalf("a cancelled order wrote %d balance entries", n)
	}
}

func TestPgFailedAcceptChangesNothing(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	svc := newPgService(t, repo)
	shop := seedShop(t, pool)
	plenty, scarce := seedProduct(t, pool, shop, 10, 0), seedProduct(t, pool, shop, 1, 0)
	id := seedOrder(t, pool, shop, StatusNew, time.Now(), seedItem{plenty, 2, 100}, seedItem{scarce, 3, 100})

	_, err := svc.Transition(ctx, shop, id, TransitionRequest{To: StatusAccepted})
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("err = %v, want ErrInsufficientStock", err)
	}
	if _, reserved := productStock(t, pool, plenty); reserved != 0 {
		t.Fatalf("the first product kept %d reserved units after a failed accept", reserved)
	}
	o, _ := repo.Get(ctx, shop, id)
	if o.Status != StatusNew || len(o.History) != 0 {
		t.Fatalf("order after failed accept = %s with %d history rows", o.Status, len(o.History))
	}
}

func TestPgFailedDeliveryRollsBackStockAndHistory(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	svc := newPgService(t, repo)
	shop := seedShop(t, pool)
	product := seedProduct(t, pool, shop, 10, 3)
	id := seedOrder(t, pool, shop, StatusHandedToCourier, time.Now(), seedItem{product, 3, 100})
	if _, err := pool.Exec(ctx, `INSERT INTO balance_entries (shop_id, order_id, type, amount) VALUES ($1, $2, 'sale', 300)`, shop, id); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Transition(ctx, shop, id, TransitionRequest{To: StatusDelivered}); err == nil {
		t.Fatal("delivery succeeded although the sale entry already exists")
	}
	if stock, reserved := productStock(t, pool, product); stock != 10 || reserved != 3 {
		t.Fatalf("stock/reserved = %d/%d after a failed delivery, want 10/3", stock, reserved)
	}
	o, _ := repo.Get(ctx, shop, id)
	if o.Status != StatusHandedToCourier || len(o.History) != 0 {
		t.Fatalf("order = %s with %d history rows, want it untouched", o.Status, len(o.History))
	}
	if n := count(t, pool, `SELECT count(*) FROM balance_entries WHERE order_id = $1`, id); n != 1 {
		t.Fatalf("balance entries = %d, want only the pre-existing one", n)
	}
}

func TestPgReserveCannotOversell(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	shop := seedShop(t, pool)
	product := seedProduct(t, pool, shop, 10, 0)

	var wg sync.WaitGroup
	results := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- repo.InTx(ctx, func(tx TxRepository) error { return tx.Reserve(ctx, shop, product, 2) })
		}()
	}
	wg.Wait()
	close(results)

	ok, refused := 0, 0
	for err := range results {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrInsufficientStock):
			refused++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 5 || refused != 7 {
		t.Fatalf("%d reserves succeeded and %d were refused, want 5 and 7 for stock 10 and 2 units each", ok, refused)
	}
	if stock, reserved := productStock(t, pool, product); stock != 10 || reserved != 10 {
		t.Fatalf("stock/reserved = %d/%d, want 10/10", stock, reserved)
	}
}

func TestPgSameOrderAcceptedTwiceAtOnce(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	svc := newPgService(t, repo)
	shop := seedShop(t, pool)
	product := seedProduct(t, pool, shop, 10, 0)
	id := seedOrder(t, pool, shop, StatusNew, time.Now(), seedItem{product, 4, 100})

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.Transition(ctx, shop, id, TransitionRequest{To: StatusAccepted})
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	ok, invalid := 0, 0
	for err := range results {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrInvalidTransition):
			invalid++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 1 || invalid != 1 {
		t.Fatalf("%d accepted and %d refused, want exactly one of each", ok, invalid)
	}
	if _, reserved := productStock(t, pool, product); reserved != 4 {
		t.Fatalf("reserved = %d, want 4 (reserved once, not twice)", reserved)
	}
	if n := count(t, pool, `SELECT count(*) FROM order_status_history WHERE order_id = $1`, id); n != 1 {
		t.Fatalf("history rows = %d, want 1", n)
	}
}

func TestPgDatabaseRejectsCancelWithoutReason(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	shop := seedShop(t, pool)
	id := seedOrder(t, pool, shop, StatusNew, time.Now())

	err := repo.InTx(ctx, func(tx TxRepository) error {
		return tx.SetStatus(ctx, shop, id, StatusCancelled, nil, time.Now())
	})
	if err == nil {
		t.Fatal("the database accepted a cancelled order without a reason")
	}
	o, _ := repo.Get(ctx, shop, id)
	if o.Status != StatusNew || len(o.History) != 0 {
		t.Fatalf("order = %s with %d history rows after the rejected change", o.Status, len(o.History))
	}
}

func TestPgListPagesAndFilters(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	shop := seedShop(t, pool)
	base := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	ids := make([]int64, 5)
	for i := range ids {
		st := StatusNew
		if i == 4 {
			st = StatusCancelled
		}
		ids[i] = seedOrder(t, pool, shop, st, base.Add(time.Duration(i)*time.Hour))
	}

	page, err := repo.List(ctx, shop, Filter{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 5 || len(page.Items) != 2 || page.Items[0].ID != ids[2] || page.Items[1].ID != ids[1] {
		t.Fatalf("page 2 of newest first = %+v", page)
	}
	cancelled := StatusCancelled
	filtered, err := repo.List(ctx, shop, Filter{Status: &cancelled, Page: 1, PageSize: 20})
	if err != nil || filtered.Total != 1 || filtered.Items[0].ID != ids[4] {
		t.Fatalf("filtered = %+v, %v", filtered, err)
	}
	beyond, err := repo.List(ctx, shop, Filter{Page: 9, PageSize: 2})
	if err != nil || beyond.Total != 5 || beyond.Items == nil || len(beyond.Items) != 0 {
		t.Fatalf("page past the end = %+v, %v", beyond, err)
	}
}

func TestPgSimulateCreatesOrderFromSellableProducts(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	svc := newPgService(t, repo)
	shop, otherShop := seedShop(t, pool), seedShop(t, pool)
	sellable := seedProduct(t, pool, shop, 10, 2)
	archived := seedProduct(t, pool, shop, 10, 0)
	soldOut := seedProduct(t, pool, shop, 5, 5)
	foreign := seedProduct(t, pool, otherShop, 10, 0)
	if _, err := pool.Exec(ctx, `UPDATE products SET status = 'archived' WHERE id = $1`, archived); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 20; i++ {
		o, err := svc.Simulate(ctx, shop)
		if err != nil {
			t.Fatal(err)
		}
		if o.Status != StatusNew || !strings.HasPrefix(o.Number, "#") || o.CreatedAt.Location() != time.UTC {
			t.Fatalf("order = %+v", o)
		}
		if len(o.Items) != 1 || o.Items[0].ProductID != sellable {
			t.Fatalf("items = %+v, only the active product with available stock may be used (not %d, %d or %d)", o.Items, archived, soldOut, foreign)
		}
		it := o.Items[0]
		if it.Quantity < 1 || it.Quantity > 3 || it.Quantity > 8 || it.Title != "Phone" || it.Price != 1000 || o.Total != it.Price*int64(it.Quantity) {
			t.Fatalf("item = %+v, total %d", it, o.Total)
		}
		if len(o.History) != 1 || o.History[0].FromStatus != nil || o.History[0].ToStatus != StatusNew {
			t.Fatalf("history = %+v, want one row from nothing to new", o.History)
		}

		got, err := repo.Get(ctx, shop, o.ID)
		if err != nil || got.Total != o.Total || len(got.Items) != 1 || len(got.History) != 1 {
			t.Fatalf("stored order = %+v, %v", got, err)
		}
	}
	if _, reserved := productStock(t, pool, sellable); reserved != 2 {
		t.Fatalf("simulating orders reserved stock: reserved = %d, want 2", reserved)
	}
	if n := count(t, pool, `SELECT count(*) FROM orders WHERE shop_id = $1`, otherShop); n != 0 {
		t.Fatalf("the other shop got %d orders", n)
	}
}

func TestPgSimulateWithoutSellableProducts(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	svc := newPgService(t, repo)
	shop := seedShop(t, pool)
	archived := seedProduct(t, pool, shop, 10, 0)
	if _, err := pool.Exec(ctx, `UPDATE products SET status = 'archived' WHERE id = $1`, archived); err != nil {
		t.Fatal(err)
	}
	seedProduct(t, pool, shop, 3, 3)

	if _, err := svc.Simulate(ctx, shop); !errors.Is(err, ErrNoProducts) {
		t.Fatalf("err = %v, want ErrNoProducts", err)
	}
	if n := count(t, pool, `SELECT count(*) FROM orders WHERE shop_id = $1`, shop); n != 0 {
		t.Fatalf("a refused simulation left %d orders", n)
	}
}

func TestPgCreateRejectsAnotherShopsProductAndStoresNothing(t *testing.T) {
	repo, pool := newPgRepo(t)
	shopA, shopB := seedShop(t, pool), seedShop(t, pool)
	foreign := seedProduct(t, pool, shopA, 10, 0)

	_, err := repo.Create(context.Background(), shopB, NewOrder{
		BuyerName: "Buyer", BuyerPhone: "+998901234567", DeliveryAddress: "Tashkent",
		Items: []NewItem{{ProductID: foreign, Title: "Phone", Price: 1000, Quantity: 1}},
	}, time.Now())
	if !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("err = %v, want ErrProductNotFound", err)
	}
	if n := count(t, pool, `SELECT count(*) FROM orders WHERE shop_id = $1`, shopB); n != 0 {
		t.Fatalf("%d orders stored for shop B, want the whole order rolled back", n)
	}
}

func TestPgBalanceEntriesKeepTheTimeOfTheStatusChange(t *testing.T) {
	repo, pool := newPgRepo(t)
	shop := seedShop(t, pool)
	product := seedProduct(t, pool, shop, 10, 3)
	id := seedOrder(t, pool, shop, StatusHandedToCourier, time.Now().Add(-72*time.Hour), seedItem{product, 3, 1000})
	at := time.Now().UTC().Add(-48 * time.Hour).Truncate(time.Second)

	svc := newPgService(t, repo).WithClock(func() time.Time { return at })
	if _, err := svc.Transition(context.Background(), shop, id, TransitionRequest{To: StatusDelivered}); err != nil {
		t.Fatal(err)
	}
	rows, err := pool.Query(context.Background(), `SELECT created_at FROM balance_entries WHERE order_id = $1`, id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var got time.Time
		if err := rows.Scan(&got); err != nil {
			t.Fatal(err)
		}
		if !got.Equal(at) {
			t.Errorf("entry created_at = %v, want the delivery time %v", got, at)
		}
		n++
	}
	if n != 2 {
		t.Fatalf("%d balance entries, want a sale and a commission", n)
	}
}
