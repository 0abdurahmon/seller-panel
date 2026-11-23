package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/stock"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	tokens, err := auth.NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := auth.NewService(auth.NewMemStore(), tokens)
	if err != nil {
		t.Fatal(err)
	}
	stockSvc, err := stock.NewService(stock.NewMemStore(), 5)
	if err != nil {
		t.Fatal(err)
	}
	return NewRouter(Deps{Auth: auth.NewHandler(svc), Tokens: tokens, Stock: stock.NewHandler(stockSvc)})
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
