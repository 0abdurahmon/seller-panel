package database_test

import (
	"net/url"
	"testing"

	"github.com/0abdurahmon/seller-panel/backend/internal/database"
	"github.com/0abdurahmon/seller-panel/backend/internal/database/dbtest"
)

func TestPoolSettingsInTheURL(t *testing.T) {
	u, err := url.Parse(dbtest.NewPool(t, "database").Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("pool_max_conns", "3")
	q.Set("pool_min_conns", "1")
	u.RawQuery = q.Encode()

	if err := database.Migrate(u.String()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := database.Connect(t.Context(), u.String())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	if got := pool.Config(); got.MaxConns != 3 || got.MinConns != 1 {
		t.Fatalf("pool = max %d, min %d; want 3 and 1", got.MaxConns, got.MinConns)
	}
}
