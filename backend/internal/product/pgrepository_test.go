package product

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0abdurahmon/seller-panel/backend/internal/database/dbtest"
)

func newPgRepo(t *testing.T) (*PgRepository, *pgxpool.Pool) {
	t.Helper()
	pool := dbtest.NewPool(t, "product")
	return NewPgRepository(pool), pool
}

func seedShop(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	ctx := context.Background()
	var sellerID, shopID int64
	email := fmt.Sprintf("product-%d@example.com", time.Now().UnixNano())
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

func seedCategory(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	var id int64
	name := fmt.Sprintf("category-%d", time.Now().UnixNano())
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO categories (name) VALUES ($1) RETURNING id`, name).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func newProduct(categoryID int64, sku string) Product {
	return Product{CategoryID: categoryID, Title: "Phone X", Description: "Good", Price: 1_000_000, SKU: sku, Stock: 10, Status: StatusActive}
}

func replace(p Product) func(Product) (Product, error) {
	return func(Product) (Product, error) { return p, nil }
}

func TestPgCreateAndGetAreScopedByShop(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	shopA, shopB, cat := seedShop(t, pool), seedShop(t, pool), seedCategory(t, pool)

	created, err := repo.Create(ctx, shopA, newProduct(cat, "PX-1"))
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 || created.ShopID != shopA || created.Stock != 10 || created.Reserved != 0 || created.Status != StatusActive {
		t.Fatalf("created = %+v", created)
	}

	got, err := repo.Get(ctx, shopA, created.ID)
	if err != nil || got.Title != "Phone X" {
		t.Fatalf("get = %+v, %v", got, err)
	}
	for name, ts := range map[string]time.Time{
		"created.CreatedAt": created.CreatedAt, "created.UpdatedAt": created.UpdatedAt,
		"got.CreatedAt": got.CreatedAt, "got.UpdatedAt": got.UpdatedAt,
	} {
		if ts.Location() != time.UTC {
			t.Errorf("%s is in %v, want UTC so the JSON never carries a local offset", name, ts.Location())
		}
	}
	if _, err := repo.Get(ctx, shopB, created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other shop get err = %v, want ErrNotFound", err)
	}
	if _, err := repo.Get(ctx, shopA, created.ID+1000); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing id err = %v, want ErrNotFound", err)
	}
}

func TestPgSKUIsUniquePerShop(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	shopA, shopB, cat := seedShop(t, pool), seedShop(t, pool), seedCategory(t, pool)

	first, err := repo.Create(ctx, shopA, newProduct(cat, "DUP"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Create(ctx, shopA, newProduct(cat, "DUP")); !errors.Is(err, ErrSKUTaken) {
		t.Fatalf("duplicate create err = %v, want ErrSKUTaken", err)
	}
	if _, err := repo.Create(ctx, shopB, newProduct(cat, "DUP")); err != nil {
		t.Fatalf("same sku in another shop: %v", err)
	}

	other, err := repo.Create(ctx, shopA, newProduct(cat, "OTHER"))
	if err != nil {
		t.Fatal(err)
	}
	other.SKU = first.SKU
	if _, err := repo.Update(ctx, shopA, other.ID, replace(other)); !errors.Is(err, ErrSKUTaken) {
		t.Fatalf("update to taken sku err = %v, want ErrSKUTaken", err)
	}
}

func TestPgUnknownCategoryIsValidationError(t *testing.T) {
	repo, pool := newPgRepo(t)
	shop := seedShop(t, pool)

	_, err := repo.Create(context.Background(), shop, newProduct(999_999_999, "NOCAT"))
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestPgUpdateKeepsStockAndIsScoped(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	shopA, shopB, cat := seedShop(t, pool), seedShop(t, pool), seedCategory(t, pool)

	p, err := repo.Create(ctx, shopA, newProduct(cat, "UPD"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE products SET reserved = 4 WHERE id = $1`, p.ID); err != nil {
		t.Fatal(err)
	}

	p.Title, p.Price, p.Status = "Renamed", 2_000_000, StatusArchived
	p.Stock, p.Reserved = 999, 999
	updated, err := repo.Update(ctx, shopA, p.ID, replace(p))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Renamed" || updated.Price != 2_000_000 || updated.Status != StatusArchived {
		t.Fatalf("updated = %+v", updated)
	}
	if updated.Stock != 10 || updated.Reserved != 4 {
		t.Fatalf("stock = %d reserved = %d, update must not touch them", updated.Stock, updated.Reserved)
	}
	if !updated.UpdatedAt.After(p.UpdatedAt) {
		t.Fatalf("updated_at did not move: %v then %v", p.UpdatedAt, updated.UpdatedAt)
	}

	if _, err := repo.Update(ctx, shopB, p.ID, replace(p)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other shop update err = %v, want ErrNotFound", err)
	}
}

