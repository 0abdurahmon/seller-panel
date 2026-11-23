package stock

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
)

func TestLabelFor(t *testing.T) {
	cases := []struct {
		available, threshold int
		want                 Label
	}{
		{0, 5, OutOfStock},
		{-1, 5, OutOfStock}, // never stored, but must not read as in stock
		{1, 5, LowStock},
		{5, 5, LowStock}, // the threshold itself is low
		{6, 5, InStock},
		{1, 0, InStock}, // threshold 0 turns the warning off
		{0, 0, OutOfStock},
	}
	for _, c := range cases {
		if got := LabelFor(c.available, c.threshold); got != c.want {
			t.Errorf("LabelFor(%d, %d) = %s, want %s", c.available, c.threshold, got, c.want)
		}
	}
}

func newService(t *testing.T) (*Service, *MemStore) {
	t.Helper()
	store := NewMemStore()
	svc, err := NewService(store, 5)
	if err != nil {
		t.Fatal(err)
	}
	return svc, store
}

func TestNewServiceRejectsNegativeThreshold(t *testing.T) {
	if _, err := NewService(NewMemStore(), -1); err == nil {
		t.Fatal("negative threshold accepted")
	}
}

func TestSetStockReturnsLevelWithLabel(t *testing.T) {
	svc, store := newService(t)
	store.Put(1, 10, 20, 3, false)
	ctx := context.Background()

	cases := []struct {
		stock         int
		wantAvailable int
		wantLabel     Label
	}{
		{50, 47, InStock},
		{8, 5, LowStock},
		{3, 0, OutOfStock}, // everything left is reserved
	}
	for _, c := range cases {
		got, err := svc.SetStock(ctx, 1, 10, c.stock)
		if err != nil {
			t.Fatalf("SetStock(%d): %v", c.stock, err)
		}
		if got.ProductID != 10 || got.Stock != c.stock || got.Reserved != 3 ||
			got.Available != c.wantAvailable || got.Label != c.wantLabel {
			t.Fatalf("SetStock(%d) = %+v, want available %d label %s", c.stock, got, c.wantAvailable, c.wantLabel)
		}
	}
}

func TestSetStockRules(t *testing.T) {
	svc, store := newService(t)
	store.Put(1, 10, 20, 4, false) // active, 4 reserved
	store.Put(1, 11, 7, 0, true)   // archived
	store.Put(2, 12, 9, 0, false)  // belongs to shop 2
	ctx := context.Background()

	cases := []struct {
		name      string
		productID int64
		stock     int
		want      error
	}{
		{"negative", 10, -1, ErrValidation},
		{"above max", 10, MaxStock + 1, ErrValidation},
		{"below reserved", 10, 3, ErrBelowReserved},
		{"archived", 11, 5, ErrArchived},
		{"missing", 99, 5, ErrNotFound},
		{"bad id", 0, 5, ErrNotFound},
		{"other shop", 12, 5, ErrNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := svc.SetStock(ctx, 1, c.productID, c.stock)
			if !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
		})
	}

	// Equal to reserved, zero and the max are allowed edges.
	for _, stock := range []int{4, MaxStock} {
		if _, err := svc.SetStock(ctx, 1, 10, stock); err != nil {
			t.Fatalf("SetStock(%d): %v", stock, err)
		}
	}
	store.Put(1, 13, 5, 0, false)
	if got, err := svc.SetStock(ctx, 1, 13, 0); err != nil || got.Label != OutOfStock {
		t.Fatalf("SetStock(0) = %+v, %v; want out_of_stock", got, err)
	}
}

func TestSetStockRefusalLeavesStockUnchanged(t *testing.T) {
	svc, store := newService(t)
	store.Put(1, 10, 20, 4, false)
	ctx := context.Background()
	if _, err := svc.SetStock(ctx, 1, 10, 2); !errors.Is(err, ErrBelowReserved) {
		t.Fatalf("err = %v, want ErrBelowReserved", err)
	}
	got, err := svc.SetStock(ctx, 1, 10, 20)
	if err != nil || got.Stock != 20 || got.Reserved != 4 {
		t.Fatalf("after refusal = %+v, %v; want stock 20 reserved 4", got, err)
	}
}

