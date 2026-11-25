package orderalert

import (
	"context"
	"errors"
)

// ErrNoChat means the shop's seller has not linked a Telegram chat.
var ErrNoChat = errors.New("no telegram chat linked to this shop")

type Repository interface {
	// ChatForShop returns the Telegram chat of the shop's seller, or ErrNoChat.
	ChatForShop(ctx context.Context, shopID int64) (int64, error)
}
