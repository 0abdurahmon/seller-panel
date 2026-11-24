package catalog

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0abdurahmon/seller-panel/backend/internal/database/dbtest"
)

func newPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return dbtest.NewPool(t, "catalog")
}

func TestPgListReturnsSeedTreeParentsFirst(t *testing.T) {
	repo := NewPgRepository(newPool(t))

	items, err := repo.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) < 12 {
		t.Fatalf("got %d categories, want the seeded 3 parents and 9 children", len(items))
	}

	seen := map[int64]bool{}
	tops := 0
	for _, c := range items {
		if c.ParentID == nil {
			tops++
		} else if !seen[*c.ParentID] {
			t.Fatalf("category %q appears before its parent %d", c.Name, *c.ParentID)
		}
		seen[c.ID] = true
	}
	if tops != 3 {
		t.Fatalf("top-level categories = %d, want 3", tops)
	}
	if items[0].Name != "Electronics" || items[0].ParentID != nil {
		t.Fatalf("first = %+v, want top-level Electronics", items[0])
	}
}

func TestPgNamesAreUniquePerParent(t *testing.T) {
	pool := newPool(t)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `INSERT INTO categories (name) VALUES ('Electronics')`); err == nil {
		t.Fatal("duplicate top-level name was accepted")
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO categories (name, parent_id)
		 SELECT 'Phones', id FROM categories WHERE name = 'Electronics' AND parent_id IS NULL`); err == nil {
		t.Fatal("duplicate name under the same parent was accepted")
	}
	if _, err := pool.Exec(ctx, `INSERT INTO categories (name) VALUES ('   ')`); err == nil {
		t.Fatal("blank name was accepted")
	}
}
