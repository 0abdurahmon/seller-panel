package order

import (
	"context"
	"errors"
	"testing"
	"time"
)

const shopA int64 = 1

func newTestService(t *testing.T, repo *memRepository) *Service {
	t.Helper()
	svc, err := NewService(repo, 10)
	if err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC) }
	return svc
}

func order(id, shopID int64, status Status, total int64, items ...Item) Order {
	return Order{ID: id, ShopID: shopID, Number: "#1", Status: status, Items: items, Total: total,
		CreatedAt: time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)}
}

func TestStatusMachine(t *testing.T) {
	all := []Status{StatusNew, StatusAccepted, StatusPacked, StatusHandedToCourier, StatusDelivered, StatusCancelled}
	want := map[[2]Status]bool{
		{StatusNew, StatusAccepted}:              true,
		{StatusNew, StatusCancelled}:             true,
		{StatusAccepted, StatusPacked}:           true,
		{StatusAccepted, StatusCancelled}:        true,
		{StatusPacked, StatusHandedToCourier}:    true,
		{StatusPacked, StatusCancelled}:          true,
		{StatusHandedToCourier, StatusDelivered}: true,
	}
	for _, from := range all {
		for _, to := range all {
			if got := CanTransition(from, to); got != want[[2]Status{from, to}] {
				t.Errorf("CanTransition(%s, %s) = %v", from, to, got)
			}
		}
	}
}

// Every from→to pair goes through the service, so the rule is enforced there too.
func TestServiceEnforcesStatusMachine(t *testing.T) {
	all := []Status{StatusNew, StatusAccepted, StatusPacked, StatusHandedToCourier, StatusDelivered, StatusCancelled}
	for _, from := range all {
		for _, to := range all {
			repo := newMemRepository()
			repo.AddOrder(order(1, shopA, from, 0))
			_, err := newTestService(t, repo).Transition(context.Background(), shopA, 1,
				TransitionRequest{To: to, Reason: "reason"})
			if CanTransition(from, to) {
				if err != nil {
					t.Errorf("%s→%s: unexpected error %v", from, to, err)
				}
			} else if !errors.Is(err, ErrInvalidTransition) {
				t.Errorf("%s→%s: err = %v, want ErrInvalidTransition", from, to, err)
			}
		}
	}
}

func TestAcceptReservesStock(t *testing.T) {
	repo := newMemRepository()
	repo.AddProduct(7, shopA, 10, 2)
	repo.AddOrder(order(1, shopA, StatusNew, 3000, Item{ProductID: 7, Quantity: 3, Price: 1000}))

	o, err := newTestService(t, repo).Transition(context.Background(), shopA, 1, TransitionRequest{To: StatusAccepted})
	if err != nil {
		t.Fatal(err)
	}
	if o.Status != StatusAccepted {
		t.Fatalf("status = %s", o.Status)
	}
	if _, reserved := repo.Product(7); reserved != 5 {
		t.Fatalf("reserved = %d, want 5", reserved)
	}
}

func TestAcceptWithoutEnoughStockChangesNothing(t *testing.T) {
	repo := newMemRepository()
	repo.AddProduct(7, shopA, 5, 0)
	repo.AddProduct(8, shopA, 1, 0)
	// Product 7 can be reserved, product 8 cannot. The first reserve must be undone.
	repo.AddOrder(order(1, shopA, StatusNew, 0,
		Item{ProductID: 7, Quantity: 2}, Item{ProductID: 8, Quantity: 3}))

	_, err := newTestService(t, repo).Transition(context.Background(), shopA, 1, TransitionRequest{To: StatusAccepted})
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("err = %v, want ErrInsufficientStock", err)
	}
	if _, reserved := repo.Product(7); reserved != 0 {
		t.Fatalf("product 7 reserved = %d after failed accept, want 0", reserved)
	}
	o, _ := repo.Get(context.Background(), shopA, 1)
	if o.Status != StatusNew {
		t.Fatalf("status = %s after failed accept, want new", o.Status)
	}
}

