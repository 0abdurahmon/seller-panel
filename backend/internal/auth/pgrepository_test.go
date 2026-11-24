package auth

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/database"
)

// These tests need a real PostgreSQL. Set TEST_DATABASE_URL to run them;
// without it they are skipped.
func newPgService(t *testing.T) (*Service, *TokenIssuer) {
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

	tokens, err := NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(NewPgRepository(pool), tokens)
	if err != nil {
		t.Fatal(err)
	}
	return svc, tokens
}

func TestPgRegisterCreatesSellerAndShop(t *testing.T) {
	svc, tokens := newPgService(t)
	ctx := context.Background()

	reg, err := svc.Register(ctx, RegisterRequest{Email: "pg@example.com", Password: "correct-horse", ShopName: "PG Shop"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	claims, err := tokens.Parse(reg.Token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.SellerID == 0 || claims.ShopID == 0 {
		t.Fatalf("claims = %+v, want non-zero ids", claims)
	}

	login, err := svc.Login(ctx, LoginRequest{Email: "pg@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	loginClaims, _ := tokens.Parse(login.Token)
	if loginClaims.SellerID != claims.SellerID || loginClaims.ShopID != claims.ShopID {
		t.Fatalf("login ids %+v differ from register ids %+v", loginClaims, claims)
	}
}

func TestPgDuplicateEmailIsRejectedByDatabase(t *testing.T) {
	svc, _ := newPgService(t)
	ctx := context.Background()
	req := RegisterRequest{Email: "dup@example.com", Password: "correct-horse", ShopName: "A"}
	if _, err := svc.Register(ctx, req); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Register(ctx, req)
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
}

func TestPgRejectedRegisterCreatesNoExtraShop(t *testing.T) {
	svc, _ := newPgService(t)
	ctx := context.Background()
	if _, err := svc.Register(ctx, RegisterRequest{Email: "one@example.com", Password: "correct-horse", ShopName: "One"}); err != nil {
		t.Fatal(err)
	}
	// The duplicate fails inside the transaction; the shop insert must roll back.
	if _, err := svc.Register(ctx, RegisterRequest{Email: "one@example.com", Password: "correct-horse", ShopName: "Two"}); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
	var shops int
	pool := svc.repo.(*PgRepository).pool
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM shops`).Scan(&shops); err != nil {
		t.Fatal(err)
	}
	if shops != 1 {
		t.Fatalf("shops = %d, want 1 (rolled-back register left a shop behind)", shops)
	}
}
