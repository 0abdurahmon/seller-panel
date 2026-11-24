//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/config"
)

type topProductJSON struct {
	ProductID    int64  `json:"product_id"`
	Title        string `json:"title"`
	QuantitySold int64  `json:"quantity_sold"`
	Revenue      int64  `json:"revenue"`
}

type dashboardJSON struct {
	TodayOrders      int              `json:"today_orders"`
	WaitingForAction int              `json:"waiting_for_action"`
	Revenue7D        int64            `json:"revenue_7d"`
	Revenue30D       int64            `json:"revenue_30d"`
	TopProducts      []topProductJSON `json:"top_products"`
}

func (s *server) summary(token string) dashboardJSON {
	s.t.Helper()
	res := s.do(http.MethodGet, "/dashboard/summary", token, nil)
	s.expect(res, http.StatusOK, "")
	var d dashboardJSON
	res.decode(&d)
	return d
}

// businessDayStart is midnight of today in the business time zone, the start of "today" on the dashboard.
func businessDayStart(t *testing.T) time.Time {
	t.Helper()
	loc, err := time.LoadLocation(config.DefaultBusinessTimeZone)
	if err != nil {
		t.Fatal(err)
	}
	n := time.Now().In(loc)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, loc)
}

// setOrderTimes places an order at exact times, which the API cannot do.
func (s *server) setOrderTimes(id int64, created, changed time.Time) {
	s.t.Helper()
	if _, err := s.pool.Exec(context.Background(),
		`UPDATE orders SET created_at = $2, status_changed_at = $3 WHERE id = $1`, id, created, changed); err != nil {
		s.t.Fatal(err)
	}
}

