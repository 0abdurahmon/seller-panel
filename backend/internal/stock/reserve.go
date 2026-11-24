package stock

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// Errors for order stock operations.
var (
	ErrInsufficientStock   = errors.New("insufficient stock")
	ErrReservationMismatch = errors.New("fewer units reserved than the order holds")
)

// Execer runs a statement; *pgxpool.Pool and pgx.Tx both satisfy it.
type Execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Reserve adds qty to reserved when an order is accepted; never oversells.
func Reserve(ctx context.Context, db Execer, shopID, productID int64, qty int) error {
	return change(ctx, db, "reserve", ErrInsufficientStock,
		`UPDATE products SET reserved = reserved + $3, updated_at = now()
		  WHERE id = $1 AND shop_id = $2 AND stock - reserved >= $3`,
		shopID, productID, qty)
}

// Release returns qty reserved units when an accepted or packed order is cancelled.
func Release(ctx context.Context, db Execer, shopID, productID int64, qty int) error {
	return change(ctx, db, "release", ErrReservationMismatch,
		`UPDATE products SET reserved = reserved - $3, updated_at = now()
		  WHERE id = $1 AND shop_id = $2 AND reserved >= $3`,
		shopID, productID, qty)
}

// Commit removes qty from stock and reserved when an order is delivered.
func Commit(ctx context.Context, db Execer, shopID, productID int64, qty int) error {
	return change(ctx, db, "commit", ErrReservationMismatch,
		`UPDATE products SET stock = stock - $3, reserved = reserved - $3, updated_at = now()
		  WHERE id = $1 AND shop_id = $2 AND reserved >= $3`,
		shopID, productID, qty)
}

// change runs one guarded UPDATE; zero changed rows returns refused.
func change(ctx context.Context, db Execer, op string, refused error, sql string, shopID, productID int64, qty int) error {
	if qty <= 0 || qty > MaxStock {
		return fmt.Errorf("%w: quantity must be between 1 and %d", ErrValidation, MaxStock)
	}
	tag, err := db.Exec(ctx, sql, productID, shopID, qty)
	if err != nil {
		return fmt.Errorf("%s stock: %w", op, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s product %d x%d: %w", op, productID, qty, refused)
	}
	return nil
}
