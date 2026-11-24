package balance

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
)

type balanceFixture struct {
	mux    *http.ServeMux
	repo   *memRepository
	tokens *auth.TokenIssuer
}

func newFixture(t *testing.T) balanceFixture {
	t.Helper()
	repo := newMemRepository()
	tokens, err := auth.NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(NewService(repo))
	protected := auth.RequireAuth(tokens)
	mux := http.NewServeMux()
	mux.Handle("GET /balance", protected(http.HandlerFunc(h.Summary)))
	mux.Handle("GET /balance/entries", protected(http.HandlerFunc(h.Entries)))
	return balanceFixture{mux: mux, repo: repo, tokens: tokens}
}

func (f balanceFixture) get(t *testing.T, shopID int64, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
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
	deliver(f.repo, 1, 100, 250000, t0)
	deliver(f.repo, 2, 200, 999000, t0)

	rec := f.get(t, 1, "/balance")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}
	var raw map[string]int64
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	want := map[string]int64{"available": 225000, "earned": 250000, "commission": -25000}
	if len(raw) != len(want) {
		t.Fatalf("body = %v, want exactly the keys %v", raw, want)
	}
	for k, v := range want {
		if raw[k] != v {
			t.Fatalf("%s = %d, want %d (body %v)", k, raw[k], v, raw)
		}
	}
}

func TestHandlerSummaryNewShopIsZeros(t *testing.T) {
	rec := newFixture(t).get(t, 5, "/balance")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"available":0,"earned":0,"commission":0}` {
		t.Fatalf("status %d body %s, want 200 and zeros", rec.Code, rec.Body)
	}
}

func TestHandlerEntries(t *testing.T) {
	f := newFixture(t)
	deliver(f.repo, 1, 100, 1000, t0)
	payout := f.repo.add(1, nil, EntryPayout, -300, t0.Add(time.Hour))
	deliver(f.repo, 2, 200, 5000, t0)

	rec := f.get(t, 1, "/balance/entries")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}
	var page struct {
		Items []struct {
			ID      int64  `json:"id"`
			OrderID *int64 `json:"order_id"`
			Type    string `json:"type"`
			Amount  int64  `json:"amount"`
		} `json:"items"`
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
		Total    int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Total != 3 || page.Page != 1 || page.PageSize != 20 || len(page.Items) != 3 {
		t.Fatalf("page = %+v, want total 3, page 1, size 20, 3 items", page)
	}
	if first := page.Items[0]; first.ID != payout || first.OrderID != nil || first.Type != "payout" || first.Amount != -300 {
		t.Fatalf("first item = %+v, want the payout with order_id null", first)
	}
	if strings.Contains(rec.Body.String(), "shop_id") {
		t.Fatalf("the response leaks shop_id: %s", rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"order_id":null`) || !strings.Contains(rec.Body.String(), `"created_at":"2026-11-20T11:00:00Z"`) {
		t.Fatalf("body = %s, want order_id null and a UTC timestamp", rec.Body)
	}
}

func TestHandlerEntriesEmptyIsArray(t *testing.T) {
	rec := newFixture(t).get(t, 5, "/balance/entries")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"items":[]`) {
		t.Fatalf("status %d body %s, want 200 and items []", rec.Code, rec.Body)
	}
}

func TestHandlerEntriesPaging(t *testing.T) {
	f := newFixture(t)
	for i := range 3 {
		f.repo.add(1, ptr(int64(i)), EntrySale, 100, t0.Add(time.Duration(i)*time.Minute))
	}
	var page Page
	rec := f.get(t, 1, "/balance/entries?page=2&page_size=2")
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK || page.Page != 2 || page.PageSize != 2 || len(page.Items) != 1 || page.Total != 3 {
		t.Fatalf("status %d page %+v, want page 2 of size 2 with 1 item of 3", rec.Code, page)
	}
}

func TestHandlerRejectsBadPaging(t *testing.T) {
	f := newFixture(t)
	for _, q := range []string{
		"page=abc", "page_size=1.5", "page=-", "page=99999999999999999999", "page=1000001", "page_size=ten",
	} {
		rec := f.get(t, 1, "/balance/entries?"+q)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"code":"validation_error"`) {
			t.Errorf("?%s: status %d body %s, want 400 validation_error", q, rec.Code, rec.Body)
		}
	}
}

func TestHandlerRequiresToken(t *testing.T) {
	f := newFixture(t)
	for _, path := range []string{"/balance", "/balance/entries"} {
		rec := f.get(t, 0, path)
		if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), `"code":"unauthorized"`) {
			t.Errorf("%s: status %d body %s, want 401 unauthorized", path, rec.Code, rec.Body)
		}
	}
}

func TestHandlerShopComesFromTheToken(t *testing.T) {
	f := newFixture(t)
	deliver(f.repo, 1, 100, 1000, t0)
	deliver(f.repo, 2, 200, 7000, t0)

	// The client cannot ask for another shop: query and headers are ignored.
	rec := f.get(t, 2, "/balance/entries?shop_id=1")
	var page Page
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || *page.Items[0].OrderID != 200 {
		t.Fatalf("shop 2 got %+v, want only its own two entries", page)
	}
}
