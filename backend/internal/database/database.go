// Package database owns the PostgreSQL connection pool, migrations and the
// transaction helper.
package database

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0abdurahmon/seller-panel/backend/migrations"
)

// DBTX is what repositories run queries on. Both the pool and a transaction
// satisfy it, so one repository method works inside or outside InTx.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Connect opens a connection pool and checks that the database answers.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

// Migrate applies all pending up migrations. golang-migrate takes an advisory
// lock, so two backend instances starting together cannot race.
func Migrate(databaseURL string) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, toPgx5URL(databaseURL))
	if err != nil {
		return fmt.Errorf("init migrate: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// InTx runs fn in a transaction. It commits when fn returns nil and rolls
// back on error or panic.
func InTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) (err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// toPgx5URL converts a postgres:// URL to the scheme golang-migrate's pgx
// driver expects.
func toPgx5URL(databaseURL string) string {
	for _, prefix := range []string{"postgres://", "postgresql://"} {
		if rest, found := strings.CutPrefix(databaseURL, prefix); found {
			return "pgx5://" + withoutPoolParams(rest)
		}
	}
	return databaseURL
}

// golang-migrate passes pool_* parameters on to PostgreSQL, which refuses them.
// The query is edited as text so nothing else in the URL is re-encoded.
func withoutPoolParams(databaseURL string) string {
	base, query, found := strings.Cut(databaseURL, "?")
	if !found {
		return databaseURL
	}
	var kept []string
	for _, param := range strings.Split(query, "&") {
		name, _, _ := strings.Cut(param, "=")
		if decoded, err := url.QueryUnescape(name); err == nil {
			name = decoded
		}
		if !strings.HasPrefix(name, "pool_") {
			kept = append(kept, param)
		}
	}
	if len(kept) == 0 {
		return base
	}
	return base + "?" + strings.Join(kept, "&")
}