func TestPgListFiltersSearchAndPagination(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	shopA, shopB := seedShop(t, pool), seedShop(t, pool)
	catPhones, catHome := seedCategory(t, pool), seedCategory(t, pool)

	create := func(shop, cat int64, title, sku string, status Status) {
		t.Helper()
		p := newProduct(cat, sku)
		p.Title, p.Status = title, status
		if _, err := repo.Create(ctx, shop, p); err != nil {
			t.Fatal(err)
		}
	}
	create(shopA, catPhones, "Phone Alpha", "PA-1", StatusActive)
	create(shopA, catPhones, "Phone Beta", "PB-2", StatusDraft)
	create(shopA, catHome, "Kettle 100%", "K_3", StatusActive)
	create(shopA, catHome, "Lamp", "LAMP-4", StatusArchived)
	create(shopB, catPhones, "Phone Alpha", "PA-1", StatusActive)

	list := func(f ListFilter) ([]Product, int) {
		t.Helper()
		if f.Page == 0 {
			f.Page = 1
		}
		if f.PageSize == 0 {
			f.PageSize = 20
		}
		items, total, err := repo.List(ctx, shopA, f)
		if err != nil {
			t.Fatal(err)
		}
		return items, total
	}

	if items, total := list(ListFilter{}); total != 4 || len(items) != 4 || items[0].SKU != "LAMP-4" {
		t.Fatalf("all: total %d, %d items, first %q, want 4 newest first", total, len(items), items[0].SKU)
	}
	if _, total := list(ListFilter{Status: StatusActive}); total != 2 {
		t.Fatalf("status active total = %d, want 2", total)
	}
	if _, total := list(ListFilter{CategoryID: catPhones}); total != 2 {
		t.Fatalf("category total = %d, want 2", total)
	}
	if items, total := list(ListFilter{Query: "alpha"}); total != 1 || items[0].SKU != "PA-1" {
		t.Fatalf("title search: total %d items %v", total, items)
	}
	if _, total := list(ListFilter{Query: "lamp-4"}); total != 1 {
		t.Fatalf("sku search total = %d, want 1", total)
	}
	if _, total := list(ListFilter{Query: "%"}); total != 1 {
		t.Fatalf("percent is literal: total = %d, want 1 (Kettle 100%%)", total)
	}
	if _, total := list(ListFilter{Query: "_"}); total != 1 {
		t.Fatalf("underscore is literal: total = %d, want 1 (K_3)", total)
	}
	if _, total := list(ListFilter{Status: StatusDraft, CategoryID: catHome}); total != 0 {
		t.Fatalf("combined filter total = %d, want 0", total)
	}

	page2, total := list(ListFilter{Page: 2, PageSize: 3})
	if total != 4 || len(page2) != 1 {
		t.Fatalf("page 2: total %d, %d items, want 4 and 1", total, len(page2))
	}
	beyond, total := list(ListFilter{Page: 9, PageSize: 3})
	if total != 4 || len(beyond) != 0 {
		t.Fatalf("page beyond the end: total %d, %d items", total, len(beyond))
	}
}

