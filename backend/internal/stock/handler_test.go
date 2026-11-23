package stock

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
)

// newMux serves the stock route behind real auth middleware, like the router.
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

func issue(t *testing.T, tokens *auth.TokenIssuer, sellerID, shopID int64) string {
	t.Helper()
	token, _, err := tokens.Issue(sellerID, shopID)
	if err != nil {
		t.Fatal(err)
	}
	return token
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
	token := issue(t, tokens, 100, 1)

	rec := patch(t, mux, token, "/products/10/stock", `{"stock": 6, "expected_stock": 20}`)
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
		{"missing stock", "/products/10/stock", `{"expected_stock": 6}`, http.StatusBadRequest, "validation_error"},
		{"missing expected", "/products/10/stock", `{"stock": 5}`, http.StatusBadRequest, "validation_error"},
		{"empty body object", "/products/10/stock", `{}`, http.StatusBadRequest, "validation_error"},
		{"negative", "/products/10/stock", `{"stock": -3, "expected_stock": 6}`, http.StatusBadRequest, "validation_error"},
		{"fraction", "/products/10/stock", `{"stock": 2.5, "expected_stock": 6}`, http.StatusBadRequest, "invalid_body"},
		{"string", "/products/10/stock", `{"stock": "5", "expected_stock": 6}`, http.StatusBadRequest, "invalid_body"},
		{"unknown field", "/products/10/stock", `{"stock": 5, "expected_stock": 6, "reserved": 0}`, http.StatusBadRequest, "invalid_body"},
		{"not json", "/products/10/stock", `stock=5`, http.StatusBadRequest, "invalid_body"},
		{"stale expected", "/products/10/stock", `{"stock": 9, "expected_stock": 20}`, http.StatusConflict, "stock_changed"},
		{"below reserved", "/products/10/stock", `{"stock": 1, "expected_stock": 6}`, http.StatusConflict, "stock_below_reserved"},
		{"archived", "/products/11/stock", `{"stock": 9, "expected_stock": 5}`, http.StatusConflict, "product_archived"},
		{"missing product", "/products/999/stock", `{"stock": 9, "expected_stock": 0}`, http.StatusNotFound, "product_not_found"},
		{"bad id", "/products/abc/stock", `{"stock": 9, "expected_stock": 0}`, http.StatusNotFound, "product_not_found"},
		{"zero id", "/products/0/stock", `{"stock": 9, "expected_stock": 0}`, http.StatusNotFound, "product_not_found"},
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

func TestHandlerConflictCarriesCurrentStock(t *testing.T) {
	mux, store, tokens := newMux(t)
	store.Put(1, 10, 12, 2, false)
	token := issue(t, tokens, 100, 1)

	rec := patch(t, mux, token, "/products/10/stock", `{"stock": 30, "expected_stock": 10}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body %s", rec.Code, rec.Body)
	}
	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Current Level  `json:"current"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != "stock_changed" || body.Message == "" ||
		body.Current.Stock != 12 || body.Current.Available != 10 || body.Current.Label != InStock {
		t.Fatalf("body = %+v, want stock_changed with current stock 12", body)
	}

	if rec := patch(t, mux, token, "/products/10/stock", `{"stock": 30, "expected_stock": 12}`); rec.Code != http.StatusOK {
		t.Fatalf("retry with fresh value: status = %d, want 200; body %s", rec.Code, rec.Body)
	}
}

func TestHandlerRequiresToken(t *testing.T) {
	mux, store, _ := newMux(t)
	store.Put(1, 10, 20, 0, false)
	if rec := patch(t, mux, "", "/products/10/stock", `{"stock": 5, "expected_stock": 20}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandlerSellerCannotChangeAnotherShopsStock(t *testing.T) {
	mux, store, tokens := newMux(t)
	store.Put(1, 10, 20, 0, false) // shop 1's product

	tokenB := issue(t, tokens, 200, 2)
	if rec := patch(t, mux, tokenB, "/products/10/stock", `{"stock": 0, "expected_stock": 20}`); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}

	tokenA := issue(t, tokens, 100, 1)
	rec := patch(t, mux, tokenA, "/products/10/stock", `{"stock": 20, "expected_stock": 20}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("shop 1's stock was changed by shop 2: status %d body %s", rec.Code, rec.Body)
	}
}
