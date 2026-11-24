package order

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// WithClock returns a copy of the service that reads the time from now, so a seed can place orders in the past.
func (s *Service) WithClock(now func() time.Time) *Service {
	c := *s
	c.now = now
	return &c
}

// Place stores a new order as a buyer would have sent it. No route calls it: orders come from buyers outside the panel.
func (s *Service) Place(ctx context.Context, shopID int64, n NewOrder) (Order, error) {
	n.BuyerName = strings.TrimSpace(n.BuyerName)
	n.BuyerPhone = strings.TrimSpace(n.BuyerPhone)
	n.DeliveryAddress = strings.TrimSpace(n.DeliveryAddress)
	if n.BuyerName == "" || n.BuyerPhone == "" || n.DeliveryAddress == "" {
		return Order{}, fmt.Errorf("%w: buyer name, phone and address are required", ErrValidation)
	}
	if len(n.Items) == 0 {
		return Order{}, fmt.Errorf("%w: an order needs at least one item", ErrValidation)
	}
	seen := make(map[int64]bool, len(n.Items))
	for _, it := range n.Items {
		switch {
		case it.ProductID <= 0 || strings.TrimSpace(it.Title) == "":
			return Order{}, fmt.Errorf("%w: every item needs a product and a title", ErrValidation)
		case it.Price <= 0 || it.Quantity <= 0:
			return Order{}, fmt.Errorf("%w: price and quantity must be positive", ErrValidation)
		case seen[it.ProductID]:
			return Order{}, fmt.Errorf("%w: product %d is in the order twice", ErrValidation, it.ProductID)
		}
		seen[it.ProductID] = true
	}
	o, err := s.repo.Create(ctx, shopID, n, s.now())
	if err != nil {
		return Order{}, err
	}
	return withLists(o), nil
}
