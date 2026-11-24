//go:build e2e

package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
)

var protectedRoutes = []struct{ method, path string }{
	{http.MethodGet, "/shop"},
	{http.MethodPut, "/shop"},
	{http.MethodGet, "/categories"},
	{http.MethodGet, "/products"},
	{http.MethodPost, "/products"},
	{http.MethodGet, "/products/1"},
	{http.MethodPut, "/products/1"},
	{http.MethodPost, "/products/1/archive"},
	{http.MethodPatch, "/products/1/stock"},
	{http.MethodGet, "/balance"},
	{http.MethodGet, "/balance/entries"},
	{http.MethodGet, "/orders"},
	{http.MethodGet, "/orders/1"},
	{http.MethodPost, "/orders/1/transition"},
}

func TestEveryProtectedRouteRejectsBadTokens(t *testing.T) {
	s := newServer(t)

	foreign, _, err := mustIssuer(t, "another-secret", time.Hour).Issue(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	expired, _, err := mustIssuer(t, jwtSecret, -time.Hour).Issue(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	tokens := map[string]string{
		"none":            "",
		"garbage":         "not-a-token",
		"signed by other": foreign,
		"expired":         expired,
	}
	for name, token := range tokens {
		for _, r := range protectedRoutes {
			res := s.do(r.method, r.path, token, "{}")
			if res.status != http.StatusUnauthorized || res.errorCode() != "unauthorized" {
				t.Errorf("%s %s with %s token: status %d body %s, want 401 unauthorized", r.method, r.path, name, res.status, res.body)
			}
		}
	}

	res := s.do(http.MethodGet, "/products", "", nil, "Authorization", "Basic abc")
	s.expect(res, http.StatusUnauthorized, "unauthorized")
}

func mustIssuer(t *testing.T, secret string, ttl time.Duration) *auth.TokenIssuer {
	t.Helper()
	issuer, err := auth.NewTokenIssuer(secret, ttl)
	if err != nil {
		t.Fatal(err)
	}
	return issuer
}

func TestAuthErrors(t *testing.T) {
	s := newServer(t)
	sel := s.register()

	dup := s.do(http.MethodPost, "/auth/register", "", map[string]string{
		"email": sel.email, "password": "correct-horse", "shop_name": "Again",
	})
	s.expect(dup, http.StatusConflict, "email_taken")

	for name, body := range map[string]any{
		"bad email":      map[string]string{"email": "nope", "password": "correct-horse", "shop_name": "S"},
		"short password": map[string]string{"email": uniqueEmail(), "password": "short", "shop_name": "S"},
		"no shop name":   map[string]string{"email": uniqueEmail(), "password": "correct-horse", "shop_name": " "},
	} {
		if res := s.do(http.MethodPost, "/auth/register", "", body); res.status != http.StatusBadRequest || res.errorCode() != "validation_error" {
			t.Errorf("%s: status %d body %s, want 400 validation_error", name, res.status, res.body)
		}
	}
	s.expect(s.do(http.MethodPost, "/auth/register", "", `{"email":"x@example.com","password":"correct-horse","shop_name":"S","admin":true}`), http.StatusBadRequest, "invalid_body")
	s.expect(s.do(http.MethodPost, "/auth/login", "", `not json`), http.StatusBadRequest, "invalid_body")

	wrong := s.login(sel.email, "wrong-password")
	unknown := s.login("nobody-"+sel.email, "wrong-password")
	s.expect(wrong, http.StatusUnauthorized, "invalid_credentials")
	s.expect(unknown, http.StatusUnauthorized, "invalid_credentials")
	if string(wrong.body) != string(unknown.body) {
		t.Fatalf("a wrong password and an unknown email must look the same: %s vs %s", wrong.body, unknown.body)
	}

	upper := s.login(" "+sel.email+" ", sel.password)
	s.expect(upper, http.StatusOK, "")
}

func TestProductValidationAndConflicts(t *testing.T) {
	s := newServer(t)
	sel := s.register()
	category := s.categoryID(sel.token)
	p := s.createProduct(sel.token, category, 10)
	other := s.createProduct(sel.token, category, 1)

	bad := map[string]map[string]any{
		"zero price":       {"price": 0},
		"negative price":   {"price": -5},
		"blank title":      {"title": "   "},
		"blank sku":        {"sku": " "},
		"unknown category": {"category_id": 999_999_999},
		"no category":      {"category_id": 0},
		"negative stock":   {"stock": -1},
		"huge stock":       {"stock": 1_000_001},
		"bad status":       {"status": "archived"},
	}
	for name, override := range bad {
		body := productBody(category, uniqueSKU(), 1)
		for k, v := range override {
			body[k] = v
		}
		if res := s.do(http.MethodPost, "/products", sel.token, body); res.status != http.StatusBadRequest || res.errorCode() != "validation_error" {
			t.Errorf("%s: status %d body %s, want 400 validation_error", name, res.status, res.body)
		}
	}

	dup := productBody(category, p.SKU, 1)
	s.expect(s.do(http.MethodPost, "/products", sel.token, dup), http.StatusConflict, "sku_taken")

	update := productBody(category, p.SKU, 0)
	delete(update, "stock")
	s.expect(s.do(http.MethodPut, productPath(other), sel.token, update), http.StatusConflict, "sku_taken")

	withStock := productBody(category, p.SKU, 99)
	s.expect(s.do(http.MethodPut, productPath(p), sel.token, withStock), http.StatusBadRequest, "invalid_body")

	s.expect(s.do(http.MethodGet, "/products/abc", sel.token, nil), http.StatusNotFound, "not_found")
	s.expect(s.do(http.MethodGet, "/products/0", sel.token, nil), http.StatusNotFound, "not_found")
	s.expect(s.do(http.MethodGet, "/products?status=sold", sel.token, nil), http.StatusBadRequest, "validation_error")
	s.expect(s.do(http.MethodGet, "/products?page=x", sel.token, nil), http.StatusBadRequest, "validation_error")
}

func TestStockRules(t *testing.T) {
	s := newServer(t)
	sel := s.register()
	p := s.createProduct(sel.token, s.categoryID(sel.token), 10)
	path := productPath(p) + "/stock"

	s.expect(s.do(http.MethodPatch, path, sel.token, map[string]int{"stock": 8, "expected_stock": 9}), http.StatusConflict, "stock_changed")
	s.expect(s.do(http.MethodPatch, path, sel.token, `{"stock":8}`), http.StatusBadRequest, "validation_error")
	s.expect(s.do(http.MethodPatch, path, sel.token, map[string]int{"stock": -1, "expected_stock": 10}), http.StatusBadRequest, "validation_error")
	s.expect(s.do(http.MethodPatch, path, sel.token, map[string]int{"stock": 1_000_001, "expected_stock": 10}), http.StatusBadRequest, "validation_error")
	s.expect(s.do(http.MethodPatch, "/products/999999999/stock", sel.token, map[string]int{"stock": 1, "expected_stock": 1}), http.StatusNotFound, "not_found")

	if _, err := s.pool.Exec(t.Context(), `UPDATE products SET reserved = 4 WHERE id = $1`, p.ID); err != nil {
		t.Fatal(err)
	}
	s.expect(s.do(http.MethodPatch, path, sel.token, map[string]int{"stock": 3, "expected_stock": 10}), http.StatusConflict, "stock_below_reserved")

	ok := s.do(http.MethodPatch, path, sel.token, map[string]int{"stock": 4, "expected_stock": 10})
	s.expect(ok, http.StatusOK, "")
	var level struct {
		Stock, Reserved, Available int
		StockLabel                 string `json:"stock_label"`
	}
	ok.decode(&level)
	if level.Stock != 4 || level.Reserved != 4 || level.Available != 0 || level.StockLabel != "out_of_stock" {
		t.Fatalf("level = %+v, want stock 4, reserved 4, available 0, out_of_stock", level)
	}
}

func TestListingPagesAndLiteralSearch(t *testing.T) {
	s := newServer(t)
	sel := s.register()
	category := s.categoryID(sel.token)
	for i := 0; i < 25; i++ {
		s.createProduct(sel.token, category, 1)
	}
	odd := productBody(category, "50%_OFF-"+uniqueSKU(), 1)
	odd["title"] = "Sale 50% off"
	s.expect(s.do(http.MethodPost, "/products", sel.token, odd), http.StatusCreated, "")

	var page productPage
	s.do(http.MethodGet, "/products?page=3&page_size=10", sel.token, nil).decode(&page)
	if page.Total != 26 || len(page.Items) != 6 || page.Page != 3 || page.PageSize != 10 {
		t.Fatalf("page 3 of 10 = total %d, %d items, page %d, size %d", page.Total, len(page.Items), page.Page, page.PageSize)
	}
	s.do(http.MethodGet, "/products?page_size=500", sel.token, nil).decode(&page)
	if page.PageSize != 100 || len(page.Items) != 26 {
		t.Fatalf("page_size 500 = size %d, %d items, want it capped at 100", page.PageSize, len(page.Items))
	}
	s.do(http.MethodGet, "/products?page=99&page_size=10", sel.token, nil).decode(&page)
	if page.Total != 26 || len(page.Items) != 0 || page.Items == nil {
		t.Fatalf("page past the end = %+v", page)
	}
	s.do(http.MethodGet, "/products?q=%25", sel.token, nil).decode(&page)
	if page.Total != 1 {
		t.Fatalf("a percent sign must be searched literally, total = %d, want 1", page.Total)
	}
}
