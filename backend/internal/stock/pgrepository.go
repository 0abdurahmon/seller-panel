package stock

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgRepository changes stock in the products table, always filtered by shop_id.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewPgRepository returns the PostgreSQL stock repository.
func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

// SetStock checks all rules in one UPDATE; a follow-up read only explains refusals.
func (s *PgRepository) SetStock(ctx context.Context, shopID, productID int64, stock, expected int) (Row, error) {
	var row Row
	err := s.pool.QueryRow(ctx,
		`UPDATE products
		    SET stock = $3, updated_at = now()
		  WHERE id = $1 AND shop_id = $2
		    AND status <> 'archived'
		    AND stock = $4
		    AND reserved <= $3
		RETURNING stock, reserved, updated_at`,
		productID, shopID, stock, expected,
	).Scan(&row.Stock, &row.Reserved, &row.UpdatedAt)
	if err == nil {
		return row, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Row{}, fmt.Errorf("set stock: %w", err)
	}

	var status string
	var current Row
	err = s.pool.QueryRow(ctx,
		`SELECT status, stock, reserved, updated_at FROM products WHERE id = $1 AND shop_id = $2`,
		productID, shopID,
	).Scan(&status, &current.Stock, &current.Reserved, &current.UpdatedAt)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Row{}, ErrNotFound
	case err != nil:
		return Row{}, fmt.Errorf("explain set stock: %w", err)
	case status == "archived":
		return Row{}, ErrArchived
	case current.Stock != expected:
		return current, ErrStockChanged
	default:
		return current, ErrBelowReserved
	}
}
