package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/catalog"
	"github.com/0abdurahmon/seller-panel/backend/internal/product"
	"github.com/0abdurahmon/seller-panel/backend/internal/shop"
	"github.com/0abdurahmon/seller-panel/backend/internal/stock"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	tokens, err := auth.NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := auth.NewService(&fakeAuthRepository{byMail: map[string]auth.Credentials{}}, tokens)
	if err != nil {
		t.Fatal(err)
	}
	stockSvc, err := stock.NewService(fakeStockRepository{}, 5)
	if err != nil {
		t.Fatal(err)
	}
	shopRepo := &fakeShopRepository{}
	return NewRouter(Deps{
		Auth:    auth.NewHandler(svc),
		Shop:    shop.NewHandler(shop.NewService(shopRepo)),
		Catalog: catalog.NewHandler(catalog.NewService(fakeCatalogRepository{})),
		Product: product.NewHandler(product.NewService(fakeProductRepository{}), func(int) string { return "in_stock" }),
		Tokens:  tokens,
		Stock:   stock.NewHandler(stockSvc),
	})
}

func TestStockRouteRequiresAuth(t *testing.T) {
	srv := httptest.NewServer(newTestRouter(t))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/products/1/stock", strings.NewReader(`{"stock": 5, "expected_stock": 0}`))
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusUnauthorized)
	}
}

func TestHealth(t *testing.T) {
	srv := httptest.NewServer(newTestRouter(t))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status field = %q, want ok", body["status"])
	}
}

func TestUnknownRouteIs404(t *testing.T) {
	srv := httptest.NewServer(newTestRouter(t))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/nope")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusNotFound)
	}
}

func TestProtectedRoutesRequireToken(t *testing.T) {
	srv := httptest.NewServer(newTestRouter(t))
	defer srv.Close()

	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/shop"},
		{http.MethodPut, "/shop"},
		{http.MethodGet, "/categories"},
		{http.MethodGet, "/products"},
		{http.MethodPost, "/products"},
		{http.MethodGet, "/products/1"},
		{http.MethodPut, "/products/1"},
		{http.MethodPost, "/products/1/archive"},
	} {
		req, err := http.NewRequest(c.method, srv.URL+c.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want %d", c.method, c.path, res.StatusCode, http.StatusUnauthorized)
		}
	}
}

func TestCategoriesRequireToken(t *testing.T) {
	srv := httptest.NewServer(newTestRouter(t))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/categories")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusUnauthorized)
	}
}

type fakeCatalogRepository struct{}

func (fakeCatalogRepository) List(context.Context) ([]catalog.Category, error) {
	return []catalog.Category{{ID: 1, Name: "Electronics"}}, nil
}

type fakeAuthRepository struct {
	byMail map[string]auth.Credentials
}

func (r *fakeAuthRepository) CreateSellerWithShop(_ context.Context, email, passwordHash, _ string) (int64, int64, error) {
	if _, taken := r.byMail[email]; taken {
		return 0, 0, auth.ErrEmailTaken
	}
	id := int64(len(r.byMail) + 1)
	r.byMail[email] = auth.Credentials{SellerID: id, ShopID: id, PasswordHash: passwordHash}
	return id, id, nil
}

func (r *fakeAuthRepository) FindCredentials(_ context.Context, email string) (auth.Credentials, error) {
	c, ok := r.byMail[email]
	if !ok {
		return auth.Credentials{}, auth.ErrNotFound
	}
	return c, nil
}

type fakeShopRepository struct{}

func (fakeShopRepository) Get(_ context.Context, shopID int64) (shop.Shop, error) {
	return shop.Shop{ID: shopID, Name: "Test shop"}, nil
}

func (fakeShopRepository) Update(_ context.Context, shopID int64, p shop.Profile) (shop.Shop, error) {
	return shop.Shop{ID: shopID, Name: p.Name}, nil
}

type fakeStockRepository struct{}

func (fakeStockRepository) SetStock(_ context.Context, _, _ int64, qty, _ int) (stock.Row, error) {
	return stock.Row{Stock: qty}, nil
}

type fakeProductRepository struct{}

func (fakeProductRepository) Create(context.Context, int64, product.Product) (product.Product, error) {
	return product.Product{}, nil
}

func (fakeProductRepository) Get(context.Context, int64, int64) (product.Product, error) {
	return product.Product{}, product.ErrNotFound
}

func (fakeProductRepository) Update(context.Context, int64, product.Product) (product.Product, error) {
	return product.Product{}, product.ErrNotFound
}

func (fakeProductRepository) List(context.Context, int64, product.ListFilter) ([]product.Product, int, error) {
	return nil, 0, nil
}
