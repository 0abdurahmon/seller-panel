package dbtest

import (
	"context"
	"errors"
	"fmt"
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

const testSuffix = "_test"

// ErrNotTestDatabase is returned for a database that is not clearly a throwaway one.
var ErrNotTestDatabase = errors.New("TEST_DATABASE_URL must point to a throwaway database whose name ends with _test")

// CheckName refuses any database whose name does not end with _test. The tests truncate tables,
// so pointing them at a development database would wipe its data.
func CheckName(name string) error {
	if !strings.HasSuffix(name, testSuffix) || len(name) == len(testSuffix) {
		return fmt.Errorf("%w, got %q", ErrNotTestDatabase, name)
	}
	return nil
}

// NewPool returns a pool on its own database "<TEST_DATABASE_URL name>_<suffix>", migrated.
// It stops the test when TEST_DATABASE_URL does not name a throwaway database.
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
	if err := CheckName(strings.TrimPrefix(u.Path, "/")); err != nil {
		t.Fatal(err)
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
