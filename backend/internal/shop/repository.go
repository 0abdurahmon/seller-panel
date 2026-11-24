package shop

import (
	"context"
	"errors"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when the shop does not exist.
var ErrNotFound = errors.New("shop not found")

// Profile is the set of fields that PUT /shop writes, already validated.
type Profile struct {
	Name          string
	LogoURL       string
	Description   string
	ContactPhone  string
	PickupAddress string
}

// Repository stores shop profiles. Every method is scoped by shopID.
type Repository interface {
	Get(ctx context.Context, shopID int64) (Shop, error)
	Update(ctx context.Context, shopID int64, p Profile) (Shop, error)
}

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

// MemRepository is an in-memory Repository for unit tests.
type MemRepository struct {
	mu    sync.Mutex
	shops map[int64]Shop
}

// NewMemRepository returns an empty in-memory repository.
func NewMemRepository() *MemRepository {
	return &MemRepository{shops: make(map[int64]Shop)}
}

// Seed adds a shop with the given ID and name.
func (m *MemRepository) Seed(id int64, name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shops[id] = Shop{ID: id, Name: name}
}

func (m *MemRepository) Get(_ context.Context, shopID int64) (Shop, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.shops[shopID]
	if !ok {
		return Shop{}, ErrNotFound
	}
	return s, nil
}

func (m *MemRepository) Update(_ context.Context, shopID int64, p Profile) (Shop, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.shops[shopID]; !ok {
		return Shop{}, ErrNotFound
	}
	s := Shop{
		ID:          shopID,
		Name:        p.Name,
		Description: p.Description,
	}
	if p.LogoURL != "" {
		s.LogoURL = &p.LogoURL
	}
	if p.ContactPhone != "" {
		s.ContactPhone = &p.ContactPhone
	}
	if p.PickupAddress != "" {
		s.PickupAddress = &p.PickupAddress
	}
	m.shops[shopID] = s
	return s, nil
}
