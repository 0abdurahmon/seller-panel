package auth

import (
	"context"
	"time"
)

type Credentials struct {
	SellerID     int64
	ShopID       int64
	PasswordHash string
}

// LoginCode is the active sign-in code of one seller. Only its hash is stored.
type LoginCode struct {
	CodeHash  string
	Attempts  int
	CreatedAt time.Time
	ExpiresAt time.Time
}

type Repository interface {
	CreateSellerWithShop(ctx context.Context, email, passwordHash, shopName string) (sellerID, shopID int64, err error)
	FindCredentials(ctx context.Context, email string) (Credentials, error)
	// ShopOwnedBy reports whether the shop exists and belongs to the seller.
	ShopOwnedBy(ctx context.Context, sellerID, shopID int64) (bool, error)

	// FindTelegramChat returns ErrNotFound unless the email belongs to a seller with a linked chat.
	FindTelegramChat(ctx context.Context, email string) (sellerID, chatID int64, err error)
	// SaveLinkToken replaces the seller's pending link token.
	SaveLinkToken(ctx context.Context, sellerID int64, tokenHash string, expiresAt time.Time) error
	// LinkTelegram consumes a live link token and stores the chat on its seller.
	// It returns ErrLinkInvalid for unknown, expired or used tokens and ErrTelegramTaken
	// when the chat belongs to another seller.
	LinkTelegram(ctx context.Context, tokenHash string, chatID int64, now time.Time) (sellerID int64, err error)

	FindLoginCode(ctx context.Context, sellerID int64) (LoginCode, error)
	// SaveLoginCode stores a code with no attempts. It replaces the seller's code only if that one
	// was created at or before replaceBefore, and reports whether the new code was stored.
	SaveLoginCode(ctx context.Context, sellerID int64, code LoginCode, replaceBefore time.Time) (bool, error)
	// ClaimLoginAttempt counts a try against a live code with tries left and returns its hash,
	// or ErrNotFound. The count and the check are one step, so parallel tries cannot pass the limit.
	ClaimLoginAttempt(ctx context.Context, sellerID int64, maxAttempts int, now time.Time) (codeHash string, err error)
	// DeleteLoginCode removes the code with this hash and reports whether this call removed it.
	DeleteLoginCode(ctx context.Context, sellerID int64, codeHash string) (bool, error)
}