func TestPgConstraintsRejectBadRows(t *testing.T) {
	_, pool := newPgRepo(t)
	ctx := context.Background()
	shop, cat := seedShop(t, pool), seedCategory(t, pool)

	cases := map[string]string{
		"zero price":           `price = 0`,
		"negative stock":       `stock = -1`,
		"negative reserved":    `reserved = -1`,
		"reserved above stock": `stock = 2, reserved = 3`,
		"blank title":          `title = '  '`,
		"blank sku":            `sku = ''`,
	}
	i := 0
	for name, override := range cases {
		i++
		sku := fmt.Sprintf("BAD-%d-%d", i, time.Now().UnixNano())
		if _, err := pool.Exec(ctx,
			`INSERT INTO products (shop_id, category_id, title, price, sku) VALUES ($1, $2, 'T', 1, $3)`,
			shop, cat, sku); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx,
			`UPDATE products SET `+override+` WHERE sku = $1`, sku); err == nil {
			t.Errorf("%s: database accepted the row", name)
		}
	}
}

func TestPgPhotos(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	shop, other, cat := seedShop(t, pool), seedShop(t, pool), seedCategory(t, pool)
	u := func(n int) string { return fmt.Sprintf("/api/uploads/%d/%032d.webp", shop, n) }

	p := newProduct(cat, "PH-1")
	p.Photos = []string{u(3), u(1), u(2)}
	created, err := repo.Create(ctx, shop, p)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(created.Photos) != fmt.Sprint(p.Photos) {
		t.Fatalf("created photos = %v, want %v", created.Photos, p.Photos)
	}
	got, err := repo.Get(ctx, shop, created.ID)
	if err != nil || fmt.Sprint(got.Photos) != fmt.Sprint(p.Photos) {
		t.Fatalf("get photos = %v, err = %v: the order of the cover and the rest must be kept", got.Photos, err)
	}

	plain, err := repo.Create(ctx, shop, newProduct(cat, "PH-2"))
	if err != nil || plain.Photos == nil || len(plain.Photos) != 0 {
		t.Fatalf("a product without photos has %#v, want an empty non-nil list (err %v)", plain.Photos, err)
	}

	items, total, err := repo.List(ctx, shop, ListFilter{Page: 1, PageSize: 10})
	if err != nil || total != 2 {
		t.Fatalf("list total = %d, err = %v", total, err)
	}
	byID := map[int64]Product{}
	for _, it := range items {
		byID[it.ID] = it
	}
	if len(byID[created.ID].Photos) != 3 || byID[plain.ID].Photos == nil || len(byID[plain.ID].Photos) != 0 {
		t.Fatalf("list photos wrong: %+v", byID)
	}

	created.Photos = []string{u(2), u(5)}
	updated, err := repo.Update(ctx, shop, created.ID, replace(created))
	if err != nil || fmt.Sprint(updated.Photos) != fmt.Sprint(created.Photos) {
		t.Fatalf("update photos = %v, err = %v", updated.Photos, err)
	}
	if got, _ = repo.Get(ctx, shop, created.ID); fmt.Sprint(got.Photos) != fmt.Sprint(created.Photos) {
		t.Fatalf("after update = %v", got.Photos)
	}

	created.Photos = nil
	if updated, err = repo.Update(ctx, shop, created.ID, replace(created)); err != nil || len(updated.Photos) != 0 {
		t.Fatalf("clear = %v, err = %v", updated.Photos, err)
	}

	foreign := created
	foreign.Photos = []string{u(1)}
	if _, err := repo.Update(ctx, other, foreign.ID, replace(foreign)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another shop's update err = %v, want ErrNotFound", err)
	}
	if got, _ = repo.Get(ctx, shop, created.ID); len(got.Photos) != 0 {
		t.Fatalf("another shop's update changed photos: %v", got.Photos)
	}
}

