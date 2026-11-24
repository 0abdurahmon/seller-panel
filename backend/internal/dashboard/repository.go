package dashboard

import "context"

// Repository reads the dashboard numbers; every method is scoped by shopID.
type Repository interface {
	// Summary counts and sums the shop's orders inside the window. A shop with
	// no orders gets zeros and no top products.
	Summary(ctx context.Context, shopID int64, w Window) (Summary, error)
}
