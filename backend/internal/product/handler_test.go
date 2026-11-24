package product

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
)

func testLabel(available int) string {
	switch {
	case available == 0:
		return "out_of_stock"
	case available <= 5:
		return "low_stock"
	}
	return "in_stock"
}

type api struct {
	t      *testing.T
	mux    *http.ServeMux
	tokens *auth.TokenIssuer
}

func newAPI(t *testing.T) *api {
	t.Helper()
	tokens, err := auth.NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(NewService(newMemRepository()), testLabel)
	protect := auth.RequireAuth(tokens)
	mux := http.NewServeMux()
	mux.Handle("GET /products", protect(http.HandlerFunc(h.List)))
	mux.Handle("POST /products", protect(http.HandlerFunc(h.Create)))
	mux.Handle("GET /products/{id}", protect(http.HandlerFunc(h.Get)))
	mux.Handle("PUT /products/{id}", protect(http.HandlerFunc(h.Update)))
	mux.Handle("POST /products/{id}/archive", protect(http.HandlerFunc(h.Archive)))
	return &api{t: t, mux: mux, tokens: tokens}
}

func (a *api) token(shopID int64) string {
	a.t.Helper()
	tok, _, err := a.tokens.Issue(shopID*10, shopID)
	if err != nil {
		a.t.Fatal(err)
	}
	return tok
}

func (a *api) do(method, path, token, body string) *httptest.ResponseRecorder {
	a.t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("body %q: %v", rec.Body.String(), err)
	}
	return v
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	return decode[struct {
		Code string `json:"code"`
	}](t, rec).Code
}

const updateBody = `{"category_id":1,"title":"Phone Z","description":"Good","price":1000000,"sku":"PX-1","status":"active"}`

const createBody = `{"category_id":1,"title":"Phone X","description":"Good","price":1000000,"sku":"PX-1","stock":10,"status":"active"}`

type productJSON struct {
	Product
	Available  int    `json:"available"`
	StockLabel string `json:"stock_label"`
}

func (a *api) create(token, body string) Product {
	a.t.Helper()
	rec := a.do(http.MethodPost, "/products", token, body)
	if rec.Code != http.StatusCreated {
		a.t.Fatalf("create status = %d, body %s", rec.Code, rec.Body.String())
	}
	return decode[Product](a.t, rec)
}

func TestHandlerRequiresToken(t *testing.T) {
	a := newAPI(t)
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/products"},
		{http.MethodPost, "/products"},
		{http.MethodGet, "/products/1"},
		{http.MethodPut, "/products/1"},
		{http.MethodPost, "/products/1/archive"},
	} {
		if rec := a.do(c.method, c.path, "", `{}`); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want 401", c.method, c.path, rec.Code)
		}
	}
}

func TestHandlerCreateTakesShopFromToken(t *testing.T) {
	a := newAPI(t)
	p := a.create(a.token(7), createBody)
	if p.ShopID != 7 || p.ID == 0 || p.Stock != 10 || p.Reserved != 0 || p.Status != StatusActive {
		t.Fatalf("created = %+v", p)
	}
	if rec := a.do(http.MethodPost, "/products", a.token(7), `{"shop_id":99,"title":"x"}`); rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_body" {
		t.Fatalf("shop_id in body: status %d code %q, want 400 invalid_body", rec.Code, errorCode(t, rec))
	}
}

func TestHandlerStatusCodes(t *testing.T) {
	a := newAPI(t)
	tok := a.token(1)
	a.create(tok, createBody)

	cases := []struct {
		name         string
		method, path string
		body         string
		status       int
		code         string
	}{
		{"bad json", http.MethodPost, "/products", `{`, 400, "invalid_body"},
		{"invalid price", http.MethodPost, "/products", `{"category_id":1,"title":"T","price":0,"sku":"S"}`, 400, "validation_error"},
		{"duplicate sku", http.MethodPost, "/products", createBody, 409, "sku_taken"},
		{"get missing", http.MethodGet, "/products/999", "", 404, "not_found"},
		{"get non numeric id", http.MethodGet, "/products/abc", "", 404, "not_found"},
		{"update missing", http.MethodPut, "/products/999", updateBody, 404, "not_found"},
		{"update with stock field", http.MethodPut, "/products/1", createBody, 400, "invalid_body"},
		{"archive missing", http.MethodPost, "/products/999/archive", "", 404, "not_found"},
		{"list bad page", http.MethodGet, "/products?page=x", "", 400, "validation_error"},
		{"list bad status", http.MethodGet, "/products?status=sold", "", 400, "validation_error"},
		{"list bad category", http.MethodGet, "/products?category=x", "", 400, "validation_error"},
	}
	for _, c := range cases {
		rec := a.do(c.method, c.path, tok, c.body)
		if rec.Code != c.status || errorCode(t, rec) != c.code {
			t.Errorf("%s: status %d code %q, want %d %q", c.name, rec.Code, errorCode(t, rec), c.status, c.code)
		}
	}
}

