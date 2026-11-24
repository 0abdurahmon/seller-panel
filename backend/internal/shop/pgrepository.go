package shop

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgRepository is the PostgreSQL implementation.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewPgRepository returns a PostgreSQL-backed repository.
func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

const selectShop = `SELECT id, name, logo_url, description, contact_phone, pickup_address FROM shops`

func (r *PgRepository) Get(ctx context.Context, shopID int64) (Shop, error) {
	return scanShop(r.pool.QueryRow(ctx, selectShop+` WHERE id = $1`, shopID))
}

func (r *PgRepository) Update(ctx context.Context, shopID int64, p Profile) (Shop, error) {
	return scanShop(r.pool.QueryRow(ctx,
		`UPDATE shops
		    SET name = $2, logo_url = $3, description = $4, contact_phone = $5, pickup_address = $6
		  WHERE id = $1
		RETURNING id, name, logo_url, description, contact_phone, pickup_address`,
		shopID, p.Name, nullable(p.LogoURL), p.Description, nullable(p.ContactPhone), nullable(p.PickupAddress),
	))
}

func scanShop(row pgx.Row) (Shop, error) {
	var s Shop
	err := row.Scan(&s.ID, &s.Name, &s.LogoURL, &s.Description, &s.ContactPhone, &s.PickupAddress)
	if errors.Is(err, pgx.ErrNoRows) {
		return Shop{}, ErrNotFound
	}
	if err != nil {
		return Shop{}, err
	}
	return s, nil
}

// nullable maps an empty string to SQL NULL.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
