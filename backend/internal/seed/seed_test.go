package seed

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/database/dbtest"
	"github.com/0abdurahmon/seller-panel/backend/internal/order"
)

func tashkent(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Tashkent")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

// seedNow is 15:00 in Tashkent on 24 November 2026.
var seedNow = time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)

func TestThePlanIsConsistent(t *testing.T) {
	bySKU := map[string]productSpec{}
	counts := map[string]int{}
	for _, p := range products {
		if bySKU[p.sku].sku != "" {
			t.Errorf("sku %s is used twice", p.sku)
		}
		bySKU[p.sku] = p
		status := p.status
		if status == "" {
			status = "active"
		}
		counts[status]++
		if p.price <= 0 || p.title == "" || p.category == "" {
			t.Errorf("product %s is incomplete: %+v", p.sku, p)
		}
	}
	if counts["active"] != 13 || counts["draft"] != 1 || counts["archived"] != 1 {
		t.Errorf("products by status = %v, want 13 active, 1 draft, 1 archived", counts)
	}

	final := map[order.Status]int{}
	for i, o := range orders {
		final[o.final]++
		if o.buyer < 0 || o.buyer >= len(buyers) {
			t.Errorf("order %d uses the unknown buyer %d", i, o.buyer)
		}
		if o.daysAgo == 0 && (o.today <= 0 || o.today >= 1) {
			t.Errorf("order %d of today needs a day fraction between 0 and 1, got %v", i, o.today)
		}
		if o.daysAgo < 0 || o.daysAgo > 29 || (o.daysAgo > 0 && (o.hour < 8 || o.hour > 20)) {
			t.Errorf("order %d is placed %d days ago at %d o'clock", i, o.daysAgo, o.hour)
		}
		if o.final == order.StatusCancelled && (o.reason == "" || !order.CanTransition(o.cancelFrom, order.StatusCancelled)) {
			t.Errorf("order %d is cancelled without a reason or from a status that cannot be cancelled: %+v", i, o)
		}
		if len(o.lines) == 0 {
			t.Errorf("order %d has no items", i)
		}
		seen := map[string]bool{}
		for _, l := range o.lines {
			p, ok := bySKU[l.sku]
			switch {
			case !ok:
				t.Errorf("order %d uses the unknown product %s", i, l.sku)
			case p.status != "":
				t.Errorf("order %d uses %s, which is a %s product", i, l.sku, p.status)
			case l.qty <= 0 || seen[l.sku]:
				t.Errorf("order %d has a bad or repeated line for %s", i, l.sku)
			}
			seen[l.sku] = true
		}
	}
	want := map[order.Status]int{
		order.StatusNew: 4, order.StatusAccepted: 2, order.StatusPacked: 2,
		order.StatusHandedToCourier: 3, order.StatusDelivered: 15, order.StatusCancelled: 4,
	}
	for status, n := range want {
		if final[status] != n {
			t.Errorf("%d %s orders planned, want %d", final[status], status, n)
		}
	}
}

func TestStepsFollowTheStatusMachine(t *testing.T) {
	cases := []struct {
		plan plannedOrder
		want []order.Status
	}{
		{plannedOrder{final: order.StatusNew}, nil},
		{plannedOrder{final: order.StatusAccepted}, []order.Status{order.StatusAccepted}},
		{plannedOrder{final: order.StatusPacked}, []order.Status{order.StatusAccepted, order.StatusPacked}},
		{plannedOrder{final: order.StatusHandedToCourier}, []order.Status{order.StatusAccepted, order.StatusPacked, order.StatusHandedToCourier}},
		{plannedOrder{final: order.StatusDelivered}, []order.Status{order.StatusAccepted, order.StatusPacked, order.StatusHandedToCourier, order.StatusDelivered}},
		{plannedOrder{final: order.StatusCancelled, cancelFrom: order.StatusNew}, []order.Status{order.StatusCancelled}},
		{plannedOrder{final: order.StatusCancelled, cancelFrom: order.StatusAccepted}, []order.Status{order.StatusAccepted, order.StatusCancelled}},
		{plannedOrder{final: order.StatusCancelled, cancelFrom: order.StatusPacked}, []order.Status{order.StatusAccepted, order.StatusPacked, order.StatusCancelled}},
	}
	for _, c := range cases {
		got := stepsTo(c.plan)
		from := order.StatusNew
		for _, to := range got {
			if !order.CanTransition(from, to) {
				t.Errorf("%+v: %s to %s is not allowed by the status machine", c.plan, from, to)
			}
			from = to
		}
		if fmt.Sprint(got) != fmt.Sprint(c.want) {
			t.Errorf("stepsTo(%+v) = %v, want %v", c.plan, got, c.want)
		}
	}
}