func TestPgPhotoConstraintsAndRollback(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	shop, cat := seedShop(t, pool), seedCategory(t, pool)

	six := newProduct(cat, "PH-6")
	for i := 0; i < 6; i++ {
		six.Photos = append(six.Photos, fmt.Sprintf("/u/%d", i))
	}
	if _, err := repo.Create(ctx, shop, six); err == nil {
		t.Fatal("six photos must violate the position check")
	}
	dup := newProduct(cat, "PH-D")
	dup.Photos = []string{"/u/a", "/u/a"}
	if _, err := repo.Create(ctx, shop, dup); err == nil {
		t.Fatal("the same url twice must violate the unique key")
	}
	blank := newProduct(cat, "PH-B")
	blank.Photos = []string{" "}
	if _, err := repo.Create(ctx, shop, blank); err == nil {
		t.Fatal("a blank url must violate the check")
	}

	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM products WHERE shop_id = $1`, shop).Scan(&n); err != nil || n != 0 {
		t.Fatalf("%d products left after failed creates (err %v), the transaction must roll back", n, err)
	}
}

func TestPgPhotosAreDeletedWithTheProduct(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	shop, cat := seedShop(t, pool), seedCategory(t, pool)
	p := newProduct(cat, "PH-C")
	p.Photos = []string{"/u/1"}
	created, err := repo.Create(ctx, shop, p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, created.ID); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM product_photos WHERE product_id = $1`, created.ID).Scan(&n); err != nil || n != 0 {
		t.Fatalf("%d photo rows left, want 0 (err %v)", n, err)
	}
}

func TestPgConcurrentUpdatesSeeEachOther(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	shop, cat := seedShop(t, pool), seedCategory(t, pool)
	p, err := repo.Create(ctx, shop, newProduct(cat, "RACE"))
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := range maxPhotos {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.Update(ctx, shop, p.ID, func(current Product) (Product, error) {
				time.Sleep(20 * time.Millisecond)
				current.Photos = append(current.Photos, fmt.Sprintf("/api/uploads/%d/%032d.webp", shop, i))
				return current, nil
			})
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	got, err := repo.Get(ctx, shop, p.ID)
	if err != nil || len(got.Photos) != maxPhotos {
		t.Fatalf("photos = %v, err = %v, want every concurrent update to keep the others", got.Photos, err)
	}
}

func TestPgFailedChangeStoresNothing(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	shop, cat := seedShop(t, pool), seedCategory(t, pool)
	p, err := repo.Create(ctx, shop, newProduct(cat, "KEEP"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Update(ctx, shop, p.ID, func(Product) (Product, error) { return Product{}, ErrArchived }); !errors.Is(err, ErrArchived) {
		t.Fatalf("err = %v, want ErrArchived", err)
	}
	if got, err := repo.Get(ctx, shop, p.ID); err != nil || got.Title != p.Title || !got.UpdatedAt.Equal(p.UpdatedAt) {
		t.Fatalf("product = %+v, err = %v", got, err)
	}
}

func TestPgCategoryFilterIncludesSubcategories(t *testing.T) {
	repo, pool := newPgRepo(t)
	ctx := context.Background()
	shop := seedShop(t, pool)
	parent, other := seedCategory(t, pool), seedCategory(t, pool)
	child := seedSubcategory(t, pool, parent)
	grandchild := seedSubcategory(t, pool, child)
	for sku, category := range map[string]int64{"P": parent, "C": child, "G": grandchild, "O": other} {
		if _, err := repo.Create(ctx, shop, newProduct(category, sku)); err != nil {
			t.Fatal(err)
		}
	}

	for name, tc := range map[string]struct {
		category int64
		want     int
	}{
		"parent":     {parent, 3},
		"child":      {child, 2},
		"grandchild": {grandchild, 1},
		"unknown":    {999_999_999, 0},
	} {
		items, total, err := repo.List(ctx, shop, ListFilter{CategoryID: tc.category, Page: 1, PageSize: 10})
		if err != nil || total != tc.want || len(items) != tc.want {
			t.Errorf("%s: total = %d, items = %d, err = %v, want %d", name, total, len(items), err, tc.want)
		}
	}
}

func seedSubcategory(t *testing.T, pool *pgxpool.Pool, parent int64) int64 {
	t.Helper()
	var id int64
	name := fmt.Sprintf("subcategory-%d", time.Now().UnixNano())
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO categories (name, parent_id) VALUES ($1, $2) RETURNING id`, name, parent).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}
