package auth

import "context"

type Credentials struct {
	SellerID     int64
	ShopID       int64
	PasswordHash string
}

type Repository interface {
	CreateSellerWithShop(ctx context.Context, email, passwordHash, shopName string) (sellerID, shopID int64, err error)
	FindCredentials(ctx context.Context, email string) (Credentials, error)
}