func TestCancelReleasesOnlyWhenStockWasReserved(t *testing.T) {
	ctx := context.Background()

	repo := newMemRepository()
	repo.AddProduct(7, shopA, 10, 0)
	repo.AddOrder(order(1, shopA, StatusNew, 0, Item{ProductID: 7, Quantity: 4}))
	if _, err := newTestService(t, repo).Transition(ctx, shopA, 1,
		TransitionRequest{To: StatusCancelled, Reason: "buyer changed mind"}); err != nil {
		t.Fatal(err)
	}
	if _, reserved := repo.Product(7); reserved != 0 {
		t.Fatalf("new→cancelled changed reserved to %d", reserved)
	}

	repo = newMemRepository()
	repo.AddProduct(7, shopA, 10, 4)
	repo.AddOrder(order(2, shopA, StatusAccepted, 0, Item{ProductID: 7, Quantity: 4}))
	o, err := newTestService(t, repo).Transition(ctx, shopA, 2,
		TransitionRequest{To: StatusCancelled, Reason: " out of stock "})
	if err != nil {
		t.Fatal(err)
	}
	if _, reserved := repo.Product(7); reserved != 0 {
		t.Fatalf("accepted→cancelled reserved = %d, want 0", reserved)
	}
	if o.CancelReason == nil || *o.CancelReason != "out of stock" {
		t.Fatalf("cancel reason = %v, want trimmed text", o.CancelReason)
	}
}

