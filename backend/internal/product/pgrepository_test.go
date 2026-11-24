package product

import (
	"context"
	"errors"
	"fmt"
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
	if _, err := repo.Update(ctx, shopA, other); !errors.Is(err, ErrSKUTaken) {
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
	updated, err := repo.Update(ctx, shopA, p)
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

	if _, err := repo.Update(ctx, shopB, p); !errors.Is(err, ErrNotFound) {
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