func TestDashboardJourney(t *testing.T) {
	s := newServer(t)
	sel := s.register()

	empty := s.do(http.MethodGet, "/dashboard/summary", sel.token, nil)
	s.expect(empty, http.StatusOK, "")
	if !strings.Contains(string(empty.body), `"top_products":[]`) {
		t.Fatalf("new shop summary = %s, want top_products []", empty.body)
	}
	if got := s.summary(sel.token); !reflect.DeepEqual(got, dashboardJSON{TopProducts: []topProductJSON{}}) {
		t.Fatalf("new shop summary = %+v, want zeros", got)
	}

	category := s.categoryID(sel.token)
	p1, p2 := s.createProduct(sel.token, category, 100), s.createProduct(sel.token, category, 100)
	fresh := s.insertOrder("new", p1.ShopID, orderLine{p1.ID, 2})
	d1 := s.insertOrder("handed_to_courier", p1.ShopID, orderLine{p1.ID, 3})
	d2 := s.insertOrder("handed_to_courier", p1.ShopID, orderLine{p2.ID, 1})

	got := s.summary(sel.token)
	if got.TodayOrders != 3 || got.WaitingForAction != 1 || got.Revenue7D != 0 || got.Revenue30D != 0 || len(got.TopProducts) != 0 {
		t.Fatalf("before any delivery = %+v, want 3 today, 1 waiting, no revenue", got)
	}

	s.expect(s.transition(sel.token, d1, "delivered", ""), http.StatusOK, "")
	s.expect(s.transition(sel.token, d2, "delivered", ""), http.StatusOK, "")
	got = s.summary(sel.token)
	want := dashboardJSON{
		TodayOrders: 3, WaitingForAction: 1,
		Revenue7D: 4 * p1.Price, Revenue30D: 4 * p1.Price,
		TopProducts: []topProductJSON{
			{p1.ID, p1.Title, 3, 3 * p1.Price},
			{p2.ID, p2.Title, 1, p1.Price},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("after two deliveries = %+v, want %+v", got, want)
	}

	for _, to := range []string{"accepted", "packed"} {
		s.expect(s.transition(sel.token, fresh, to, ""), http.StatusOK, "")
		if got := s.summary(sel.token); got.WaitingForAction != 1 {
			t.Fatalf("order %s: waiting = %d, want 1", to, got.WaitingForAction)
		}
	}
	s.expect(s.transition(sel.token, fresh, "handed_to_courier", ""), http.StatusOK, "")
	if got := s.summary(sel.token); got.WaitingForAction != 0 || got.Revenue30D != want.Revenue30D {
		t.Fatalf("handed to courier = %+v, want nothing waiting and the same revenue", got)
	}
	s.expect(s.transition(sel.token, fresh, "delivered", ""), http.StatusOK, "")
	got = s.summary(sel.token)
	want.WaitingForAction = 0
	want.Revenue7D, want.Revenue30D = 6*p1.Price, 6*p1.Price
	want.TopProducts = []topProductJSON{
		{p1.ID, p1.Title, 5, 5 * p1.Price},
		{p2.ID, p2.Title, 1, p1.Price},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("after the last delivery = %+v, want %+v", got, want)
	}

	cancelled := s.insertOrder("new", p1.ShopID, orderLine{p1.ID, 1})
	s.expect(s.transition(sel.token, cancelled, "cancelled", "buyer changed their mind"), http.StatusOK, "")
	after := s.summary(sel.token)
	if after.TodayOrders != 4 || after.WaitingForAction != 0 || after.Revenue30D != want.Revenue30D || !reflect.DeepEqual(after.TopProducts, want.TopProducts) {
		t.Fatalf("after a cancelled order = %+v, want it counted as today's order only", after)
	}
}

func TestDashboardWindowsFollowBusinessDays(t *testing.T) {
	s := newServer(t)
	sel := s.register()
	p := s.createProduct(sel.token, s.categoryID(sel.token), 100)
	day := businessDayStart(t)
	sec := time.Second

	at := func(status string, created, changed time.Time) {
		id := s.insertOrder(status, p.ShopID, orderLine{p.ID, 1})
		s.setOrderTimes(id, created, changed)
	}
	long := day.AddDate(0, 0, -40)
	at("new", day, day)
	at("new", day.AddDate(0, 0, 1).Add(-sec), day.AddDate(0, 0, 1).Add(-sec))
	at("new", day.Add(-sec), day.Add(-sec))
	at("new", day.AddDate(0, 0, 1), day.AddDate(0, 0, 1))

	at("delivered", long, day.AddDate(0, 0, -6))
	at("delivered", long, day.AddDate(0, 0, -6).Add(-sec))
	at("delivered", long, day.AddDate(0, 0, -29))
	at("delivered", long, day.AddDate(0, 0, -29).Add(-sec))
	at("delivered", long, day.AddDate(0, 0, 1))

	got := s.summary(sel.token)
	if got.TodayOrders != 2 {
		t.Errorf("today_orders = %d, want 2: from 00:00 to 23:59:59 in %s", got.TodayOrders, config.DefaultBusinessTimeZone)
	}
	if got.WaitingForAction != 4 {
		t.Errorf("waiting_for_action = %d, want all 4 new orders, whatever their age", got.WaitingForAction)
	}
	if want := p.Price; got.Revenue7D != want {
		t.Errorf("revenue_7d = %d, want %d: today and the 6 days before it", got.Revenue7D, want)
	}
	if want := 3 * p.Price; got.Revenue30D != want {
		t.Errorf("revenue_30d = %d, want %d: today and the 29 days before it", got.Revenue30D, want)
	}
	if len(got.TopProducts) != 1 || got.TopProducts[0].QuantitySold != 3 || got.TopProducts[0].Revenue != 3*p.Price {
		t.Errorf("top_products = %+v, want %d units in the same 30 days", got.TopProducts, 3)
	}
}

func TestDashboardTopFiveProducts(t *testing.T) {
	s := newServer(t)
	sel := s.register()
	category := s.categoryID(sel.token)

	var products []product
	for i := 1; i <= 6; i++ {
		p := s.createProduct(sel.token, category, 100)
		products = append(products, p)
		s.insertOrder("delivered", p.ShopID, orderLine{p.ID, i})
	}
	renamed := productBody(category, products[5].SKU, 0)
	delete(renamed, "stock")
	renamed["title"] = "Renamed after the sale"
	s.expect(s.do(http.MethodPut, productPath(products[5]), sel.token, renamed), http.StatusOK, "")

	got := s.summary(sel.token)
	if len(got.TopProducts) != 5 {
		t.Fatalf("top_products has %d items, want 5: %+v", len(got.TopProducts), got.TopProducts)
	}
	for i, top := range got.TopProducts {
		p := products[5-i]
		if top.ProductID != p.ID || top.QuantitySold != int64(6-i) || top.Revenue != int64(6-i)*p.Price {
			t.Errorf("top_products[%d] = %+v, want product %d with %d units", i, top, p.ID, 6-i)
		}
	}
	if got.TopProducts[0].Title != "Renamed after the sale" {
		t.Errorf("top title = %q, want the current product title", got.TopProducts[0].Title)
	}
}

func TestSellerSeesOnlyTheirOwnDashboard(t *testing.T) {
	s := newServer(t)
	a, b := s.register(), s.register()
	p := s.createProduct(a.token, s.categoryID(a.token), 10)
	s.insertOrder("new", p.ShopID, orderLine{p.ID, 1})
	s.insertOrder("delivered", p.ShopID, orderLine{p.ID, 2})

	if got := s.summary(a.token); got.TodayOrders != 2 || got.WaitingForAction != 1 || got.Revenue30D != 2*p.Price || len(got.TopProducts) != 1 {
		t.Fatalf("seller A summary = %+v", got)
	}
	if got := s.summary(b.token); !reflect.DeepEqual(got, dashboardJSON{TopProducts: []topProductJSON{}}) {
		t.Fatalf("seller B sees %+v of seller A's shop, want zeros", got)
	}

	// The shop is chosen by the token, never by the query.
	res := s.do(http.MethodGet, "/dashboard/summary?shop_id="+strconv.FormatInt(p.ShopID, 10), b.token, nil)
	s.expect(res, http.StatusOK, "")
	var viaQuery dashboardJSON
	res.decode(&viaQuery)
	if viaQuery.TodayOrders != 0 || viaQuery.Revenue30D != 0 || len(viaQuery.TopProducts) != 0 {
		t.Fatalf("seller B read seller A's shop through the query: %+v", viaQuery)
	}
}

func TestDashboardRequestErrors(t *testing.T) {
	s := newServer(t)
	sel := s.register()

	s.expect(s.do(http.MethodGet, "/dashboard/summary", "", nil), http.StatusUnauthorized, "unauthorized")
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		res := s.do(method, "/dashboard/summary", sel.token, nil)
		s.expect(res, http.StatusMethodNotAllowed, "method_not_allowed")
		if res.header.Get("Allow") == "" {
			t.Errorf("%s: the Allow header is missing", method)
		}
	}
	s.expect(s.do(http.MethodGet, "/dashboard/nope", sel.token, nil), http.StatusNotFound, "not_found")
}

func TestDashboardIsLimitedPerShop(t *testing.T) {
	s := newServer(t, func(c *config.Config) { c.ShopRPS, c.ShopBurst = 0.001, 3 })
	a, b := s.register(), s.register()

	for i := 0; i < 3; i++ {
		s.summary(a.token)
	}
	limited := s.do(http.MethodGet, "/dashboard/summary", a.token, nil)
	s.expect(limited, http.StatusTooManyRequests, "rate_limited")
	if limited.header.Get("Retry-After") == "" {
		t.Fatal("Retry-After missing")
	}
	s.summary(b.token)
}
