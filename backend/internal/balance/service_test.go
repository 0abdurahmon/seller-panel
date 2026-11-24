package balance

import (
	"context"
	"errors"
	"testing"
	"time"
)

func ptr(n int64) *int64 { return &n }

var t0 = time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)

// deliver records an order the way the order feature does: sale +total, commission -10%.
func deliver(r *memRepository, shopID, orderID, total int64, at time.Time) {
	r.add(shopID, ptr(orderID), EntrySale, total, at)
	r.add(shopID, ptr(orderID), EntryCommission, -(total * 10 / 100), at)
}

func TestSummary(t *testing.T) {
	repo := newMemRepository()
	svc := NewService(repo)
	ctx := context.Background()

	deliver(repo, 1, 100, 250000, t0)
	deliver(repo, 1, 101, 12345, t0.Add(time.Hour))
	deliver(repo, 2, 200, 999000, t0) // another shop
	repo.add(1, nil, EntryPayout, -50000, t0.Add(2*time.Hour))

	cases := []struct {
		name   string
		shopID int64
		want   Summary
	}{
		// 250000 - 25000 + 12345 - 1234 - 50000 (payout counts only in available).
		{"shop with sales, commission and a payout", 1, Summary{Available: 186111, Earned: 262345, Commission: -26234}},
		{"other shop sees only its own", 2, Summary{Available: 899100, Earned: 999000, Commission: -99900}},
		{"shop with no entries gets zeros", 3, Summary{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := svc.Summary(ctx, c.shopID)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Fatalf("Summary = %+v, want %+v", got, c.want)
			}
		})
	}
}

// The DATA_MODEL example: total 12345 gives commission 1234 and net 11111, nothing lost.
func TestSummaryKeepsEverySom(t *testing.T) {
	repo := newMemRepository()
	deliver(repo, 1, 100, 12345, t0)
	got, err := NewService(repo).Summary(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Earned != 12345 || got.Commission != -1234 || got.Available != 11111 {
		t.Fatalf("Summary = %+v, want earned 12345 commission -1234 available 11111", got)
	}
}

func TestEntriesNewestFirstAndScopedByShop(t *testing.T) {
	repo := newMemRepository()
	svc := NewService(repo)
	deliver(repo, 1, 100, 1000, t0)
	deliver(repo, 2, 200, 5000, t0.Add(time.Minute)) // another shop
	payout := repo.add(1, nil, EntryPayout, -500, t0.Add(time.Hour))

	page, err := svc.Entries(context.Background(), 1, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 3 || len(page.Items) != 3 {
		t.Fatalf("total %d, items %d; want 3 and 3", page.Total, len(page.Items))
	}
	if page.Items[0].ID != payout || page.Items[0].OrderID != nil {
		t.Fatalf("first item = %+v, want the payout (newest, no order)", page.Items[0])
	}
	// Same timestamp: the later id comes first, so the order is stable.
	if page.Items[1].Type != EntryCommission || page.Items[2].Type != EntrySale {
		t.Fatalf("items 2-3 = %s, %s; want commission then sale", page.Items[1].Type, page.Items[2].Type)
	}
	for _, e := range page.Items {
		if e.OrderID != nil && *e.OrderID == 200 {
			t.Fatal("another shop's entry is visible")
		}
	}
}

func TestEntriesPagination(t *testing.T) {
	repo := newMemRepository()
	svc := NewService(repo)
	for i := range 25 {
		repo.add(1, ptr(int64(i)), EntrySale, 100, t0.Add(time.Duration(i)*time.Minute))
	}
	ctx := context.Background()

	cases := []struct {
		name               string
		page, pageSize     int
		wantPage, wantSize int
		wantItems          int
	}{
		{"defaults", 0, 0, 1, 20, 20},
		{"second page", 2, 20, 2, 20, 5},
		{"negative values fall back", -3, -1, 1, 20, 20},
		{"page size above max is clamped", 1, 500, 1, 100, 25},
		{"page past the end is empty", 9, 20, 9, 20, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := svc.Entries(ctx, 1, c.page, c.pageSize)
			if err != nil {
				t.Fatal(err)
			}
			if got.Page != c.wantPage || got.PageSize != c.wantSize || len(got.Items) != c.wantItems || got.Total != 25 {
				t.Fatalf("got page %d size %d items %d total %d; want %d %d %d 25",
					got.Page, got.PageSize, len(got.Items), got.Total, c.wantPage, c.wantSize, c.wantItems)
			}
			if got.Items == nil {
				t.Fatal("items is nil, want an empty slice so JSON shows []")
			}
		})
	}
}

func TestEntriesRejectsHugePage(t *testing.T) {
	svc := NewService(newMemRepository())
	if _, err := svc.Entries(context.Background(), 1, maxPage+1, 20); !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation (a huge page would overflow the offset)", err)
	}
	if _, err := svc.Entries(context.Background(), 1, maxPage, 100); err != nil {
		t.Fatalf("the largest allowed page failed: %v", err)
	}
}

func TestEntriesEmptyShop(t *testing.T) {
	got, err := NewService(newMemRepository()).Entries(context.Background(), 7, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if got.Items == nil || len(got.Items) != 0 || got.Total != 0 {
		t.Fatalf("got %+v, want empty items [] and total 0", got)
	}
}
