//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/0abdurahmon/seller-panel/backend/internal/config"
)

type orderItemJSON struct {
	ID        int64  `json:"id"`
	ProductID int64  `json:"product_id"`
	Title     string `json:"title"`
	Price     int64  `json:"price"`
	Quantity  int    `json:"quantity"`
}

type historyJSON struct {
	FromStatus *string `json:"from_status"`
	ToStatus   string  `json:"to_status"`
	Reason     *string `json:"reason"`
	CreatedAt  string  `json:"created_at"`
}

type orderJSON struct {
	ID              int64           `json:"id"`
	Number          string          `json:"number"`
	Status          string          `json:"status"`
	BuyerName       string          `json:"buyer_name"`
	BuyerPhone      string          `json:"buyer_phone"`
	DeliveryAddress string          `json:"delivery_address"`
	Items           []orderItemJSON `json:"items"`
	Total           int64           `json:"total"`
	CancelReason    *string         `json:"cancel_reason"`
	CreatedAt       string          `json:"created_at"`
	StatusChangedAt string          `json:"status_changed_at"`
	History         []historyJSON   `json:"history"`
}

type orderSummaryJSON struct {
	ID        int64  `json:"id"`
	Number    string `json:"number"`
	Status    string `json:"status"`
	BuyerName string `json:"buyer_name"`
	Total     int64  `json:"total"`
	CreatedAt string `json:"created_at"`
}

