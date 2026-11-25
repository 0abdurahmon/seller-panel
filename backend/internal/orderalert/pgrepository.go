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
