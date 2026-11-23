package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0abdurahmon/seller-panel/backend/internal/database"
)

const uniqueViolation = "23505"

type pgStore struct {
	pool *pgxpool.Pool
}

// NewPgStore returns the PostgreSQL-backed store for sellers and shops.
func NewPgStore(pool *pgxpool.Pool) store {
	return &pgStore{pool: pool}
}

func (s *pgStore) CreateSellerWithShop(ctx context.Context, email, passwordHash, shopName string) (int64, int64, error) {
	var sellerID, shopID int64
	err := database.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`INSERT INTO sellers (email, password_hash) VALUES ($1, $2) RETURNING id`,
			email, passwordHash,
		).Scan(&sellerID); err != nil {
			return err
		}
		return tx.QueryRow(ctx,
			`INSERT INTO shops (seller_id, name) VALUES ($1, $2) RETURNING id`,
			sellerID, shopName,
		).Scan(&shopID)
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return 0, 0, ErrEmailTaken
		}
		return 0, 0, fmt.Errorf("create seller: %w", err)
	}
	return sellerID, shopID, nil
}

func (s *pgStore) FindCredentials(ctx context.Context, email string) (Credentials, error) {
	var c Credentials
	err := s.pool.QueryRow(ctx,
		`SELECT se.id, sh.id, se.password_hash
		   FROM sellers se
		   JOIN shops sh ON sh.seller_id = se.id
		  WHERE se.email = $1`,
		email,
	).Scan(&c.SellerID, &c.ShopID, &c.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Credentials{}, ErrNotFound
	}
	if err != nil {
		return Credentials{}, fmt.Errorf("find seller: %w", err)
	}
	return c, nil
}
