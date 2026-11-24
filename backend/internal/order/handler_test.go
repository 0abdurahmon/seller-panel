package order

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
)

type api struct {
	t      *testing.T
	repo   *memRepository
	mux    *http.ServeMux
	tokens *auth.TokenIssuer
}

func newAPI(t *testing.T) *api {
	t.Helper()
	repo := newMemRepository()
	return newAPIWith(t, repo, repo)
}

func newAPIWith(t *testing.T, mem *memRepository, repo Repository) *api {
	t.Helper()
	tokens, err := auth.NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(repo, 10)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(svc)
	protect := auth.RequireAuth(tokens)
	mux := http.NewServeMux()
	mux.Handle("GET /orders", protect(http.HandlerFunc(h.List)))
	mux.Handle("GET /orders/{id}", protect(http.HandlerFunc(h.Get)))
	mux.Handle("POST /orders/{id}/transition", protect(http.HandlerFunc(h.Transition)))
	return &api{t: t, repo: mem, mux: mux, tokens: tokens}
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

func post(a *api, token string, id int64, body string) *httptest.ResponseRecorder {
	return a.do(http.MethodPost, "/orders/"+strconv.FormatInt(id, 10)+"/transition", token, body)
}

func TestHandlerRequiresToken(t *testing.T) {
	a := newAPI(t)
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/orders"},
		{http.MethodGet, "/orders/1"},
		{http.MethodPost, "/orders/1/transition"},
	} {
		if rec := a.do(c.method, c.path, "", `{}`); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want 401", c.method, c.path, rec.Code)
		}
	}
}

func TestHandlerOrderShape(t *testing.T) {
	a := newAPI(t)
	a.repo.AddProduct(7, shopA, 10, 0)
	a.repo.AddOrder(order(1, shopA, StatusNew, 3000, Item{ID: 1, ProductID: 7, Title: "Phone", Price: 1000, Quantity: 3}))

	rec := a.do(http.MethodGet, "/orders/1", a.token(shopA), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	var raw map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"id", "number", "status", "buyer_name", "buyer_phone", "delivery_address", "items", "total", "cancel_reason", "created_at", "status_changed_at", "history"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("order JSON has no %q", key)
		}
	}
	if _, leaked := raw["shop_id"]; leaked {
		t.Error("shop_id must not be in the order JSON")
	}
	if h, ok := raw["history"].([]any); !ok || len(h) != 0 {
		t.Errorf("history = %v, want an empty array", raw["history"])
	}
	items, _ := raw["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["title"] != "Phone" {
		t.Errorf("items = %v", raw["items"])
	}
}

func TestHandlerStatusCodes(t *testing.T) {
	a := newAPI(t)
	a.repo.AddProduct(7, shopA, 1, 0)
	a.repo.AddOrder(order(1, shopA, StatusNew, 3000, Item{ProductID: 7, Quantity: 3}))
	a.repo.AddOrder(order(2, shopA, StatusNew, 100, Item{ProductID: 99, Quantity: 1}))
	a.repo.AddOrder(order(3, shopA, StatusDelivered, 100))
	tok := a.token(shopA)

	cases := []struct {
		name         string
		method, path string
		body         string
		status       int
		code         string
	}{
		{"get non numeric id", http.MethodGet, "/orders/abc", "", 404, "not_found"},
		{"get missing", http.MethodGet, "/orders/999", "", 404, "not_found"},
		{"transition non numeric id", http.MethodPost, "/orders/abc/transition", `{"to":"accepted"}`, 404, "not_found"},
		{"transition missing", http.MethodPost, "/orders/999/transition", `{"to":"accepted"}`, 404, "not_found"},
		{"bad json", http.MethodPost, "/orders/1/transition", `{`, 400, "invalid_body"},
		{"unknown field", http.MethodPost, "/orders/1/transition", `{"to":"accepted","x":1}`, 400, "invalid_body"},
		{"unknown status", http.MethodPost, "/orders/1/transition", `{"to":"shipped"}`, 400, "validation_error"},
		{"cancel without reason", http.MethodPost, "/orders/1/transition", `{"to":"cancelled"}`, 400, "validation_error"},
		{"invalid transition", http.MethodPost, "/orders/1/transition", `{"to":"delivered"}`, 409, "invalid_transition"},
		{"final status", http.MethodPost, "/orders/3/transition", `{"to":"cancelled","reason":"x"}`, 409, "invalid_transition"},
		{"not enough stock", http.MethodPost, "/orders/1/transition", `{"to":"accepted"}`, 409, "insufficient_stock"},
		{"product gone", http.MethodPost, "/orders/2/transition", `{"to":"accepted"}`, 409, "product_not_found"},
		{"list bad page", http.MethodGet, "/orders?page=x", "", 400, "validation_error"},
		{"list bad page size", http.MethodGet, "/orders?page_size=x", "", 400, "validation_error"},
		{"list unknown status", http.MethodGet, "/orders?status=shipped", "", 400, "validation_error"},
	}
	for _, c := range cases {
		rec := a.do(c.method, c.path, tok, c.body)
		if rec.Code != c.status || errorCode(t, rec) != c.code {
			t.Errorf("%s: status %d code %q, want %d %q (%s)", c.name, rec.Code, errorCode(t, rec), c.status, c.code, rec.Body.String())
		}
	}
}