func TestHandlerGetUpdateArchiveFlow(t *testing.T) {
	a := newAPI(t)
	tok := a.token(1)
	p := a.create(tok, createBody)
	path := "/products/" + strconv.FormatInt(p.ID, 10)

	if rec := a.do(http.MethodGet, path, tok, ""); rec.Code != http.StatusOK || decode[Product](t, rec).Title != "Phone X" {
		t.Fatalf("get: %d %s", rec.Code, rec.Body.String())
	}

	rec := a.do(http.MethodPut, path, tok, `{"category_id":1,"title":"Phone Y","description":"New","price":2000000,"sku":"PX-1","status":"draft"}`)
	if got := decode[Product](t, rec); rec.Code != http.StatusOK || got.Title != "Phone Y" || got.Price != 2_000_000 || got.Stock != 10 {
		t.Fatalf("update: %d %+v", rec.Code, got)
	}

	for i := 0; i < 2; i++ {
		rec = a.do(http.MethodPost, path+"/archive", tok, "")
		if rec.Code != http.StatusOK || decode[Product](t, rec).Status != StatusArchived {
			t.Fatalf("archive #%d: %d %s", i+1, rec.Code, rec.Body.String())
		}
	}

	rec = a.do(http.MethodPut, path, tok, updateBody)
	if rec.Code != http.StatusConflict || errorCode(t, rec) != "product_archived" {
		t.Fatalf("update archived: %d %q, want 409 product_archived", rec.Code, errorCode(t, rec))
	}
}

func TestHandlerShopsAreIsolated(t *testing.T) {
	a := newAPI(t)
	tokA, tokB := a.token(1), a.token(2)
	p := a.create(tokA, createBody)
	path := "/products/" + strconv.FormatInt(p.ID, 10)

	for _, c := range []struct{ method, path, body string }{
		{http.MethodGet, path, ""},
		{http.MethodPut, path, updateBody},
		{http.MethodPost, path + "/archive", ""},
	} {
		if rec := a.do(c.method, c.path, tokB, c.body); rec.Code != http.StatusNotFound {
			t.Errorf("shop B %s %s status = %d, want 404", c.method, c.path, rec.Code)
		}
	}
	if rec := a.do(http.MethodGet, path, tokA, ""); decode[Product](t, rec).Status != StatusActive {
		t.Fatal("shop B's calls changed shop A's product")
	}

	page := decode[Page](t, a.do(http.MethodGet, "/products", tokB, ""))
	if page.Total != 0 || page.Items == nil || len(page.Items) != 0 {
		t.Fatalf("shop B list = %+v, want empty page with items []", page)
	}
	if rec := a.do(http.MethodPost, "/products", tokB, createBody); rec.Code != http.StatusCreated {
		t.Fatalf("same sku in another shop: status %d", rec.Code)
	}
}

func TestHandlerListFiltersAndPaging(t *testing.T) {
	a := newAPI(t)
	tok := a.token(1)
	a.create(tok, `{"category_id":1,"title":"Phone Alpha","price":100,"sku":"A","status":"active"}`)
	a.create(tok, `{"category_id":2,"title":"Phone Beta","price":100,"sku":"B","status":"draft"}`)
	a.create(tok, `{"category_id":2,"title":"Lamp","price":100,"sku":"C","status":"active"}`)

	list := func(query string) Page {
		t.Helper()
		rec := a.do(http.MethodGet, "/products"+query, tok, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("list %q status = %d", query, rec.Code)
		}
		return decode[Page](t, rec)
	}
	if p := list(""); p.Total != 3 || p.Page != 1 || p.PageSize != 20 || p.Items[0].SKU != "C" {
		t.Fatalf("default list = %+v", p)
	}
	if p := list("?q=phone"); p.Total != 2 {
		t.Fatalf("q total = %d, want 2", p.Total)
	}
	if p := list("?category=2&status=active"); p.Total != 1 || p.Items[0].SKU != "C" {
		t.Fatalf("category+status = %+v", p)
	}
	if p := list("?page=2&page_size=2"); p.Total != 3 || len(p.Items) != 1 || p.Page != 2 || p.PageSize != 2 {
		t.Fatalf("paging = %+v", p)
	}
	if p := list("?page_size=500"); p.PageSize != 100 {
		t.Fatalf("page_size cap = %d, want 100", p.PageSize)
	}
}

func TestHandlerReturnsAvailableAndStockLabel(t *testing.T) {
	a := newAPI(t)
	tok := a.token(1)

	cases := []struct {
		stock     int
		available int
		label     string
	}{{0, 0, "out_of_stock"}, {3, 3, "low_stock"}, {40, 40, "in_stock"}}
	for i, c := range cases {
		body := `{"category_id":1,"title":"T","price":1,"sku":"S` + strconv.Itoa(i) + `","stock":` + strconv.Itoa(c.stock) + `}`
		rec := a.do(http.MethodPost, "/products", tok, body)
		got := decode[productJSON](t, rec)
		if got.Available != c.available || got.StockLabel != c.label {
			t.Errorf("stock %d: available %d label %q, want %d %q", c.stock, got.Available, got.StockLabel, c.available, c.label)
		}
		rec = a.do(http.MethodGet, "/products/"+strconv.FormatInt(got.ID, 10), tok, "")
		if again := decode[productJSON](t, rec); again.StockLabel != c.label {
			t.Errorf("get stock %d: label %q, want %q", c.stock, again.StockLabel, c.label)
		}
	}

	var page struct {
		Items []productJSON `json:"items"`
	}
	if err := json.Unmarshal(a.do(http.MethodGet, "/products", tok, "").Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 3 || page.Items[0].StockLabel != "in_stock" || page.Items[2].StockLabel != "out_of_stock" {
		t.Fatalf("list labels = %+v", page.Items)
	}
}
