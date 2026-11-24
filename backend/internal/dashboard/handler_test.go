package dashboard

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
)

type dashboardFixture struct {
	mux    *http.ServeMux
	repo   *memRepository
	tokens *auth.TokenIssuer
}

func newFixture(t *testing.T) dashboardFixture {
	t.Helper()
	repo := newMemRepository()
	tokens, err := auth.NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(newTestService(t, repo))
	mux := http.NewServeMux()
	mux.Handle("GET /dashboard/summary", auth.RequireAuth(tokens)(http.HandlerFunc(h.Summary)))
	return dashboardFixture{mux: mux, repo: repo, tokens: tokens}
}

func (f dashboardFixture) get(t *testing.T, shopID int64) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/summary", nil)
	if shopID != 0 {
		token, _, err := f.tokens.Issue(shopID*10, shopID)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	return rec
}

func TestHandlerSummary(t *testing.T) {
	f := newFixture(t)
	f.repo.titles[10] = "Phone"
	delivered := utc("2026-11-20T10:00:00Z")
	f.repo.add(memOrder{shopID: 1, status: "delivered", total: 3000, createdAt: delivered, changedAt: delivered, items: []memItem{{10, 3, 1000}}})
	f.repo.add(memOrder{shopID: 1, status: "new", total: 500, createdAt: now})
	f.repo.add(memOrder{shopID: 2, status: "new", total: 500, createdAt: now})

	rec := f.get(t, 1)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want JSON", ct)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"today_orders", "waiting_for_action", "revenue_7d", "revenue_30d", "top_products"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("body %s is missing %q", rec.Body, key)
		}
	}
	if len(raw) != 5 {
		t.Errorf("body %s has %d keys, want exactly 5", rec.Body, len(raw))
	}
	want := `{"today_orders":1,"waiting_for_action":1,"revenue_7d":3000,"revenue_30d":3000,` +
		`"top_products":[{"product_id":10,"title":"Phone","quantity_sold":3,"revenue":3000}]}`
	if got := strings.TrimSpace(rec.Body.String()); got != want {
		t.Fatalf("body = %s\nwant   %s", got, want)
	}
}

func TestHandlerSummaryNewShopIsZerosAndEmptyList(t *testing.T) {
	rec := newFixture(t).get(t, 5)
	want := `{"today_orders":0,"waiting_for_action":0,"revenue_7d":0,"revenue_30d":0,"top_products":[]}`
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != want {
		t.Fatalf("status %d body %s, want 200 and %s", rec.Code, rec.Body, want)
	}
}

func TestHandlerSummaryNeedsAToken(t *testing.T) {
	rec := newFixture(t).get(t, 0)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	var e struct{ Code, Message string }
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil || e.Code != "unauthorized" || e.Message == "" {
		t.Fatalf("body = %s, want {code: unauthorized, message}", rec.Body)
	}
}

func TestHandlerSummaryHidesInternalErrors(t *testing.T) {
	f := newFixture(t)
	f.repo.err = errors.New(`pq: relation "orders" does not exist`)
	rec := f.get(t, 1)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var e struct{ Code, Message string }
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil || e.Code != "internal" {
		t.Fatalf("body = %s, want code internal", rec.Body)
	}
	if strings.Contains(rec.Body.String(), "relation") {
		t.Fatalf("body %s leaks the error", rec.Body)
	}
}
