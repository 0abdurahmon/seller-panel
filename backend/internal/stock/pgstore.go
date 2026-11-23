package stock

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/0abdurahmon/seller-panel/backend/internal/database"
)

// PgStore changes stock in the products table. The table itself is created by
// the products migration; this store only reads and writes stock, reserved
// and updated_at. Every statement is filtered by shop_id.
type PgStore struct {
	db database.DBTX
}

// NewPgStore returns the PostgreSQL stock store. db is usually the pool.
func NewPgStore(db database.DBTX) *PgStore {
	return &PgStore{db: db}
}

// SetStock implements Repository. The update is one statement whose WHERE
// clause holds every rule, so a concurrent order accept cannot slip between a
// check and the write. When no row changes, a second read explains why.
func (s *PgStore) SetStock(ctx context.Context, shopID, productID int64, stock int) (Row, error) {
	var row Row
	err := s.db.QueryRow(ctx,
		`UPDATE products
		    SET stock = $3, updated_at = now()
		  WHERE id = $1 AND shop_id = $2
		    AND status <> 'archived'
		    AND reserved <= $3
		RETURNING stock, reserved, updated_at`,
		productID, shopID, stock,
	).Scan(&row.Stock, &row.Reserved, &row.UpdatedAt)
	if err == nil {
		return row, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Row{}, fmt.Errorf("set stock: %w", err)
	}

	var status string
	var reserved int
	err = s.db.QueryRow(ctx,
		`SELECT status, reserved FROM products WHERE id = $1 AND shop_id = $2`,
		productID, shopID,
	).Scan(&status, &reserved)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Row{}, ErrNotFound
	case err != nil:
		return Row{}, fmt.Errorf("explain set stock: %w", err)
	case status == "archived":
		return Row{}, ErrArchived
	default:
		return Row{}, ErrBelowReserved
	}
}

// Reserve adds qty to reserved when an order is accepted. It never oversells:
// the row only changes when stock - reserved >= qty, otherwise it returns
// ErrInsufficientStock. Run it inside the order's transaction (pass the tx),
// one item at a time in ascending product id order to avoid deadlocks.
func (s *PgStore) Reserve(ctx context.Context, db database.DBTX, shopID, productID int64, qty int) error {
	if qty <= 0 {
		return fmt.Errorf("%w: quantity must be positive", ErrValidation)
	}
	tag, err := db.Exec(ctx,
		`UPDATE products
		    SET reserved = reserved + $3, updated_at = now()
		  WHERE id = $1 AND shop_id = $2 AND stock - reserved >= $3`,
		productID, shopID, qty,
	)
	if err != nil {
		return fmt.Errorf("reserve stock: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInsufficientStock
	}
	return nil
}

// Release returns qty reserved units when an accepted or packed order is
// cancelled.
func (s *PgStore) Release(ctx context.Context, db database.DBTX, shopID, productID int64, qty int) error {
	return s.change(ctx, db, "release stock",
		`UPDATE products
		    SET reserved = reserved - $3, updated_at = now()
		  WHERE id = $1 AND shop_id = $2 AND reserved >= $3`,
		shopID, productID, qty)
}

// Commit removes qty units from both stock and reserved when an order is
// delivered: the goods have left the shop.
func (s *PgStore) Commit(ctx context.Context, db database.DBTX, shopID, productID int64, qty int) error {
	return s.change(ctx, db, "commit stock",
		`UPDATE products
		    SET stock = stock - $3, reserved = reserved - $3, updated_at = now()
		  WHERE id = $1 AND shop_id = $2 AND reserved >= $3`,
		shopID, productID, qty)
}

// change runs a release or commit. Zero rows means the product is missing or
// fewer units are reserved than the order holds, which is a bug in the caller:
// it is reported as an error so the order transaction rolls back.
func (s *PgStore) change(ctx context.Context, db database.DBTX, what, sql string, shopID, productID int64, qty int) error {
	if qty <= 0 {
		return fmt.Errorf("%w: quantity must be positive", ErrValidation)
	}
	tag, err := db.Exec(ctx, sql, productID, shopID, qty)
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s: product %d has fewer than %d reserved units", what, productID, qty)
	}
	return nil
}
