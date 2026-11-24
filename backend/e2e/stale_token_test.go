//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

// A token stays valid after the database was reset, but the seller and shop in it are gone:
// the API tells the client to sign in again instead of answering 200 with empty lists and 500 on writes.
func TestATokenOfADeletedAccountIsRefused(t *testing.T) {
	s := newServer(t)
	old := s.register()
	cat := s.categoryID(old.token)
	s.createProduct(old.token, cat, 3)
	s.expect(s.do(http.MethodGet, "/products", old.token, nil), http.StatusOK, "")

	if _, err := s.pool.Exec(t.Context(), `TRUNCATE products, shops, sellers RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}

	// The check remembers a live shop for a short time; a fresh server has no memory, a second one stands in for it.
	later := newServer(t)
	for _, path := range []string{"/products", "/shop", "/orders", "/balance", "/dashboard/summary", "/categories"} {
		later.expect(later.do(http.MethodGet, path, old.token, nil), http.StatusUnauthorized, "unauthorized")
	}
	later.expect(later.do(http.MethodPost, "/products", old.token, productBody(cat, uniqueSKU(), 1)), http.StatusUnauthorized, "unauthorized")
	later.expect(later.do(http.MethodGet, "/health", "", nil), http.StatusOK, "")

	fresh := later.register()
	later.expect(later.do(http.MethodGet, "/shop", fresh.token, nil), http.StatusOK, "")
}