type orderPage struct {
	Items    []orderSummaryJSON `json:"items"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Total    int                `json:"total"`
}

type orderLine struct {
	productID int64
	quantity  int
}

func (s *server) insertOrder(status string, shopID int64, lines ...orderLine) int64 {
	s.t.Helper()
	ctx := context.Background()
	type row struct {
		productID int64
		title     string
		price     int64
		quantity  int
	}
	var rows []row
	var total int64
	for _, l := range lines {
		r := row{productID: l.productID, quantity: l.quantity}
		if err := s.pool.QueryRow(ctx, `SELECT title, price FROM products WHERE id = $1 AND shop_id = $2`, l.productID, shopID).Scan(&r.title, &r.price); err != nil {
			s.t.Fatal(err)
		}
		total += r.price * int64(r.quantity)
		rows = append(rows, r)
	}
	var reason any
	if status == "cancelled" {
		reason = "seeded"
	}
	var id int64
	if err := s.pool.QueryRow(ctx,
		`INSERT INTO orders (shop_id, status, buyer_name, buyer_phone, delivery_address, total, cancel_reason)
		 VALUES ($1, $2::order_status, 'E2E buyer', '+998901234567', 'Tashkent, Chilonzor 7', $3, $4) RETURNING id`,
		shopID, status, total, reason).Scan(&id); err != nil {
		s.t.Fatal(err)
	}
	reserves := status == "accepted" || status == "packed" || status == "handed_to_courier"
	for _, r := range rows {
		if _, err := s.pool.Exec(ctx,
			`INSERT INTO order_items (order_id, product_id, title_snapshot, price_snapshot, quantity) VALUES ($1, $2, $3, $4, $5)`,
			id, r.productID, r.title, r.price, r.quantity); err != nil {
			s.t.Fatal(err)
		}
		if reserves {
			if _, err := s.pool.Exec(ctx, `UPDATE products SET reserved = reserved + $2 WHERE id = $1`, r.productID, r.quantity); err != nil {
				s.t.Fatal(err)
			}
		}
	}
	return id
}

func orderPath(id int64) string { return "/orders/" + strconv.FormatInt(id, 10) }

func (s *server) transition(token string, id int64, to, reason string) response {
	s.t.Helper()
	body := map[string]string{"to": to}
	if reason != "" {
		body["reason"] = reason
	}
	return s.do(http.MethodPost, orderPath(id)+"/transition", token, body)
}

func (s *server) getProduct(token string, id int64) product {
	s.t.Helper()
	var p product
	res := s.do(http.MethodGet, "/products/"+strconv.FormatInt(id, 10), token, nil)
	s.expect(res, http.StatusOK, "")
	res.decode(&p)
	return p
}

func (s *server) getOrder(token string, id int64) orderJSON {
	s.t.Helper()
	var o orderJSON
	res := s.do(http.MethodGet, orderPath(id), token, nil)
	s.expect(res, http.StatusOK, "")
	res.decode(&o)
	return o
}

func TestOrderJourneyFromNewToBalance(t *testing.T) {
	s := newServer(t)
	sel := s.register()
	p := s.createProduct(sel.token, s.categoryID(sel.token), 10)
	id := s.insertOrder("new", p.ShopID, orderLine{p.ID, 3})
	total := 3 * p.Price

	var empty orderPage
	s.do(http.MethodGet, "/orders?status=delivered", sel.token, nil).decode(&empty)
	if empty.Total != 0 || empty.Items == nil {
		t.Fatalf("delivered filter before anything = %+v, want an empty page", empty)
	}

	var page orderPage
	s.do(http.MethodGet, "/orders", sel.token, nil).decode(&page)
	if page.Total != 1 || page.Items[0].ID != id || page.Items[0].Total != total || page.Items[0].Status != "new" {
		t.Fatalf("list = %+v", page)
	}

	renamed := productBody(p.CategoryID, p.SKU, 0)
	delete(renamed, "stock")
	renamed["title"], renamed["price"] = "Renamed later", 99
	s.expect(s.do(http.MethodPut, productPath(p), sel.token, renamed), http.StatusOK, "")

	o := s.getOrder(sel.token, id)
	if len(o.Number) < 7 || o.Number[0] != '#' || o.Status != "new" || len(o.History) != 0 {
		t.Fatalf("new order = %+v", o)
	}
	if len(o.Items) != 1 || o.Items[0].Title != p.Title || o.Items[0].Price != p.Price || o.Items[0].Quantity != 3 {
		t.Fatalf("items = %+v, want the title and price from the time of the order, not the later edit", o.Items)
	}

	s.expect(s.transition(sel.token, id, "accepted", ""), http.StatusOK, "")
	if got := s.getProduct(sel.token, p.ID); got.Stock != 10 || got.Reserved != 3 || (got.Stock-got.Reserved) != 7 {
		t.Fatalf("after accept product = stock %d reserved %d available %d, want 10/3/7", got.Stock, got.Reserved, (got.Stock - got.Reserved))
	}
	s.expect(s.transition(sel.token, id, "packed", ""), http.StatusOK, "")
	s.expect(s.transition(sel.token, id, "handed_to_courier", ""), http.StatusOK, "")
	res := s.transition(sel.token, id, "delivered", "")
	s.expect(res, http.StatusOK, "")
	res.decode(&o)
	if o.Status != "delivered" || len(o.History) != 4 {
		t.Fatalf("delivered order = status %s, %d history rows", o.Status, len(o.History))
	}
	if got := s.getProduct(sel.token, p.ID); got.Stock != 7 || got.Reserved != 0 || (got.Stock-got.Reserved) != 7 {
		t.Fatalf("after delivery product = stock %d reserved %d available %d, want 7/0/7", got.Stock, got.Reserved, (got.Stock - got.Reserved))
	}

	history := s.getOrder(sel.token, id).History
	want := []string{"accepted", "packed", "handed_to_courier", "delivered"}
	for i, h := range history {
		if h.ToStatus != want[i] || h.FromStatus == nil || h.CreatedAt == "" {
			t.Errorf("history[%d] = %+v, want to %s", i, h, want[i])
		}
	}
	if *history[0].FromStatus != "new" {
		t.Errorf("first history row starts from %q, want new", *history[0].FromStatus)
	}

	commission := total * 10 / 100
	var bal balanceSummary
	s.do(http.MethodGet, "/balance", sel.token, nil).decode(&bal)
	if bal.Earned != total || bal.Commission != -commission || bal.Available != total-commission {
		t.Fatalf("balance = %+v, want earned %d, commission %d, available %d", bal, total, -commission, total-commission)
	}
	var entries balancePage
	s.do(http.MethodGet, "/balance/entries", sel.token, nil).decode(&entries)
	if entries.Total != 2 {
		t.Fatalf("balance entries = %+v, want a sale and a commission", entries)
	}
	for _, e := range entries.Items {
		if e.OrderID == nil || *e.OrderID != id {
			t.Errorf("entry %+v is not linked to order %d", e, id)
		}
	}

	var delivered orderPage
	s.do(http.MethodGet, "/orders?status=delivered", sel.token, nil).decode(&delivered)
	s.do(http.MethodGet, "/orders?status=new", sel.token, nil).decode(&empty)
	if delivered.Total != 1 || empty.Total != 0 {
		t.Fatalf("status filter: delivered %d, new %d, want 1 and 0", delivered.Total, empty.Total)
	}
}

func TestCommissionRoundsDownToWholeSom(t *testing.T) {
	s := newServer(t)
	sel := s.register()
	category := s.categoryID(sel.token)
	body := productBody(category, uniqueSKU(), 5)
	body["price"] = 4115
	res := s.postProduct(sel.token, body)
	s.expect(res, http.StatusCreated, "")
	var p product
	res.decode(&p)

	id := s.insertOrder("handed_to_courier", p.ShopID, orderLine{p.ID, 3})
	s.expect(s.transition(sel.token, id, "delivered", ""), http.StatusOK, "")

	var bal balanceSummary
	s.do(http.MethodGet, "/balance", sel.token, nil).decode(&bal)
	if bal.Earned != 12345 || bal.Commission != -1234 || bal.Available != 11111 {
		t.Fatalf("balance = %+v, want 12345, -1234 and 11111 (10%% of 12345 rounds down, no so'm is lost)", bal)
	}
}

func TestCancelRules(t *testing.T) {
	s := newServer(t)
	sel := s.register()
	p := s.createProduct(sel.token, s.categoryID(sel.token), 20)

	fromNew := s.insertOrder("new", p.ShopID, orderLine{p.ID, 2})
	s.expect(s.transition(sel.token, fromNew, "cancelled", "   "), http.StatusBadRequest, "validation_error")
	s.expect(s.transition(sel.token, fromNew, "cancelled", "buyer changed mind"), http.StatusOK, "")
	if got := s.getProduct(sel.token, p.ID); got.Reserved != 0 {
		t.Fatalf("cancelling a new order changed reserved to %d", got.Reserved)
	}
	o := s.getOrder(sel.token, fromNew)
	if o.Status != "cancelled" || o.CancelReason == nil || *o.CancelReason != "buyer changed mind" {
		t.Fatalf("cancelled order = %+v", o)
	}

	for _, from := range []string{"accepted", "packed"} {
		id := s.insertOrder(from, p.ShopID, orderLine{p.ID, 4})
		before := s.getProduct(sel.token, p.ID).Reserved
		s.expect(s.transition(sel.token, id, "cancelled", "out of stock"), http.StatusOK, "")
		if after := s.getProduct(sel.token, p.ID).Reserved; after != before-4 {
			t.Fatalf("cancelling a %s order: reserved %d -> %d, want it to drop by 4", from, before, after)
		}
	}

	handed := s.insertOrder("handed_to_courier", p.ShopID, orderLine{p.ID, 1})
	s.expect(s.transition(sel.token, handed, "cancelled", "too late"), http.StatusConflict, "invalid_transition")
	if got := s.getOrder(sel.token, handed); got.Status != "handed_to_courier" {
		t.Fatalf("order moved to %s although the cancel was refused", got.Status)
	}

	var bal balanceSummary
	s.do(http.MethodGet, "/balance", sel.token, nil).decode(&bal)
	if bal.Available != 0 {
		t.Fatalf("cancelled orders changed the balance: %+v", bal)
	}
}

func TestEveryStatusPairIsEnforcedByTheBackend(t *testing.T) {
	s := newServer(t)
	sel := s.register()
	p := s.createProduct(sel.token, s.categoryID(sel.token), 1000)
	statuses := []string{"new", "accepted", "packed", "handed_to_courier", "delivered", "cancelled"}
	allowed := map[[2]string]bool{
		{"new", "accepted"}: true, {"new", "cancelled"}: true,
		{"accepted", "packed"}: true, {"accepted", "cancelled"}: true,
		{"packed", "handed_to_courier"}: true, {"packed", "cancelled"}: true,
		{"handed_to_courier", "delivered"}: true,
	}
	for _, from := range statuses {
		for _, to := range statuses {
			id := s.insertOrder(from, p.ShopID, orderLine{p.ID, 1})
			res := s.transition(sel.token, id, to, "reason")
			if allowed[[2]string{from, to}] {
				if res.status != http.StatusOK {
					t.Errorf("%s -> %s: status %d, want 200 (%s)", from, to, res.status, res.body)
				}
				continue
			}
			if res.status != http.StatusConflict || res.errorCode() != "invalid_transition" {
				t.Errorf("%s -> %s: status %d, want 409 invalid_transition (%s)", from, to, res.status, res.body)
			}
			if got := s.getOrder(sel.token, id); got.Status != from {
				t.Errorf("%s -> %s was refused but the order is now %s", from, to, got.Status)
			}
		}
	}
}

func TestAcceptNeedsEnoughStock(t *testing.T) {
	s := newServer(t)
	sel := s.register()
	category := s.categoryID(sel.token)
	scarce := s.createProduct(sel.token, category, 2)
	plenty := s.createProduct(sel.token, category, 10)
	id := s.insertOrder("new", scarce.ShopID, orderLine{plenty.ID, 2}, orderLine{scarce.ID, 3})

	s.expect(s.transition(sel.token, id, "accepted", ""), http.StatusConflict, "insufficient_stock")
	if got := s.getOrder(sel.token, id); got.Status != "new" || len(got.History) != 0 {
		t.Fatalf("order after a refused accept = %s with %d history rows", got.Status, len(got.History))
	}
	if got := s.getProduct(sel.token, plenty.ID); got.Reserved != 0 {
		t.Fatalf("the product that did have stock kept %d reserved units", got.Reserved)
	}

	level := s.do(http.MethodPatch, productPath(scarce)+"/stock", sel.token, map[string]int{"stock": 5, "expected_stock": 2})
	s.expect(level, http.StatusOK, "")
	s.expect(s.transition(sel.token, id, "accepted", ""), http.StatusOK, "")
	if got := s.getProduct(sel.token, scarce.ID); got.Reserved != 3 || (got.Stock-got.Reserved) != 2 {
		t.Fatalf("scarce product reserved %d available %d, want 3 and 2", got.Reserved, (got.Stock - got.Reserved))
	}
}

func TestConcurrentAcceptsCannotOversell(t *testing.T) {
	s := newServer(t)
	sel := s.register()
	p := s.createProduct(sel.token, s.categoryID(sel.token), 10)
	a := s.insertOrder("new", p.ShopID, orderLine{p.ID, 6})
	b := s.insertOrder("new", p.ShopID, orderLine{p.ID, 6})

	statuses := runAtOnce(t, 2, func(i int) response {
		id := []int64{a, b}[i]
		res, err := s.tryDo(http.MethodPost, orderPath(id)+"/transition", sel.token, map[string]string{"to": "accepted"})
		if err != nil {
			t.Error(err)
		}
		return res
	})
	ok, refused := 0, 0
	for _, res := range statuses {
		switch res.status {
		case http.StatusOK:
			ok++
		case http.StatusConflict:
			refused++
			if res.errorCode() != "insufficient_stock" {
				t.Errorf("refused with %s, want insufficient_stock", res.body)
			}
		default:
			t.Errorf("unexpected status %d: %s", res.status, res.body)
		}
	}
	if ok != 1 || refused != 1 {
		t.Fatalf("%d accepted and %d refused, want exactly one of each for 6 + 6 units on stock 10", ok, refused)
	}
	if got := s.getProduct(sel.token, p.ID); got.Reserved != 6 || (got.Stock-got.Reserved) != 4 {
		t.Fatalf("reserved %d available %d, want 6 and 4", got.Reserved, (got.Stock - got.Reserved))
	}
}

func TestSameOrderAcceptedTwiceAtOnce(t *testing.T) {
	s := newServer(t)
	sel := s.register()
	p := s.createProduct(sel.token, s.categoryID(sel.token), 10)
	id := s.insertOrder("new", p.ShopID, orderLine{p.ID, 4})

	results := runAtOnce(t, 2, func(int) response {
		res, err := s.tryDo(http.MethodPost, orderPath(id)+"/transition", sel.token, map[string]string{"to": "accepted"})
		if err != nil {
			t.Error(err)
		}
		return res
	})
	ok, invalid := 0, 0
	for _, res := range results {
		switch {
		case res.status == http.StatusOK:
			ok++
		case res.status == http.StatusConflict && res.errorCode() == "invalid_transition":
			invalid++
		default:
			t.Errorf("unexpected response %d: %s", res.status, res.body)
		}
	}
	if ok != 1 || invalid != 1 {
		t.Fatalf("%d accepted and %d refused, want one of each", ok, invalid)
	}
	if got := s.getProduct(sel.token, p.ID); got.Reserved != 4 {
		t.Fatalf("reserved = %d, want 4 (the stock must be reserved once)", got.Reserved)
	}
	if h := s.getOrder(sel.token, id).History; len(h) != 1 {
		t.Fatalf("history rows = %d, want 1", len(h))
	}
}

func runAtOnce(t *testing.T, n int, fn func(i int) response) []response {
	t.Helper()
	start := make(chan struct{})
	out := make([]response, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			out[i] = fn(i)
		}()
	}
	close(start)
	wg.Wait()
	return out
}

func TestSellerCannotSeeOrTouchAnotherSellersOrders(t *testing.T) {
	s := newServer(t)
	a, b := s.register(), s.register()
	p := s.createProduct(a.token, s.categoryID(a.token), 10)
	id := s.insertOrder("new", p.ShopID, orderLine{p.ID, 2})

	s.expect(s.do(http.MethodGet, orderPath(id), b.token, nil), http.StatusNotFound, "not_found")
	s.expect(s.transition(b.token, id, "accepted", ""), http.StatusNotFound, "not_found")
	s.expect(s.transition(b.token, id, "cancelled", "mine now"), http.StatusNotFound, "not_found")

	var page orderPage
	s.do(http.MethodGet, "/orders", b.token, nil).decode(&page)
	if page.Total != 0 || page.Items == nil || len(page.Items) != 0 {
		t.Fatalf("seller B list = %+v, want an empty page", page)
	}
	if got := s.getProduct(a.token, p.ID); got.Reserved != 0 {
		t.Fatalf("seller B's attempts reserved %d units of seller A's product", got.Reserved)
	}
	if got := s.getOrder(a.token, id); got.Status != "new" || len(got.History) != 0 {
		t.Fatalf("seller A's order changed: %+v", got)
	}

	var bal balanceSummary
	s.do(http.MethodGet, "/balance", b.token, nil).decode(&bal)
	s.expect(s.transition(a.token, s.insertOrder("handed_to_courier", p.ShopID, orderLine{p.ID, 1}), "delivered", ""), http.StatusOK, "")
	s.do(http.MethodGet, "/balance", b.token, nil).decode(&bal)
	if bal.Available != 0 || bal.Earned != 0 {
		t.Fatalf("seller B sees seller A's earnings: %+v", bal)
	}
}

func TestOrderRequestErrors(t *testing.T) {
	s := newServer(t)
	sel := s.register()
	p := s.createProduct(sel.token, s.categoryID(sel.token), 5)
	id := s.insertOrder("new", p.ShopID, orderLine{p.ID, 1})

	s.expect(s.do(http.MethodGet, "/orders/abc", sel.token, nil), http.StatusNotFound, "not_found")
	s.expect(s.do(http.MethodGet, "/orders/999999999", sel.token, nil), http.StatusNotFound, "not_found")
	s.expect(s.do(http.MethodPost, orderPath(id)+"/transition", sel.token, `{`), http.StatusBadRequest, "invalid_body")
	s.expect(s.do(http.MethodPost, orderPath(id)+"/transition", sel.token, `{"to":"accepted","extra":1}`), http.StatusBadRequest, "invalid_body")
	s.expect(s.transition(sel.token, id, "shipped", ""), http.StatusBadRequest, "validation_error")
	s.expect(s.transition(sel.token, id, "cancelled", strings.Repeat("ж", 501)), http.StatusBadRequest, "validation_error")
	s.expect(s.transition(sel.token, id, "cancelled", strings.Repeat("ж", 500)), http.StatusOK, "")
	s.expect(s.do(http.MethodGet, "/orders?status=shipped", sel.token, nil), http.StatusBadRequest, "validation_error")
	s.expect(s.do(http.MethodGet, "/orders?page=x", sel.token, nil), http.StatusBadRequest, "validation_error")
	s.expect(s.do(http.MethodGet, "/orders?page_size=x", sel.token, nil), http.StatusBadRequest, "validation_error")
}

func TestOrderListPagingIsClamped(t *testing.T) {
	s := newServer(t)
	sel := s.register()
	p := s.createProduct(sel.token, s.categoryID(sel.token), 10)
	var ids []int64
	for i := 0; i < 5; i++ {
		ids = append(ids, s.insertOrder("new", p.ShopID, orderLine{p.ID, 1}))
	}

	var page orderPage
	s.do(http.MethodGet, "/orders?page_size=500&page=0", sel.token, nil).decode(&page)
	if page.PageSize != 100 || page.Page != 1 || page.Total != 5 || len(page.Items) != 5 {
		t.Fatalf("clamped page = %+v", page)
	}
	if page.Items[0].ID != ids[4] || page.Items[4].ID != ids[0] {
		t.Fatalf("orders must be newest first, got ids %d ... %d", page.Items[0].ID, page.Items[4].ID)
	}
	s.do(http.MethodGet, "/orders?page=3&page_size=2", sel.token, nil).decode(&page)
	if page.Total != 5 || len(page.Items) != 1 || page.Page != 3 || page.PageSize != 2 {
		t.Fatalf("page 3 of 2 = %+v", page)
	}
	s.do(http.MethodGet, "/orders?page=99&page_size=2", sel.token, nil).decode(&page)
	if len(page.Items) != 0 || page.Items == nil {
		t.Fatalf("page past the end = %+v", page)
	}
}

func TestSimulatedOrderGoesAllTheWayToTheBalance(t *testing.T) {
	s := newServer(t)
	sel := s.register()
	category := s.categoryID(sel.token)
	a := s.createProduct(sel.token, category, 10)
	b := s.createProduct(sel.token, category, 6)
	owned := map[int64]product{a.ID: a, b.ID: b}

	res := s.do(http.MethodPost, "/dev/simulate-order", sel.token, nil)
	s.expect(res, http.StatusCreated, "")
	var o orderJSON
	res.decode(&o)
	if o.Status != "new" || len(o.Number) < 7 || o.Number[0] != '#' || o.BuyerName == "" || o.BuyerPhone == "" || o.DeliveryAddress == "" {
		t.Fatalf("simulated order = %+v", o)
	}
	if len(o.Items) < 1 || len(o.Items) > 2 || len(o.History) != 1 || o.History[0].FromStatus != nil || o.History[0].ToStatus != "new" {
		t.Fatalf("items %d, history %+v", len(o.Items), o.History)
	}
	var total int64
	for _, it := range o.Items {
		p, ok := owned[it.ProductID]
		if !ok || it.Quantity < 1 || it.Quantity > 3 || it.Price != p.Price || it.Title != p.Title {
			t.Fatalf("item %+v does not match the seller's products", it)
		}
		total += it.Price * int64(it.Quantity)
	}
	if o.Total != total {
		t.Fatalf("total = %d, want %d", o.Total, total)
	}
	before := s.getProduct(sel.token, o.Items[0].ProductID)
	if before.Reserved != 0 {
		t.Fatalf("simulating an order reserved %d units", before.Reserved)
	}

	for _, to := range []string{"accepted", "packed", "handed_to_courier", "delivered"} {
		s.expect(s.transition(sel.token, o.ID, to, ""), http.StatusOK, "")
	}
	after := s.getProduct(sel.token, o.Items[0].ProductID)
	if after.Stock != before.Stock-o.Items[0].Quantity || after.Reserved != 0 {
		t.Fatalf("after delivery stock %d reserved %d, want %d and 0", after.Stock, after.Reserved, before.Stock-o.Items[0].Quantity)
	}

	commission := total * 10 / 100
	var bal balanceSummary
	s.do(http.MethodGet, "/balance", sel.token, nil).decode(&bal)
	if bal.Earned != total || bal.Commission != -commission || bal.Available != total-commission {
		t.Fatalf("balance = %+v, want earned %d, commission %d", bal, total, -commission)
	}
}

func TestSimulateOrderNeedsSellableProducts(t *testing.T) {
	s := newServer(t)
	sel := s.register()
	s.expect(s.do(http.MethodPost, "/dev/simulate-order", sel.token, nil), http.StatusConflict, "no_products_to_order")

	p := s.createProduct(sel.token, s.categoryID(sel.token), 4)
	s.expect(s.do(http.MethodPost, productPath(p)+"/archive", sel.token, nil), http.StatusOK, "")
	s.expect(s.do(http.MethodPost, "/dev/simulate-order", sel.token, nil), http.StatusConflict, "no_products_to_order")

	other := s.register()
	s.createProduct(other.token, s.categoryID(other.token), 9)
	s.expect(s.do(http.MethodPost, "/dev/simulate-order", sel.token, nil), http.StatusConflict, "no_products_to_order")

	var page orderPage
	s.do(http.MethodGet, "/orders", other.token, nil).decode(&page)
	if page.Total != 0 {
		t.Fatalf("the refused simulations created %d orders for the other seller", page.Total)
	}
}

func TestSimulateOrderIsNotAvailableInProduction(t *testing.T) {
	s := newServer(t, func(c *config.Config) { c.Env = config.EnvProduction })
	sel := s.register()
	s.createProduct(sel.token, s.categoryID(sel.token), 5)

	s.expect(s.do(http.MethodPost, "/dev/simulate-order", sel.token, nil), http.StatusNotFound, "not_found")
	s.expect(s.do(http.MethodPost, "/dev/simulate-order", "", nil), http.StatusNotFound, "not_found")
	var page orderPage
	s.do(http.MethodGet, "/orders", sel.token, nil).decode(&page)
	if page.Total != 0 {
		t.Fatalf("production created %d orders", page.Total)
	}
}
