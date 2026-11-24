// Package seed fills an empty database with a demo shop, products and orders for development.
// Everything goes through the service code, so stock, reservations, history and balance stay consistent.
package seed

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/catalog"
	"github.com/0abdurahmon/seller-panel/backend/internal/order"
	"github.com/0abdurahmon/seller-panel/backend/internal/product"
	"github.com/0abdurahmon/seller-panel/backend/internal/shop"
	"github.com/0abdurahmon/seller-panel/backend/internal/stock"
)

// stepSpan is how long each status change takes in a past order, unless that would end in the future.
const stepSpan = 6 * time.Hour

// Options configure one seed run. Zero values fall back to the demo defaults.
type Options struct {
	Email    string
	Password string
	ShopName string
	// Now is the moment the data is relative to; the newest record is never later than Now.
	Now time.Time
	// Loc is the business time zone that defines the days orders are placed on.
	Loc               *time.Location
	CommissionPercent int
	LowStockThreshold int
}

// Result tells what a run did.
type Result struct {
	// AlreadySeeded is true when the demo seller exists; nothing was written.
	AlreadySeeded bool
	Email         string
	Password      string
	Products      int
	Orders        int
	ByStatus      map[order.Status]int
}

type services struct {
	auth    *auth.Service
	tokens  *auth.TokenIssuer
	shop    *shop.Service
	catalog *catalog.Service
	product *product.Service
	stock   *stock.Service
	order   *order.Service
}

// Run creates the demo data, or does nothing when the demo seller already exists.
func Run(ctx context.Context, pool *pgxpool.Pool, o Options) (Result, error) {
	o = withDefaults(o)
	if o.Loc == nil {
		return Result{}, errors.New("seed: the business time zone is required")
	}
	svc, err := newServices(pool, o)
	if err != nil {
		return Result{}, err
	}

	reg, err := svc.auth.Register(ctx, auth.RegisterRequest{Email: o.Email, Password: o.Password, ShopName: o.ShopName})
	if errors.Is(err, auth.ErrEmailTaken) {
		return Result{AlreadySeeded: true, Email: o.Email}, nil
	}
	if err != nil {
		return Result{}, fmt.Errorf("register the demo seller: %w", err)
	}
	claims, err := svc.tokens.Parse(reg.Token)
	if err != nil {
		return Result{}, fmt.Errorf("read the demo shop id: %w", err)
	}
	shopID := claims.ShopID

	if _, err := svc.shop.Update(ctx, shopID, shop.UpdateRequest{
		Name:          o.ShopName,
		Description:   "Phones, laptops, clothing and home goods with delivery across Uzbekistan.",
		ContactPhone:  "+998712000000",
		PickupAddress: "Tashkent, Amir Temur 15",
	}); err != nil {
		return Result{}, fmt.Errorf("fill the shop profile: %w", err)
	}

	items, err := createProducts(ctx, svc, shopID)
	if err != nil {
		return Result{}, err
	}
	byStatus, err := placeOrders(ctx, svc, shopID, items, o)
	if err != nil {
		return Result{}, err
	}
	if err := setFinalStock(ctx, svc, shopID, items); err != nil {
		return Result{}, err
	}

	total := 0
	for _, n := range byStatus {
		total += n
	}
	return Result{Email: o.Email, Password: o.Password, Products: len(products), Orders: total, ByStatus: byStatus}, nil
}

func withDefaults(o Options) Options {
	if o.Email == "" {
		o.Email = DefaultEmail
	}
	if o.Password == "" {
		o.Password = DefaultPassword
	}
	if o.ShopName == "" {
		o.ShopName = defaultShopName
	}
	if o.Now.IsZero() {
		o.Now = time.Now()
	}
	return o
}

func newServices(pool *pgxpool.Pool, o Options) (*services, error) {
	// The token is only read back to learn the shop id, so any secret will do.
	tokens, err := auth.NewTokenIssuer("seed-only-secret", time.Minute)
	if err != nil {
		return nil, err
	}
	authSvc, err := auth.NewService(auth.NewPgRepository(pool), tokens)
	if err != nil {
		return nil, err
	}
	stockSvc, err := stock.NewService(stock.NewPgRepository(pool), o.LowStockThreshold)
	if err != nil {
		return nil, err
	}
	orderSvc, err := order.NewService(order.NewPgRepository(pool), o.CommissionPercent)
	if err != nil {
		return nil, err
	}
	return &services{
		auth:    authSvc,
		tokens:  tokens,
		shop:    shop.NewService(shop.NewPgRepository(pool)),
		catalog: catalog.NewService(catalog.NewPgRepository(pool)),
		product: product.NewService(product.NewPgRepository(pool), noUploads{}),
		stock:   stockSvc,
		order:   orderSvc,
	}, nil
}

// noUploads is the photo check for the seed: demo products have no photos, so no URL is ever accepted.
type noUploads struct{}

func (noUploads) Owns(context.Context, int64, string) (bool, error) { return false, nil }

