package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/catalog"
	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
	"github.com/0abdurahmon/seller-panel/backend/internal/product"
	"github.com/0abdurahmon/seller-panel/backend/internal/shop"
	"github.com/0abdurahmon/seller-panel/backend/internal/stock"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	return newTestRouterWithLimit(t, 1000, 1000)
}

func newTestRouterWithLimit(t *testing.T, shopRPS float64, shopBurst int) http.Handler {
	t.Helper()
	return newTestRouterOptions(t, shopRPS, shopBurst, false)
}

func newTestRouterOptions(t *testing.T, shopRPS float64, shopBurst int, devEndpoints bool, spec ...http.Handler) http.Handler {
	t.Helper()
	tokens, err := auth.NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := auth.NewService(&fakeAuthRepository{byMail: map[string]auth.Credentials{}}, tokens, nil)
	if err != nil {
		t.Fatal(err)
	}
	stockSvc := stock.NewService(fakeStockRepository{})
	shopRepo := &fakeShopRepository{}
	return NewRouter(Deps{
		Auth:    auth.NewHandler(svc, httpx.NewLimiterPer(5, time.Hour), func(r *http.Request) string { return r.RemoteAddr }),
		Shop:    shop.NewHandler(shop.NewService(shopRepo, fakePhotoStore{})),
		Catalog: catalog.NewHandler(catalog.NewService(fakeCatalogRepository{})),
		Product: product.NewHandler(product.NewService(fakeProductRepository{}, fakePhotoStore{})),
		Tokens:  tokens,
		Stock:   stock.NewHandler(stockSvc, func(context.Context, int64, int64) (any, error) { return nil, nil }),

		ShopLimiter:  httpx.NewLimiter(shopRPS, shopBurst),
		DevEndpoints: devEndpoints,
		Spec:         firstHandler(spec),
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

func TestUnknownRouteIsJSON404(t *testing.T) {
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
	var e struct{ Code, Message string }
	if err := json.NewDecoder(res.Body).Decode(&e); err != nil || e.Code != "not_found" || e.Message == "" {
		t.Fatalf("body is not the shared error shape: %+v, %v", e, err)
	}
}

func TestWrongMethodIsJSON405(t *testing.T) {
	srv := httptest.NewServer(newTestRouter(t))
	defer srv.Close()

	for _, path := range []string{"/health", "/auth/login", "/products"} {
		method := http.MethodDelete
		req, err := http.NewRequest(method, srv.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		var e struct{ Code, Message string }
		decodeErr := json.NewDecoder(res.Body).Decode(&e)
		res.Body.Close()
		if res.StatusCode != http.StatusMethodNotAllowed || res.Header.Get("Allow") == "" || decodeErr != nil || e.Code != "method_not_allowed" {
			t.Errorf("%s %s: status %d, Allow %q, error %+v (%v)", method, path, res.StatusCode, res.Header.Get("Allow"), e, decodeErr)
		}
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
		{http.MethodGet, "/orders"},
		{http.MethodGet, "/orders/1"},
		{http.MethodPost, "/orders/1/transition"},
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
	auth.Repository // the sign-in code methods are not exercised by these tests
	byMail          map[string]auth.Credentials
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

func (r *fakeAuthRepository) ShopOwnedBy(context.Context, int64, int64) (bool, error) {
	return true, nil
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

type fakePhotoStore struct{}

func (fakePhotoStore) Save(context.Context, int64, io.Reader) (string, error) { return "", nil }

func (fakePhotoStore) Remove(context.Context, int64, []string) {}

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

func TestProtectedRoutesAreLimitedPerShop(t *testing.T) {
	tokens, err := auth.NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	router := newTestRouterWithLimit(t, 0.001, 3)

	get := func(token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/categories", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	shopA, _, _ := tokens.Issue(10, 1)
	shopB, _, _ := tokens.Issue(20, 2)

	for i := 1; i <= 3; i++ {
		if rec := get(shopA); rec.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want 200", i, rec.Code)
		}
	}
	rec := get(shopA)
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("4th request: status %d, Retry-After %q, want 429 with a header", rec.Code, rec.Header().Get("Retry-After"))
	}
	if rec := get(shopB); rec.Code != http.StatusOK {
		t.Fatalf("another shop got %d, one shop must not use up the other's budget", rec.Code)
	}

	bad := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/categories", nil)
	for i := 0; i < 10; i++ {
		bad = httptest.NewRecorder()
		router.ServeHTTP(bad, req)
	}
	if bad.Code != http.StatusUnauthorized {
		t.Fatalf("requests without a token got %d, want 401 and no limiter bucket", bad.Code)
	}
}

func TestSimulateOrderRouteExistsOnlyInDevelopment(t *testing.T) {
	tokens, err := auth.NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := tokens.Issue(10, 1)
	if err != nil {
		t.Fatal(err)
	}
	post := func(h http.Handler, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/dev/simulate-order", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	production := newTestRouterOptions(t, 1000, 1000, false)
	rec := post(production, token)
	var e struct{ Code string }
	if rec.Code != http.StatusNotFound || json.NewDecoder(rec.Body).Decode(&e) != nil || e.Code != "not_found" {
		t.Fatalf("production: status %d code %q, want the JSON 404", rec.Code, e.Code)
	}

	development := newTestRouterOptions(t, 1000, 1000, true)
	if rec := post(development, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("development without a token: status %d, want 401", rec.Code)
	}
}

func firstHandler(hs []http.Handler) http.Handler {
	if len(hs) == 0 {
		return nil
	}
	return hs[0]
}

func TestOpenAPIRouteExistsOnlyWhenConfigured(t *testing.T) {
	spec := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"openapi":"3.0.3"}`))
	})
	get := func(h http.Handler) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
		return rec
	}

	off := get(newTestRouterOptions(t, 1000, 1000, false))
	var e struct{ Code string }
	if off.Code != http.StatusNotFound || json.NewDecoder(off.Body).Decode(&e) != nil || e.Code != "not_found" {
		t.Fatalf("without a spec: status %d code %q, want the JSON 404", off.Code, e.Code)
	}

	on := get(newTestRouterOptions(t, 1000, 1000, false, spec))
	if on.Code != http.StatusOK || !strings.Contains(on.Body.String(), `"openapi"`) {
		t.Fatalf("with a spec: status %d body %s", on.Code, on.Body.String())
	}
}
