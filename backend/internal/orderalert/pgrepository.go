package orderalert

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgRepository struct {
	pool *pgxpool.Pool
}

// NewPgRepository returns the PostgreSQL-backed chat lookup.
func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

func (r *PgRepository) ChatForShop(ctx context.Context, shopID int64) (int64, error) {
	var chat *int64
	err := r.pool.QueryRow(ctx,
		`SELECT se.telegram_chat_id FROM shops sh JOIN sellers se ON se.id = sh.seller_id WHERE sh.id = $1`,
		shopID,
	).Scan(&chat)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && chat == nil) {
		return 0, ErrNoChat
	}
	if err != nil {
		return 0, fmt.Errorf("find telegram chat: %w", err)
	}
	return *chat, nil
}

func (r *PgRepository) Lines(ctx context.Context, shopID, orderID int64) ([]Line, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT i.title_snapshot, i.quantity, i.price_snapshot
		   FROM order_items i JOIN orders o ON o.id = i.order_id
		  WHERE o.id = $2 AND o.shop_id = $1
		  ORDER BY i.id`,
		shopID, orderID)
	if err != nil {
		return nil, fmt.Errorf("load order lines: %w", err)
	}
	lines, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Line])
	if err != nil {
		return nil, fmt.Errorf("load order lines: %w", err)
	}
	if len(lines) == 0 {
		return nil, ErrNotFound
	}
	return lines, nil
}
