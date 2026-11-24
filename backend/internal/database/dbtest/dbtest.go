package dbtest

import (
	"context"
	"errors"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0abdurahmon/seller-panel/backend/internal/database"
)

var validName = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

func NewPool(t *testing.T, suffix string) *pgxpool.Pool {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("TEST_DATABASE_URL: %v", err)
	}
	name := strings.TrimPrefix(u.Path, "/") + "_" + suffix
	if !validName.MatchString(name) {
		t.Fatalf("unexpected database name %q", name)
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize())
	var pgErr *pgconn.PgError
	if err != nil && !(errors.As(err, &pgErr) && (pgErr.Code == "42P04" || pgErr.Code == "23505")) {
		t.Fatalf("create test database: %v", err)
	}

	u.Path = "/" + name
	if err := database.Migrate(u.String()); err != nil {
		t.Fatal(err)
	}
	pool, err := database.Connect(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
