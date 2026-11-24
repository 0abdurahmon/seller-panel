//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

type balanceSummary struct {
	Available  int64 `json:"available"`
	Earned     int64 `json:"earned"`
	Commission int64 `json:"commission"`
}

type balanceEntry struct {
	ID        int64  `json:"id"`
	OrderID   *int64 `json:"order_id"`
	Type      string `json:"type"`
	Amount    int64  `json:"amount"`
	CreatedAt string `json:"created_at"`
}

type balancePage struct {
	Items    []balanceEntry `json:"items"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
	Total    int            `json:"total"`
}

func (s *server) shopIDOf(sel seller) int64 {
	s.t.Helper()
	var id int64
	err := s.pool.QueryRow(context.Background(),
		`SELECT sh.id FROM shops sh JOIN sellers se ON se.id = sh.seller_id WHERE se.email = $1`, sel.email).Scan(&id)
	if err != nil {
		s.t.Fatal(err)
	}
	return id
}

// deliverOrder writes what the order feature writes on delivery: a sale and a commission.
// Orders cannot be delivered through the API yet, so the rows go in directly.
func (s *server) deliverOrder(shopID, total int64, at time.Time) {
	s.t.Helper()
	var order int64
	if err := s.pool.QueryRow(context.Background(),
		`INSERT INTO orders (shop_id, status, buyer_name, buyer_phone, delivery_address, total, created_at, status_changed_at)
		 VALUES ($1, 'delivered', 'E2E buyer', '+998901234567', 'Tashkent', $2, $3, $3) RETURNING id`,
		shopID, total, at).Scan(&order); err != nil {
		s.t.Fatal(err)
	}
	for _, e := range []struct {
		typ    string
		amount int64
	}{{"sale", total}, {"commission", -(total * 10 / 100)}} {
		if _, err := s.pool.Exec(context.Background(),
			`INSERT INTO balance_entries (shop_id, order_id, type, amount, created_at) VALUES ($1, $2, $3::balance_entry_type, $4, $5)`,
			shopID, order, e.typ, e.amount, at); err != nil {
			s.t.Fatal(err)
		}
	}
}

func TestBalanceJourney(t *testing.T) {
	s := newServer(t)
	sel := s.register()

	var zero balanceSummary
	s.expect(s.do(http.MethodGet, "/balance", sel.token, nil), http.StatusOK, "")
	s.do(http.MethodGet, "/balance", sel.token, nil).decode(&zero)
	if zero != (balanceSummary{}) {
		t.Fatalf("new shop balance = %+v, want zeros", zero)
	}
	empty := s.do(http.MethodGet, "/balance/entries", sel.token, nil)
	if !strings.Contains(string(empty.body), `"items":[]`) {
		t.Fatalf("new shop history = %s, want items []", empty.body)
	}

	shop := s.shopIDOf(sel)
	base := time.Now().UTC().Add(-time.Hour)
	s.deliverOrder(shop, 250000, base)
	s.deliverOrder(shop, 12345, base.Add(time.Minute))

	var sum balanceSummary
	s.do(http.MethodGet, "/balance", sel.token, nil).decode(&sum)
	if want := (balanceSummary{Available: 225000 + 11111, Earned: 262345, Commission: -26234}); sum != want {
		t.Fatalf("balance = %+v, want %+v", sum, want)
	}

	var page balancePage
	s.do(http.MethodGet, "/balance/entries?page_size=3", sel.token, nil).decode(&page)
	if page.Total != 4 || len(page.Items) != 3 || page.PageSize != 3 || page.Page != 1 {
		t.Fatalf("history page 1 = %+v, want 3 of 4 entries", page)
	}
	if page.Items[0].Type != "commission" || page.Items[0].Amount != -1234 || page.Items[0].OrderID == nil {
		t.Fatalf("newest entry = %+v, want the commission -1234 of the second order", page.Items[0])
	}
	s.do(http.MethodGet, "/balance/entries?page=2&page_size=3", sel.token, nil).decode(&page)
	if len(page.Items) != 1 || page.Items[0].Type != "sale" || page.Items[0].Amount != 250000 {
		t.Fatalf("history page 2 = %+v, want only the oldest sale 250000", page)
	}
}

func TestSellerSeesOnlyTheirOwnBalance(t *testing.T) {
	s := newServer(t)
	a, b := s.register(), s.register()
	s.deliverOrder(s.shopIDOf(a), 100000, time.Now().UTC())

	var sum balanceSummary
	s.do(http.MethodGet, "/balance", b.token, nil).decode(&sum)
	if sum != (balanceSummary{}) {
		t.Fatalf("seller B sees %+v of seller A's money", sum)
	}
	var page balancePage
	// The shop is chosen by the token, never by the query.
	s.do(http.MethodGet, "/balance/entries?shop_id="+strconv.FormatInt(s.shopIDOf(a), 10), b.token, nil).decode(&page)
	if page.Total != 0 || len(page.Items) != 0 || page.Items == nil {
		t.Fatalf("seller B history = %+v, want an empty page", page)
	}
	s.do(http.MethodGet, "/balance", a.token, nil).decode(&sum)
	if sum.Available != 90000 {
		t.Fatalf("seller A balance = %+v, want available 90000", sum)
	}
}

func TestBalanceHistoryRejectsBadPaging(t *testing.T) {
	s := newServer(t)
	sel := s.register()
	for _, q := range []string{"page=abc", "page_size=1.5", "page=99999999999999999999", "page=1000001"} {
		s.expect(s.do(http.MethodGet, "/balance/entries?"+q, sel.token, nil), http.StatusBadRequest, "validation_error")
	}
	var page balancePage
	s.do(http.MethodGet, "/balance/entries?page_size=500", sel.token, nil).decode(&page)
	if page.PageSize != 100 {
		t.Fatalf("page_size = %d, want 500 clamped to 100", page.PageSize)
	}
}