func TestHandlerWalksTheWholeStatusMachine(t *testing.T) {
	a := newAPI(t)
	a.repo.AddProduct(7, shopA, 10, 0)
	a.repo.AddOrder(order(1, shopA, StatusNew, 3000, Item{ProductID: 7, Quantity: 3, Price: 1000}))
	tok := a.token(shopA)

	for i, to := range []Status{StatusAccepted, StatusPacked, StatusHandedToCourier, StatusDelivered} {
		rec := post(a, tok, 1, `{"to":"`+string(to)+`"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("to %s: status %d: %s", to, rec.Code, rec.Body.String())
		}
		o := decode[Order](t, rec)
		if o.Status != to || len(o.History) != i+1 || o.History[i].ToStatus != to {
			t.Fatalf("to %s: order = %+v", to, o)
		}
	}
	if stock, reserved := a.repo.Product(7); stock != 7 || reserved != 0 {
		t.Fatalf("stock/reserved = %d/%d, want 7/0", stock, reserved)
	}
	if entries := a.repo.Balance(); len(entries) != 2 || entries[0].Amount != 3000 || entries[1].Amount != -300 {
		t.Fatalf("balance entries = %+v", entries)
	}
}

func TestHandlerShopsAreIsolated(t *testing.T) {
	a := newAPI(t)
	const shopB int64 = 2
	a.repo.AddProduct(7, shopA, 10, 0)
	a.repo.AddOrder(order(1, shopA, StatusNew, 100, Item{ProductID: 7, Quantity: 1}))
	tokB := a.token(shopB)

	if rec := a.do(http.MethodGet, "/orders/1", tokB, ""); rec.Code != http.StatusNotFound {
		t.Errorf("shop B GET status = %d, want 404", rec.Code)
	}
	if rec := post(a, tokB, 1, `{"to":"accepted"}`); rec.Code != http.StatusNotFound {
		t.Errorf("shop B transition status = %d, want 404", rec.Code)
	}
	if _, reserved := a.repo.Product(7); reserved != 0 {
		t.Errorf("shop B's attempt reserved %d units of shop A's product", reserved)
	}
	page := decode[Page[Summary]](t, a.do(http.MethodGet, "/orders", tokB, ""))
	if page.Total != 0 || page.Items == nil || len(page.Items) != 0 {
		t.Errorf("shop B list = %+v, want an empty page with items []", page)
	}
}

func TestHandlerListFiltersAndClampsPaging(t *testing.T) {
	a := newAPI(t)
	for i := int64(1); i <= 3; i++ {
		st := StatusNew
		if i == 3 {
			st = StatusCancelled
		}
		a.repo.AddOrder(order(i, shopA, st, 0))
	}
	tok := a.token(shopA)

	page := decode[Page[Summary]](t, a.do(http.MethodGet, "/orders?page_size=500&page=0", tok, ""))
	if page.PageSize != 100 || page.Page != 1 || page.Total != 3 || len(page.Items) != 3 {
		t.Errorf("clamped page = %+v", page)
	}
	filtered := decode[Page[Summary]](t, a.do(http.MethodGet, "/orders?status=cancelled", tok, ""))
	if filtered.Total != 1 || filtered.Items[0].ID != 3 {
		t.Errorf("filtered = %+v", filtered)
	}
}

type brokenRepo struct{ *memRepository }

func (brokenRepo) Get(context.Context, int64, int64) (Order, error) {
	return Order{}, errors.New("connection reset by peer")
}

func TestHandlerHidesInternalErrors(t *testing.T) {
	mem := newMemRepository()
	a := newAPIWith(t, mem, brokenRepo{mem})
	rec := a.do(http.MethodGet, "/orders/1", a.token(shopA), "")
	if rec.Code != http.StatusInternalServerError || errorCode(t, rec) != "internal" {
		t.Fatalf("status %d body %s, want 500 internal", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "connection reset") {
		t.Fatalf("internal error leaked: %s", rec.Body.String())
	}
}
