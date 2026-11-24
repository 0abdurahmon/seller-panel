package shop

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/0abdurahmon/seller-panel/backend/internal/database"
)

// These tests need PostgreSQL. Set TEST_DATABASE_URL to run them.
func newPgRepo(t *testing.T) (*PgRepository, func(sql string, args ...any) int64) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	if err := database.Migrate(url); err != nil {
		t.Fatal(err)
	}
	pool, err := database.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `TRUNCATE shops, sellers RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}
	insertSeller := func(sql string, args ...any) int64 {
		var id int64
		if err := pool.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	return NewPgRepository(pool), insertSeller
}

func TestPgShopGetUpdateAndScoping(t *testing.T) {
	repo, insert := newPgRepo(t)
	ctx := context.Background()

	sellerA := insert(`INSERT INTO sellers (email, password_hash) VALUES ('a@example.com', 'x') RETURNING id`)
	shopA := insert(`INSERT INTO shops (seller_id, name) VALUES ($1, 'Shop A') RETURNING id`, sellerA)
	sellerB := insert(`INSERT INTO sellers (email, password_hash) VALUES ('b@example.com', 'x') RETURNING id`)
	shopB := insert(`INSERT INTO shops (seller_id, name) VALUES ($1, 'Shop B') RETURNING id`, sellerB)

	got, err := repo.Get(ctx, shopA)
	if err != nil || got.Name != "Shop A" || got.LogoURL != nil {
		t.Fatalf("get A = %+v, %v", got, err)
	}

	updated, err := repo.Update(ctx, shopA, Profile{
		Name: "Shop A new", LogoURL: "https://x.example/logo.png", Description: "d",
		ContactPhone: "+998901234567", PickupAddress: "addr",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Shop A new" || updated.ContactPhone == nil || *updated.ContactPhone != "+998901234567" {
		t.Fatalf("updated = %+v", updated)
	}

	// Shop B is untouched by an update to shop A.
	b, err := repo.Get(ctx, shopB)
	if err != nil || b.Name != "Shop B" || b.ContactPhone != nil {
		t.Fatalf("shop B changed: %+v, %v", b, err)
	}

	// Clearing optional fields stores NULL.
	cleared, err := repo.Update(ctx, shopA, Profile{Name: "Shop A new"})
	if err != nil {
		t.Fatal(err)
	}
	if cleared.LogoURL != nil || cleared.ContactPhone != nil || cleared.PickupAddress != nil {
		t.Fatalf("optional fields not NULL after clear: %+v", cleared)
	}

	if _, err := repo.Get(ctx, 999999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing shop err = %v, want ErrNotFound", err)
	}
}