// Whatever time of day the seed runs, nothing is dated after now and today's orders stay on today.
func TestTimesStayInThePastAtAnyTimeOfDay(t *testing.T) {
	loc := tashkent(t)
	for _, now := range []time.Time{
		time.Date(2026, 11, 24, 19, 0, 30, 0, time.UTC), // 00:00:30 in Tashkent
		time.Date(2026, 11, 24, 19, 5, 0, 0, time.UTC),  // 00:05
		time.Date(2026, 11, 24, 22, 0, 0, 0, time.UTC),  // 03:00
		seedNow,
		time.Date(2026, 11, 25, 18, 59, 59, 0, time.UTC), // 23:59:59
	} {
		local := now.In(loc)
		todayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
		for i, p := range orders {
			created := placementTime(p, i, now, loc)
			if created.After(now) {
				t.Errorf("now %v: order %d is placed at %v, in the future", now, i, created)
			}
			if p.daysAgo == 0 && created.Before(todayStart) {
				t.Errorf("now %v: order %d of today is placed at %v, before today starts", now, i, created)
			}
			steps := stepsTo(p)
			times := stepTimes(created, len(steps), now)
			prev := created
			for j, at := range times {
				if at.Before(prev) || at.After(now) {
					t.Errorf("now %v: order %d step %d at %v, want between %v and now", now, i, j, at, prev)
				}
				prev = at
			}
		}
	}
}

func TestRunNeedsABusinessZone(t *testing.T) {
	if _, err := Run(context.Background(), nil, Options{}); err == nil {
		t.Fatal("Run without a time zone succeeded")
	}
}

func scalar[T any](t *testing.T, pool *pgxpool.Pool, sql string, args ...any) T {
	t.Helper()
	var v T
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return v
}

