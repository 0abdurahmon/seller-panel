package dashboard

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func tashkent(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Tashkent")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func utc(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// now is 01:30 on 25 November in Tashkent, still 24 November in UTC.
var now = utc("2026-11-24T20:30:00Z")

func newTestService(t *testing.T, repo Repository) *Service {
	t.Helper()
	svc, err := NewService(repo, tashkent(t))
	if err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return now }
	return svc
}

func TestNewServiceNeedsATimeZone(t *testing.T) {
	if _, err := NewService(newMemRepository(), nil); err == nil {
		t.Fatal("a service without a time zone was accepted")
	}
}

func TestWindowFollowsCalendarDaysInTheBusinessZone(t *testing.T) {
	svc := newTestService(t, newMemRepository())
	cases := []struct {
		name string
		at   time.Time
		want Window
	}{
		{
			"after midnight in Tashkent, before it in UTC",
			now,
			Window{
				TodayStart: utc("2026-11-24T19:00:00Z"),
				TodayEnd:   utc("2026-11-25T19:00:00Z"),
				From7D:     utc("2026-11-18T19:00:00Z"),
				From30D:    utc("2026-10-26T19:00:00Z"),
			},
		},
		{
			"one second before midnight in Tashkent",
			utc("2026-11-25T18:59:59Z"),
			Window{
				TodayStart: utc("2026-11-24T19:00:00Z"),
				TodayEnd:   utc("2026-11-25T19:00:00Z"),
				From7D:     utc("2026-11-18T19:00:00Z"),
				From30D:    utc("2026-10-26T19:00:00Z"),
			},
		},
		{
			"exactly midnight in Tashkent starts a new day",
			utc("2026-11-25T19:00:00Z"),
			Window{
				TodayStart: utc("2026-11-25T19:00:00Z"),
				TodayEnd:   utc("2026-11-26T19:00:00Z"),
				From7D:     utc("2026-11-19T19:00:00Z"),
				From30D:    utc("2026-10-27T19:00:00Z"),
			},
		},
		{
			"midday",
			utc("2026-03-10T07:00:00Z"),
			Window{
				TodayStart: utc("2026-03-09T19:00:00Z"),
				TodayEnd:   utc("2026-03-10T19:00:00Z"),
				From7D:     utc("2026-03-03T19:00:00Z"),
				From30D:    utc("2026-02-08T19:00:00Z"),
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := svc.window(c.at); got != c.want {
				t.Fatalf("window(%s) = %+v, want %+v", c.at, got, c.want)
			}
		})
	}
}

func TestSummaryCountsOnlyWhatBelongsToTheShop(t *testing.T) {
	repo := newMemRepository()
	repo.titles[10], repo.titles[20] = "Phone", "Case"

	todayStart := utc("2026-11-24T19:00:00Z")
	d7 := utc("2026-11-18T19:00:00Z")
	d30 := utc("2026-10-26T19:00:00Z")
	sec := time.Second

	for _, o := range []memOrder{
		{shopID: 1, status: "new", total: 100, createdAt: todayStart},
		{shopID: 1, status: "new", total: 100, createdAt: todayStart.Add(-sec)},
		{shopID: 1, status: "accepted", total: 100, createdAt: now},
		{shopID: 1, status: "packed", total: 100, createdAt: d30},
		{shopID: 1, status: "handed_to_courier", total: 100, createdAt: now},
		{shopID: 1, status: "cancelled", total: 100, createdAt: now},
		{shopID: 1, status: "delivered", total: 1000, createdAt: d30, changedAt: d7, items: []memItem{{10, 2, 500}}},
		{shopID: 1, status: "delivered", total: 400, createdAt: d30, changedAt: d7.Add(-sec), items: []memItem{{20, 4, 100}}},
		{shopID: 1, status: "delivered", total: 50, createdAt: d30, changedAt: d30.Add(-sec), items: []memItem{{20, 1, 50}}},
		{shopID: 2, status: "new", total: 9, createdAt: now},
		{shopID: 2, status: "delivered", total: 9_000_000, createdAt: now, changedAt: now, items: []memItem{{10, 90, 100_000}}},
	} {
		repo.add(o)
	}

	cases := []struct {
		name   string
		shopID int64
		want   Summary
	}{
		{
			"windows are half open and cancelled or shipped orders are not waiting",
			1,
			Summary{
				TodayOrders:      4,
				WaitingForAction: 4,
				Revenue7D:        1000,
				Revenue30D:       1400,
				TopProducts: []TopProduct{
					{ProductID: 20, Title: "Case", QuantitySold: 4, Revenue: 400},
					{ProductID: 10, Title: "Phone", QuantitySold: 2, Revenue: 1000},
				},
			},
		},
		{
			"another shop sees only its own orders",
			2,
			Summary{
				TodayOrders:      2,
				WaitingForAction: 1,
				Revenue7D:        9_000_000,
				Revenue30D:       9_000_000,
				TopProducts:      []TopProduct{{ProductID: 10, Title: "Phone", QuantitySold: 90, Revenue: 9_000_000}},
			},
		},
		{
			"a shop without orders gets zeros and an empty list",
			3,
			Summary{TopProducts: []TopProduct{}},
		},
	}
	svc := newTestService(t, repo)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := svc.Summary(context.Background(), c.shopID)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("Summary = %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestSummaryAsksTheRepositoryForTheCurrentWindow(t *testing.T) {
	repo := newMemRepository()
	svc := newTestService(t, repo)
	if _, err := svc.Summary(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	want := Window{
		TodayStart: utc("2026-11-24T19:00:00Z"),
		TodayEnd:   utc("2026-11-25T19:00:00Z"),
		From7D:     utc("2026-11-18T19:00:00Z"),
		From30D:    utc("2026-10-26T19:00:00Z"),
	}
	if repo.last != want {
		t.Fatalf("window = %+v, want %+v", repo.last, want)
	}
}

func TestTopProductsAreRankedAndCutAtFive(t *testing.T) {
	repo := newMemRepository()
	delivered := utc("2026-11-20T10:00:00Z")
	// Product ids 1..7 with these quantities and unit prices. Products 3 and 4 tie on
	// quantity, product 4 has the higher revenue; products 5 and 6 tie on both.
	for id, line := range map[int64][2]int64{1: {1, 10}, 2: {9, 10}, 3: {5, 10}, 4: {5, 20}, 5: {3, 10}, 6: {3, 10}, 7: {2, 10}} {
		repo.titles[id] = "P"
		repo.add(memOrder{shopID: 1, status: "delivered", total: line[0] * line[1], createdAt: delivered, changedAt: delivered,
			items: []memItem{{id, line[0], line[1]}}})
	}
	got, err := newTestService(t, repo).Summary(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for _, p := range got.TopProducts {
		ids = append(ids, p.ProductID)
	}
	if want := []int64{2, 4, 3, 5, 6}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("top products = %v, want %v", ids, want)
	}
}

func TestSummaryReturnsRepositoryErrors(t *testing.T) {
	repo := newMemRepository()
	boom := errors.New("boom")
	repo.err = boom
	if _, err := newTestService(t, repo).Summary(context.Background(), 1); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}
}
