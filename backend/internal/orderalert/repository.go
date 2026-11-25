package orderalert

import (
	"context"
	"errors"
)

// ErrNoChat means the shop's seller has not linked a Telegram chat.
var ErrNoChat = errors.New("no telegram chat linked to this shop")

// ErrNotFound means the shop has no such order.
var ErrNotFound = errors.New("order not found")

type Repository interface {
	// ChatForShop returns the Telegram chat of the shop's seller, or ErrNoChat.
	ChatForShop(ctx context.Context, shopID int64) (int64, error)
	// Lines returns the products of the shop's order, or ErrNotFound.
	Lines(ctx context.Context, shopID, orderID int64) ([]Line, error)
}