// newMux serves the stock route behind the real auth middleware, the same way
// the router does.
func newMux(t *testing.T) (*http.ServeMux, *MemStore, *auth.TokenIssuer) {
	t.Helper()
	svc, store := newService(t)
	tokens, err := auth.NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("PATCH /products/{id}/stock", auth.RequireAuth(tokens)(http.HandlerFunc(NewHandler(svc).Update)))
	return mux, store, tokens
}

func patch(t *testing.T, mux http.Handler, token, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestHandlerUpdateStock(t *testing.T) {
	mux, store, tokens := newMux(t)
	store.Put(1, 10, 20, 2, false)
	store.Put(1, 11, 5, 0, true)
	token, _, err := tokens.Issue(100, 1)
	if err != nil {
		t.Fatal(err)
	}

	rec := patch(t, mux, token, "/products/10/stock", `{"stock": 6}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}
	var got Level
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ProductID != 10 || got.Stock != 6 || got.Available != 4 || got.Label != LowStock {
		t.Fatalf("body = %+v, want stock 6 available 4 low_stock", got)
	}

	cases := []struct {
		name, path, body string
		status           int
		code             string
	}{
		{"missing stock", "/products/10/stock", `{}`, http.StatusBadRequest, "validation_error"},
		{"negative", "/products/10/stock", `{"stock": -3}`, http.StatusBadRequest, "validation_error"},
		{"fraction", "/products/10/stock", `{"stock": 2.5}`, http.StatusBadRequest, "invalid_body"},
		{"string", "/products/10/stock", `{"stock": "5"}`, http.StatusBadRequest, "invalid_body"},
		{"unknown field", "/products/10/stock", `{"stock": 5, "reserved": 0}`, http.StatusBadRequest, "invalid_body"},
		{"not json", "/products/10/stock", `stock=5`, http.StatusBadRequest, "invalid_body"},
		{"below reserved", "/products/10/stock", `{"stock": 1}`, http.StatusConflict, "stock_below_reserved"},
		{"archived", "/products/11/stock", `{"stock": 9}`, http.StatusConflict, "product_archived"},
		{"missing product", "/products/999/stock", `{"stock": 9}`, http.StatusNotFound, "product_not_found"},
		{"bad id", "/products/abc/stock", `{"stock": 9}`, http.StatusNotFound, "product_not_found"},
		{"zero id", "/products/0/stock", `{"stock": 9}`, http.StatusNotFound, "product_not_found"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := patch(t, mux, token, c.path, c.body)
			if rec.Code != c.status {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, c.status, rec.Body)
			}
			var e struct{ Code string }
			if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil || e.Code != c.code {
				t.Fatalf("code = %q (%v), want %q", e.Code, err, c.code)
			}
		})
	}
}

func TestHandlerRequiresToken(t *testing.T) {
	mux, store, _ := newMux(t)
	store.Put(1, 10, 20, 0, false)
	if rec := patch(t, mux, "", "/products/10/stock", `{"stock": 5}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandlerSellerCannotChangeAnotherShopsStock(t *testing.T) {
	mux, store, tokens := newMux(t)
	store.Put(1, 10, 20, 0, false) // shop 1's product
	tokenB, _, err := tokens.Issue(200, 2)
	if err != nil {
		t.Fatal(err)
	}
	if rec := patch(t, mux, tokenB, "/products/10/stock", `{"stock": 0}`); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	tokenA, _, _ := tokens.Issue(100, 1)
	rec := patch(t, mux, tokenA, "/products/10/stock", `{"stock": 20}`)
	var got Level
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if rec.Code != http.StatusOK || got.Stock != 20 {
		t.Fatalf("shop 1 stock changed by shop 2: status %d body %s", rec.Code, rec.Body)
	}
}