func createProducts(ctx context.Context, svc *services, shopID int64) (map[string]product.Product, error) {
	cats, err := svc.catalog.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	categoryID := map[string]int64{}
	for _, c := range cats.Items {
		if c.ParentID != nil {
			categoryID[c.Name] = c.ID
		}
	}

	created := make(map[string]product.Product, len(products))
	for _, spec := range products {
		cat, ok := categoryID[spec.category]
		if !ok {
			return nil, fmt.Errorf("category %q is not in the database", spec.category)
		}
		status, stockLevel := product.StatusActive, initialStock
		if spec.status == string(product.StatusDraft) {
			status = product.StatusDraft
		}
		if spec.stock != 0 {
			stockLevel = spec.stock
		}
		p, err := svc.product.Create(ctx, shopID, product.CreateInput{
			CategoryID: cat, Title: spec.title, Description: spec.description,
			Price: spec.price, SKU: spec.sku, Stock: stockLevel, Status: status,
		})
		if err != nil {
			return nil, fmt.Errorf("create product %s: %w", spec.sku, err)
		}
		if spec.status == string(product.StatusArchived) {
			if p, err = svc.product.Archive(ctx, shopID, p.ID); err != nil {
				return nil, fmt.Errorf("archive product %s: %w", spec.sku, err)
			}
		}
		created[spec.sku] = p
	}
	return created, nil
}

// placeOrders places the planned orders from the oldest to the newest, so order numbers follow time,
// and moves each one through its statuses with the clock set to the moment of every change.
func placeOrders(ctx context.Context, svc *services, shopID int64, items map[string]product.Product, o Options) (map[order.Status]int, error) {
	type placed struct {
		plan      plannedOrder
		createdAt time.Time
	}
	plan := make([]placed, len(orders))
	for i, p := range orders {
		plan[i] = placed{plan: p, createdAt: placementTime(p, i, o.Now, o.Loc)}
	}
	sort.SliceStable(plan, func(i, j int) bool { return plan[i].createdAt.Before(plan[j].createdAt) })

	byStatus := map[order.Status]int{}
	for _, pl := range plan {
		p := pl.plan
		newOrder := order.NewOrder{
			BuyerName:       buyers[p.buyer].name,
			BuyerPhone:      buyers[p.buyer].phone,
			DeliveryAddress: buyers[p.buyer].address,
		}
		for _, l := range p.lines {
			prod, ok := items[l.sku]
			if !ok {
				return nil, fmt.Errorf("order uses the unknown product %q", l.sku)
			}
			newOrder.Items = append(newOrder.Items, order.NewItem{ProductID: prod.ID, Title: prod.Title, Price: prod.Price, Quantity: l.qty})
		}

		placedOrder, err := svc.order.WithClock(fixed(pl.createdAt)).Place(ctx, shopID, newOrder)
		if err != nil {
			return nil, fmt.Errorf("place an order of %s: %w", buyers[p.buyer].name, err)
		}
		steps := stepsTo(p)
		times := stepTimes(pl.createdAt, len(steps), o.Now)
		for i, to := range steps {
			at := times[i]
			req := order.TransitionRequest{To: to}
			if to == order.StatusCancelled {
				req.Reason = p.reason
			}
			if _, err := svc.order.WithClock(fixed(at)).Transition(ctx, shopID, placedOrder.ID, req); err != nil {
				return nil, fmt.Errorf("move order %s to %s: %w", placedOrder.Number, to, err)
			}
		}
		byStatus[p.final]++
	}
	return byStatus, nil
}

// setFinalStock sets what the seller sees at the end, on top of what the orders reserved, so the product
// list shows low stock and out of stock.
func setFinalStock(ctx context.Context, svc *services, shopID int64, items map[string]product.Product) error {
	for _, spec := range products {
		if spec.status != "" {
			continue
		}
		current, err := svc.product.Get(ctx, shopID, items[spec.sku].ID)
		if err != nil {
			return fmt.Errorf("read product %s: %w", spec.sku, err)
		}
		if _, err := svc.stock.SetStock(ctx, shopID, current.ID, current.Reserved+spec.finalAvailable, current.Stock); err != nil {
			return fmt.Errorf("set the stock of %s: %w", spec.sku, err)
		}
	}
	return nil
}

// stepsTo lists the status changes an order goes through after it is placed.
func stepsTo(p plannedOrder) []order.Status {
	chain := []order.Status{order.StatusAccepted, order.StatusPacked, order.StatusHandedToCourier, order.StatusDelivered}
	last := p.final
	if p.final == order.StatusCancelled {
		last = p.cancelFrom
	}
	var steps []order.Status
	if last != order.StatusNew {
		for _, s := range chain {
			steps = append(steps, s)
			if s == last {
				break
			}
		}
	}
	if p.final == order.StatusCancelled {
		steps = append(steps, order.StatusCancelled)
	}
	return steps
}

// placementTime is the moment a planned order was placed. index spreads the minutes so orders on the
// same hour do not share a timestamp.
func placementTime(p plannedOrder, index int, now time.Time, loc *time.Location) time.Time {
	local := now.In(loc)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	if p.daysAgo == 0 {
		return today.Add(time.Duration(float64(now.Sub(today)) * p.today)).UTC()
	}
	day := today.AddDate(0, 0, -p.daysAgo)
	return day.Add(time.Duration(p.hour)*time.Hour + time.Duration(index*13%60)*time.Minute).UTC()
}

func fixed(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

// stepTimes spreads n status changes between the placement and the end of the order. The end is n steps
// later, but never after a minute before now, and never before the placement itself.
func stepTimes(createdAt time.Time, n int, now time.Time) []time.Time {
	end := createdAt.Add(time.Duration(n) * stepSpan)
	if limit := now.Add(-time.Minute); end.After(limit) {
		end = limit
	}
	if end.Before(createdAt) {
		end = createdAt
	}
	times := make([]time.Time, n)
	for i := range times {
		times[i] = createdAt.Add(end.Sub(createdAt) * time.Duration(i+1) / time.Duration(n))
	}
	return times
}