func runSeed(t *testing.T, pool *pgxpool.Pool, email string) Result {
	t.Helper()
	res, err := Run(context.Background(), pool, Options{
		Email: email, Now: seedNow, Loc: tashkent(t), CommissionPercent: 10, LowStockThreshold: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestRunCreatesAConsistentDemoShop(t *testing.T) {
	pool := dbtest.NewPool(t, "seed")
	ctx := context.Background()
	email := fmt.Sprintf("seed-%d@example.com", time.Now().UnixNano())

	res := runSeed(t, pool, email)
	if res.AlreadySeeded || res.Products != 15 || res.Orders != 30 || res.Email != email || res.Password != DefaultPassword {
		t.Fatalf("result = %+v", res)
	}
	shop := scalar[int64](t, pool, `SELECT sh.id FROM shops sh JOIN sellers se ON se.id = sh.seller_id WHERE se.email = $1`, email)

	for status, want := range map[order.Status]int{
		order.StatusNew: 4, order.StatusAccepted: 2, order.StatusPacked: 2,
		order.StatusHandedToCourier: 3, order.StatusDelivered: 15, order.StatusCancelled: 4,
	} {
		if got := scalar[int](t, pool, `SELECT count(*) FROM orders WHERE shop_id = $1 AND status = $2::order_status`, shop, string(status)); got != want {
			t.Errorf("%d %s orders stored, want %d", got, status, want)
		}
		if res.ByStatus[status] != want {
			t.Errorf("result reports %d %s orders, want %d", res.ByStatus[status], status, want)
		}
	}
	for status, want := range map[string]int{"active": 13, "draft": 1, "archived": 1} {
		if got := scalar[int](t, pool, `SELECT count(*) FROM products WHERE shop_id = $1 AND status = $2::product_status`, shop, status); got != want {
			t.Errorf("%d %s products stored, want %d", got, status, want)
		}
	}

	t.Run("nothing is dated after now or before the 30 day window", func(t *testing.T) {
		if n := scalar[int](t, pool, `SELECT count(*) FROM orders WHERE shop_id = $1 AND (created_at > $2 OR status_changed_at > $2 OR created_at < $3)`,
			shop, seedNow, seedNow.AddDate(0, 0, -30)); n != 0 {
			t.Errorf("%d orders are dated outside the last 30 days", n)
		}
		if n := scalar[int](t, pool, `SELECT count(*) FROM order_status_history h JOIN orders o ON o.id = h.order_id WHERE o.shop_id = $1 AND h.created_at > $2`, shop, seedNow); n != 0 {
			t.Errorf("%d history rows are dated after now", n)
		}
		if n := scalar[int](t, pool, `SELECT count(*) FROM balance_entries WHERE shop_id = $1 AND created_at > $2`, shop, seedNow); n != 0 {
			t.Errorf("%d balance entries are dated after now", n)
		}
	})

	t.Run("order numbers follow time", func(t *testing.T) {
		if n := scalar[int](t, pool, `SELECT count(*) FROM orders a JOIN orders b ON a.number < b.number AND a.created_at > b.created_at WHERE a.shop_id = $1 AND b.shop_id = $1`, shop); n != 0 {
			t.Errorf("%d pairs of orders have numbers in the wrong order", n)
		}
	})

	t.Run("every order has a complete timeline that ends in its status", func(t *testing.T) {
		rows, err := pool.Query(ctx, `
			SELECT h.order_id, h.from_status::text, h.to_status::text, h.created_at, o.status::text, o.status_changed_at, o.created_at
			  FROM order_status_history h JOIN orders o ON o.id = h.order_id
			 WHERE o.shop_id = $1 ORDER BY h.order_id, h.created_at, h.id`, shop)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		type state struct {
			status string
			at     time.Time
		}
		last := map[int64]state{}
		final := map[int64]string{}
		for rows.Next() {
			var id int64
			var from *string
			var to, status string
			var at, changed, created time.Time
			if err := rows.Scan(&id, &from, &to, &at, &status, &changed, &created); err != nil {
				t.Fatal(err)
			}
			prev, seen := last[id]
			switch {
			case !seen && (from != nil || to != "new" || !at.Equal(created)):
				t.Errorf("order %d starts with %v to %s at %v, want nothing to new at its creation %v", id, from, to, at, created)
			case seen && (from == nil || *from != prev.status || !order.CanTransition(order.Status(prev.status), order.Status(to))):
				t.Errorf("order %d goes from %v to %s after %s, which the status machine forbids", id, from, to, prev.status)
			case seen && at.Before(prev.at):
				t.Errorf("order %d history goes back in time at %s", id, to)
			}
			last[id] = state{to, at}
			final[id] = status
			if !changed.Equal(at) && to == status {
				t.Errorf("order %d: status_changed_at %v differs from its last history row %v", id, changed, at)
			}
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		if len(last) != 30 {
			t.Fatalf("%d orders have a history, want 30", len(last))
		}
		for id, st := range last {
			if st.status != final[id] {
				t.Errorf("order %d ends its timeline in %s but is %s", id, st.status, final[id])
			}
		}
		if n := scalar[int](t, pool, `SELECT count(*) FROM orders WHERE shop_id = $1 AND status = 'cancelled' AND (cancel_reason IS NULL OR cancel_reason = '')`, shop); n != 0 {
			t.Errorf("%d cancelled orders have no reason", n)
		}
	})

	t.Run("reservations match the orders that hold them", func(t *testing.T) {
		rows, err := pool.Query(ctx, `
			SELECT p.sku, p.stock, p.reserved,
			       coalesce((SELECT sum(i.quantity) FROM order_items i JOIN orders o ON o.id = i.order_id
			                  WHERE i.product_id = p.id AND o.status IN ('accepted', 'packed', 'handed_to_courier')), 0)::int
			  FROM products p WHERE p.shop_id = $1`, shop)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var sku string
			var stock, reserved, held int
			if err := rows.Scan(&sku, &stock, &reserved, &held); err != nil {
				t.Fatal(err)
			}
			if reserved != held {
				t.Errorf("%s reserves %d units, the orders in progress hold %d", sku, reserved, held)
			}
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("the seller sees low stock and out of stock", func(t *testing.T) {
		want := map[string]int{}
		for _, p := range products {
			if p.status == "" {
				want[p.sku] = p.finalAvailable
			}
		}
		out, low := 0, 0
		for sku, available := range want {
			got := scalar[int](t, pool, `SELECT stock - reserved FROM products WHERE shop_id = $1 AND sku = $2`, shop, sku)
			if got != available {
				t.Errorf("%s has %d available, want %d", sku, got, available)
			}
			switch {
			case got == 0:
				out++
			case got <= 5:
				low++
			}
		}
		if out != 1 || low != 2 {
			t.Errorf("%d out of stock and %d low stock, want 1 and 2 for the demo", out, low)
		}
	})

	t.Run("balance is the delivered orders minus the commission, stamped with the delivery time", func(t *testing.T) {
		if n := scalar[int](t, pool, `SELECT count(*) FROM balance_entries WHERE shop_id = $1`, shop); n != 30 {
			t.Errorf("%d balance entries, want a sale and a commission for each of the 15 delivered orders", n)
		}
		if n := scalar[int](t, pool, `
			SELECT count(*) FROM orders o WHERE o.shop_id = $1 AND o.status = 'delivered' AND (
			    (SELECT sum(amount) FILTER (WHERE type = 'sale') FROM balance_entries b WHERE b.order_id = o.id) <> o.total
			 OR (SELECT sum(amount) FILTER (WHERE type = 'commission') FROM balance_entries b WHERE b.order_id = o.id) <> -(o.total * 10 / 100)
			 OR (SELECT count(*) FROM balance_entries b WHERE b.order_id = o.id AND b.created_at <> o.status_changed_at) > 0)`, shop); n != 0 {
			t.Errorf("%d delivered orders have balance entries that do not match them", n)
		}
		if n := scalar[int](t, pool, `SELECT count(*) FROM balance_entries b JOIN orders o ON o.id = b.order_id WHERE b.shop_id = $1 AND o.status <> 'delivered'`, shop); n != 0 {
			t.Errorf("%d balance entries belong to orders that were not delivered", n)
		}
		if available := scalar[int64](t, pool, `SELECT sum(amount) FROM balance_entries WHERE shop_id = $1`, shop); available <= 0 {
			t.Errorf("available balance = %d, want it positive", available)
		}
	})

	t.Run("the dashboard has a story to tell", func(t *testing.T) {
		loc := tashkent(t)
		local := seedNow.In(loc)
		day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
		if n := scalar[int](t, pool, `SELECT count(*) FROM orders WHERE shop_id = $1 AND created_at >= $2 AND created_at < $3`, shop, day, day.AddDate(0, 0, 1)); n != 5 {
			t.Errorf("%d orders were placed today, want 5", n)
		}
		if n := scalar[int](t, pool, `SELECT count(*) FROM orders WHERE shop_id = $1 AND status IN ('new', 'accepted', 'packed')`, shop); n != 8 {
			t.Errorf("%d orders wait for action, want 8", n)
		}
		if n := scalar[int](t, pool, `SELECT count(*) FROM orders WHERE shop_id = $1 AND status = 'delivered' AND status_changed_at >= $2`, shop, day.AddDate(0, 0, -6)); n != 6 {
			t.Errorf("%d orders were delivered in the last 7 days, want 6", n)
		}
		if n := scalar[int](t, pool, `SELECT count(*) FROM orders WHERE shop_id = $1 AND status = 'delivered' AND status_changed_at >= $2`, shop, day.AddDate(0, 0, -29)); n != 15 {
			t.Errorf("%d orders were delivered in the last 30 days, want all 15", n)
		}
	})

	t.Run("the demo login works", func(t *testing.T) {
		tokens, err := auth.NewTokenIssuer("test-secret", time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		svc, err := auth.NewService(auth.NewPgRepository(pool), tokens)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Login(ctx, auth.LoginRequest{Email: email, Password: DefaultPassword}); err != nil {
			t.Fatalf("login: %v", err)
		}
	})

	t.Run("running it again changes nothing", func(t *testing.T) {
		again := runSeed(t, pool, email)
		if !again.AlreadySeeded || again.Orders != 0 {
			t.Fatalf("second run = %+v, want AlreadySeeded and no writes", again)
		}
		if n := scalar[int](t, pool, `SELECT count(*) FROM orders WHERE shop_id = $1`, shop); n != 30 {
			t.Errorf("%d orders after a second run, want still 30", n)
		}
		if n := scalar[int](t, pool, `SELECT count(*) FROM products WHERE shop_id = $1`, shop); n != 15 {
			t.Errorf("%d products after a second run, want still 15", n)
		}
	})
}