func TestCancelNeedsReason(t *testing.T) {
	repo := newMemRepository()
	repo.AddOrder(order(1, shopA, StatusNew, 0))
	_, err := newTestService(t, repo).Transition(context.Background(), shopA, 1,
		TransitionRequest{To: StatusCancelled, Reason: "   "})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestDeliveredDeductsStockAndWritesBalance(t *testing.T) {
	repo := newMemRepository()
	repo.AddProduct(7, shopA, 10, 3)
	repo.AddOrder(order(1, shopA, StatusHandedToCourier, 3000, Item{ProductID: 7, Quantity: 3, Price: 1000}))

	if _, err := newTestService(t, repo).Transition(context.Background(), shopA, 1,
		TransitionRequest{To: StatusDelivered}); err != nil {
		t.Fatal(err)
	}
	stock, reserved := repo.Product(7)
	if stock != 7 || reserved != 0 {
		t.Fatalf("stock/reserved = %d/%d, want 7/0", stock, reserved)
	}
	entries := repo.Balance()
	if len(entries) != 2 || entries[0].Type != EntrySale || entries[0].Amount != 3000 ||
		entries[1].Type != EntryCommission || entries[1].Amount != -300 {
		t.Fatalf("balance entries = %+v", entries)
	}
}

func TestCommissionRoundsDownToWholeSom(t *testing.T) {
	repo := newMemRepository()
	repo.AddOrder(order(1, shopA, StatusHandedToCourier, 1005))
	if _, err := newTestService(t, repo).Transition(context.Background(), shopA, 1,
		TransitionRequest{To: StatusDelivered}); err != nil {
		t.Fatal(err)
	}
	if got := repo.Balance()[1].Amount; got != -100 {
		t.Fatalf("commission = %d, want -100 (10%% of 1005, rounded down)", got)
	}
}

func TestOrdersAreScopedToShop(t *testing.T) {
	ctx := context.Background()
	repo := newMemRepository()
	repo.AddProduct(7, shopA, 10, 0)
	repo.AddOrder(order(1, shopA, StatusNew, 0, Item{ProductID: 7, Quantity: 1}))
	svc := newTestService(t, repo)

	const shopB int64 = 2
	if _, err := svc.Get(ctx, shopB, 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get across shops: err = %v, want ErrNotFound", err)
	}
	if _, err := svc.Transition(ctx, shopB, 1, TransitionRequest{To: StatusAccepted}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("transition across shops: err = %v, want ErrNotFound", err)
	}
	if _, reserved := repo.Product(7); reserved != 0 {
		t.Fatalf("other shop's transition reserved stock: %d", reserved)
	}
	page, _ := svc.List(ctx, shopB, nil, 1, 20)
	if page.Total != 0 {
		t.Fatalf("other shop sees %d orders", page.Total)
	}
}

func TestListPaginatesNewestFirstAndFilters(t *testing.T) {
	repo := newMemRepository()
	base := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	for i := int64(1); i <= 5; i++ {
		o := order(i, shopA, StatusNew, 0)
		o.CreatedAt = base.Add(time.Duration(i) * time.Hour)
		if i == 5 {
			o.Status = StatusCancelled
		}
		repo.AddOrder(o)
	}
	svc := newTestService(t, repo)
	ctx := context.Background()

	page, err := svc.List(ctx, shopA, nil, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 5 || len(page.Items) != 2 || page.Items[0].ID != 3 {
		t.Fatalf("page 2 = %+v", page)
	}

	st := StatusCancelled
	filtered, _ := svc.List(ctx, shopA, &st, 1, 20)
	if filtered.Total != 1 || filtered.Items[0].ID != 5 {
		t.Fatalf("filtered = %+v", filtered)
	}
}

func TestListClampsPagingLikeProductLists(t *testing.T) {
	repo := newMemRepository()
	for i := int64(1); i <= 3; i++ {
		repo.AddOrder(order(i, shopA, StatusNew, 0))
	}
	svc := newTestService(t, repo)
	ctx := context.Background()

	page, err := svc.List(ctx, shopA, nil, 0, 0)
	if err != nil || page.Page != 1 || page.PageSize != defaultPageSize {
		t.Fatalf("page 0, size 0 = %+v, %v; want page 1, size %d", page, err, defaultPageSize)
	}
	page, err = svc.List(ctx, shopA, nil, -4, 500)
	if err != nil || page.Page != 1 || page.PageSize != maxPageSize {
		t.Fatalf("page -4, size 500 = %+v, %v; want page 1, size %d", page, err, maxPageSize)
	}
	empty, err := svc.List(ctx, shopA, nil, 99, 10)
	if err != nil || empty.Items == nil || len(empty.Items) != 0 || empty.Total != 3 {
		t.Fatalf("page past the end = %+v, %v; want an empty non-nil slice and total 3", empty, err)
	}
}

func TestListRejectsUnknownStatusAndHugePage(t *testing.T) {
	svc := newTestService(t, newMemRepository())
	bad := Status("shipped")
	cases := map[string]func() error{
		"unknown status": func() error { _, err := svc.List(context.Background(), shopA, &bad, 1, 20); return err },
		"huge page":      func() error { _, err := svc.List(context.Background(), shopA, nil, maxPage+1, 20); return err },
	}
	for name, call := range cases {
		if err := call(); !errors.Is(err, ErrValidation) {
			t.Errorf("%s: err = %v, want ErrValidation", name, err)
		}
	}
}

func TestTransitionRecordsHistory(t *testing.T) {
	repo := newMemRepository()
	repo.AddProduct(7, shopA, 10, 0)
	repo.AddOrder(order(1, shopA, StatusNew, 0, Item{ProductID: 7, Quantity: 1}))
	svc := newTestService(t, repo)
	ctx := context.Background()

	o, err := svc.Transition(ctx, shopA, 1, TransitionRequest{To: StatusAccepted})
	if err != nil {
		t.Fatal(err)
	}
	if len(o.History) != 1 || o.History[0].FromStatus == nil || *o.History[0].FromStatus != StatusNew || o.History[0].ToStatus != StatusAccepted {
		t.Fatalf("history after accept = %+v", o.History)
	}
	o, err = svc.Transition(ctx, shopA, 1, TransitionRequest{To: StatusCancelled, Reason: "no reply"})
	if err != nil {
		t.Fatal(err)
	}
	if len(o.History) != 2 || o.History[1].Reason == nil || *o.History[1].Reason != "no reply" {
		t.Fatalf("history after cancel = %+v", o.History)
	}
	got, _ := svc.Get(ctx, shopA, 1)
	if len(got.History) != 2 {
		t.Fatalf("Get returns %d history rows, want 2", len(got.History))
	}
}

func TestListsInOrderJSONAreNeverNull(t *testing.T) {
	repo := newMemRepository()
	repo.AddOrder(order(1, shopA, StatusNew, 0))
	o, err := newTestService(t, repo).Get(context.Background(), shopA, 1)
	if err != nil {
		t.Fatal(err)
	}
	if o.Items == nil || o.History == nil {
		t.Fatalf("items %v, history %v: both must be empty slices, not nil", o.Items, o.History)
	}
}

func TestNewServiceRejectsBadCommission(t *testing.T) {
	if _, err := NewService(newMemRepository(), 101); err == nil {
		t.Fatal("commission 101 accepted")
	}
}

func TestSimulateBuildsAnOrderFromAvailableProducts(t *testing.T) {
	ctx := context.Background()
	repo := newMemRepository()
	repo.AddPricedProduct(1, shopA, 10, 8, "Phone", 5000)
	repo.AddPricedProduct(2, shopA, 0, 0, "Sold out", 100)
	repo.AddPricedProduct(3, shopA, 20, 0, "Laptop", 9000)
	repo.AddPricedProduct(4, 2, 50, 0, "Other shop", 100)
	svc := newTestService(t, repo)
	svc.intn = func(int) int { return 0 }

	o, err := svc.Simulate(ctx, shopA)
	if err != nil {
		t.Fatal(err)
	}
	if o.Status != StatusNew || len(o.Items) != 1 || o.Items[0].ProductID != 1 || o.Items[0].Title != "Phone" ||
		o.Items[0].Price != 5000 || o.Items[0].Quantity != 1 || o.Total != 5000 {
		t.Fatalf("order = %+v", o)
	}
	if o.BuyerName != fakeBuyers[0].name || o.BuyerPhone == "" || o.DeliveryAddress == "" {
		t.Fatalf("buyer = %q %q %q", o.BuyerName, o.BuyerPhone, o.DeliveryAddress)
	}
	if len(o.History) != 1 || o.History[0].FromStatus != nil || o.History[0].ToStatus != StatusNew {
		t.Fatalf("history = %+v, want one row from nothing to new", o.History)
	}
	if _, reserved := repo.Product(1); reserved != 8 {
		t.Fatalf("a simulated order must not reserve stock, reserved = %d", reserved)
	}
}

func TestSimulateNeedsAvailableProducts(t *testing.T) {
	ctx := context.Background()
	repo := newMemRepository()
	if _, err := newTestService(t, repo).Simulate(ctx, shopA); !errors.Is(err, ErrNoProducts) {
		t.Fatalf("no products: err = %v, want ErrNoProducts", err)
	}
	repo.AddPricedProduct(1, shopA, 3, 3, "Reserved", 100)
	repo.AddPricedProduct(2, 2, 10, 0, "Other shop", 100)
	if _, err := newTestService(t, repo).Simulate(ctx, shopA); !errors.Is(err, ErrNoProducts) {
		t.Fatalf("nothing available in this shop: err = %v, want ErrNoProducts", err)
	}
}

func TestSimulateStaysWithinWhatTheShopHas(t *testing.T) {
	repo := newMemRepository()
	repo.AddPricedProduct(1, shopA, 10, 8, "Two left", 100)
	repo.AddPricedProduct(2, shopA, 20, 0, "Plenty", 200)
	repo.AddPricedProduct(3, shopA, 1, 0, "One left", 300)
	repo.AddPricedProduct(4, shopA, 5, 5, "Taken", 400)
	repo.AddPricedProduct(5, 2, 50, 0, "Other shop", 500)
	svc := newTestService(t, repo)
	available := map[int64]int{1: 2, 2: 20, 3: 1}

	for i := 0; i < 300; i++ {
		o, err := svc.Simulate(context.Background(), shopA)
		if err != nil {
			t.Fatal(err)
		}
		if len(o.Items) < 1 || len(o.Items) > 3 {
			t.Fatalf("%d lines, want 1 to 3", len(o.Items))
		}
		seen := map[int64]bool{}
		var total int64
		for _, it := range o.Items {
			max, ok := available[it.ProductID]
			if !ok {
				t.Fatalf("used product %d, which this shop cannot sell", it.ProductID)
			}
			if seen[it.ProductID] {
				t.Fatalf("product %d appears twice", it.ProductID)
			}
			seen[it.ProductID] = true
			if it.Quantity < 1 || it.Quantity > max || it.Quantity > 3 {
				t.Fatalf("quantity %d for a product with %d available", it.Quantity, max)
			}
			total += it.Price * int64(it.Quantity)
		}
		if o.Total != total {
			t.Fatalf("total = %d, want %d", o.Total, total)
		}
	}
}

func TestSimulateCanUseEveryProductAndQuantity(t *testing.T) {
	repo := newMemRepository()
	repo.AddPricedProduct(1, shopA, 9, 0, "A", 100)
	repo.AddPricedProduct(2, shopA, 9, 0, "B", 100)
	repo.AddPricedProduct(3, shopA, 9, 0, "C", 100)
	svc := newTestService(t, repo)
	svc.intn = func(n int) int { return n - 1 }

	o, err := svc.Simulate(context.Background(), shopA)
	if err != nil {
		t.Fatal(err)
	}
	if len(o.Items) != 3 {
		t.Fatalf("%d lines, want 3 when the randomness always picks the maximum", len(o.Items))
	}
	for _, it := range o.Items {
		if it.Quantity != 3 {
			t.Errorf("product %d quantity %d, want the cap of 3", it.ProductID, it.Quantity)
		}
	}
}
