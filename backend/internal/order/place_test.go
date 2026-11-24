package order

import (
	"context"
	"errors"
	"testing"
	"time"
)

func validNewOrder() NewOrder {
	return NewOrder{
		BuyerName:       " Aziz Karimov ",
		BuyerPhone:      "+998901112233",
		DeliveryAddress: "Tashkent, Chilonzor 7",
		Items:           []NewItem{{ProductID: 7, Title: "Phone", Price: 1500, Quantity: 2}},
	}
}

func TestPlaceStoresANewOrderAtTheClockTime(t *testing.T) {
	ctx := context.Background()
	repo := newMemRepository()
	repo.AddPricedProduct(7, shopA, 10, 0, "Phone", 1500)
	svc := newTestService(t, repo)
	at := time.Date(2026, 10, 30, 9, 30, 0, 0, time.UTC)

	got, err := svc.WithClock(func() time.Time { return at }).Place(ctx, shopA, validNewOrder())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusNew || got.Total != 3000 || got.BuyerName != "Aziz Karimov" {
		t.Fatalf("order = %+v, want a new order of 3000 for a trimmed buyer name", got)
	}
	if !got.CreatedAt.Equal(at) || !got.StatusChangedAt.Equal(at) {
		t.Fatalf("times = %v / %v, want %v", got.CreatedAt, got.StatusChangedAt, at)
	}
	if len(got.History) != 1 || got.History[0].FromStatus != nil || got.History[0].ToStatus != StatusNew || !got.History[0].CreatedAt.Equal(at) {
		t.Fatalf("history = %+v, want one row from nothing to new at %v", got.History, at)
	}

	again, err := svc.Place(ctx, shopA, validNewOrder())
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC); !again.CreatedAt.Equal(want) {
		t.Fatalf("the original service moved to %v, want its own clock %v", again.CreatedAt, want)
	}
}

func TestPlaceValidation(t *testing.T) {
	cases := []struct {
		name   string
		change func(n *NewOrder)
	}{
		{"blank buyer name", func(n *NewOrder) { n.BuyerName = "  " }},
		{"blank phone", func(n *NewOrder) { n.BuyerPhone = "" }},
		{"blank address", func(n *NewOrder) { n.DeliveryAddress = " " }},
		{"no items", func(n *NewOrder) { n.Items = nil }},
		{"zero quantity", func(n *NewOrder) { n.Items[0].Quantity = 0 }},
		{"negative quantity", func(n *NewOrder) { n.Items[0].Quantity = -1 }},
		{"zero price", func(n *NewOrder) { n.Items[0].Price = 0 }},
		{"blank title", func(n *NewOrder) { n.Items[0].Title = " " }},
		{"no product", func(n *NewOrder) { n.Items[0].ProductID = 0 }},
		{"same product twice", func(n *NewOrder) { n.Items = append(n.Items, n.Items[0]) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			repo := newMemRepository()
			repo.AddProduct(7, shopA, 10, 0)
			svc := newTestService(t, repo)
			n := validNewOrder()
			c.change(&n)
			if _, err := svc.Place(ctx, shopA, n); !errors.Is(err, ErrValidation) {
				t.Fatalf("err = %v, want ErrValidation", err)
			}
			if page, err := svc.List(ctx, shopA, nil, 1, 20); err != nil || page.Total != 0 {
				t.Fatalf("list after a rejected order = %+v, %v, want nothing stored", page, err)
			}
		})
	}
}

func TestPlaceRejectsAnotherShopsProduct(t *testing.T) {
	ctx := context.Background()
	repo := newMemRepository()
	repo.AddProduct(7, shopA+1, 10, 0)
	svc := newTestService(t, repo)
	if _, err := svc.Place(ctx, shopA, validNewOrder()); !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("err = %v, want ErrProductNotFound", err)
	}
	if page, err := svc.List(ctx, shopA, nil, 1, 20); err != nil || page.Total != 0 {
		t.Fatalf("list = %+v, %v, want nothing stored", page, err)
	}
}

func TestTransitionUsesTheClockAndStampsTheBalanceEntries(t *testing.T) {
	repo := newMemRepository()
	repo.AddProduct(7, shopA, 10, 3)
	repo.AddOrder(order(1, shopA, StatusHandedToCourier, 3000, Item{ProductID: 7, Quantity: 3, Price: 1000}))
	at := time.Date(2026, 11, 2, 15, 45, 0, 0, time.UTC)

	got, err := newTestService(t, repo).WithClock(func() time.Time { return at }).
		Transition(context.Background(), shopA, 1, TransitionRequest{To: StatusDelivered})
	if err != nil {
		t.Fatal(err)
	}
	if !got.StatusChangedAt.Equal(at) {
		t.Fatalf("status_changed_at = %v, want %v", got.StatusChangedAt, at)
	}
	entries := repo.Balance()
	if len(entries) != 2 {
		t.Fatalf("balance entries = %+v, want a sale and a commission", entries)
	}
	for _, e := range entries {
		if !e.CreatedAt.Equal(at) {
			t.Errorf("entry %+v is stamped %v, want the delivery time %v", e, e.CreatedAt, at)
		}
	}
}
