package shop

import "context"

// Profile is the set of fields that PUT /shop writes, already validated.
type Profile struct {
	Name          string
	LogoURL       string
	Description   string
	ContactPhone  string
	PickupAddress string
}

// Repository stores shop profiles. Every method is scoped by shopID.
type Repository interface {
	Get(ctx context.Context, shopID int64) (Shop, error)
	Update(ctx context.Context, shopID int64, p Profile) (Shop, error)
}
