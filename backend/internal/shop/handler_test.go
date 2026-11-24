package shop

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
)

// The shop ID must come from the token. Seller A's token must never reach
// seller B's shop, whatever the request body says.
func TestHandlerScopesEachSellerToOwnShop(t *testing.T) {
	tokens, err := auth.NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewMemRepository()
	repo.Seed(1, "Shop A")
	repo.Seed(2, "Shop B")
	h := NewHandler(NewService(repo))

	mux := http.NewServeMux()
	mux.Handle("GET /shop", auth.RequireAuth(tokens)(http.HandlerFunc(h.Get)))
	mux.Handle("PUT /shop", auth.RequireAuth(tokens)(http.HandlerFunc(h.Update)))

	tokenA, _, _ := tokens.Issue(10, 1)
	tokenB, _, _ := tokens.Issue(20, 2)

	do := func(method, token, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/shop", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+token)
		mux.ServeHTTP(rec, req)
		return rec
	}

	// Seller B updates their shop. The body has no shop ID field, so only shop 2 can change.
	body := `{"name":"Hacked","description":"","logo_url":"","contact_phone":"","pickup_address":""}`
	if rec := do(http.MethodPut, tokenB, body); rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200", rec.Code)
	}

	var got Shop
	rec := do(http.MethodGet, tokenA, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", rec.Code)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != 1 || got.Name != "Shop A" {
		t.Fatalf("seller A sees %+v, want their own unchanged shop", got)
	}

	rec = do(http.MethodGet, tokenB, "")
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != 2 || got.Name != "Hacked" {
		t.Fatalf("seller B sees %+v, want shop 2 updated", got)
	}
}

func TestHandlerStatusCodes(t *testing.T) {
	tokens, _ := auth.NewTokenIssuer("test-secret", time.Hour)
	repo := NewMemRepository()
	repo.Seed(1, "Shop A")
	h := NewHandler(NewService(repo))
	mux := http.NewServeMux()
	mux.Handle("PUT /shop", auth.RequireAuth(tokens)(http.HandlerFunc(h.Update)))
	token, _, _ := tokens.Issue(10, 1)

	send := func(body string) int {
		req := httptest.NewRequest(http.MethodPut, "/shop", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec.Code
	}
	if code := send(`{"name":""}`); code != http.StatusBadRequest {
		t.Fatalf("empty name status = %d, want 400", code)
	}
	if code := send(`{"name":"X","unknown":1}`); code != http.StatusBadRequest {
		t.Fatalf("unknown field status = %d, want 400", code)
	}
	if code := send(`{"name":"X"}`); code != http.StatusOK {
		t.Fatalf("valid status = %d, want 200", code)
	}
}
