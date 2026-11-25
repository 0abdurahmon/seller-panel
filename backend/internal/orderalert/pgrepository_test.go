package orderalert

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/database/dbtest"
)

func TestPgChatForShop(t *testing.T) {
	pool := dbtest.NewPool(t, "orderalert")
	ctx := context.Background()
	var linked, unlinked, linkedShop, unlinkedShop int64
	// The test database is reused between runs, so emails are unique per run.
	n := time.Now().UnixNano()
	chat := n%1_000_000_000 + 3_000_000 // chat ids are unique too
	if err := pool.QueryRow(ctx, `INSERT INTO sellers (email, password_hash, telegram_chat_id) VALUES ($1, 'x', $2) RETURNING id`,
		fmt.Sprintf("alert-linked-%d@example.com", n), chat).Scan(&linked); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO sellers (email, password_hash) VALUES ($1, 'x') RETURNING id`,
		fmt.Sprintf("alert-unlinked-%d@example.com", n)).Scan(&unlinked); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO shops (seller_id, name) VALUES ($1, 'Linked') RETURNING id`, linked).Scan(&linkedShop); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO shops (seller_id, name) VALUES ($1, 'Unlinked') RETURNING id`, unlinked).Scan(&unlinkedShop); err != nil {
		t.Fatal(err)
	}

	repo := NewPgRepository(pool)
	got, err := repo.ChatForShop(ctx, linkedShop)
	if err != nil || got != chat {
		t.Fatalf("linked shop: chat = %d, err = %v", got, err)
	}
	if _, err := repo.ChatForShop(ctx, unlinkedShop); !errors.Is(err, ErrNoChat) {
		t.Fatalf("unlinked shop: err = %v, want ErrNoChat", err)
	}
	if _, err := repo.ChatForShop(ctx, 999999999); !errors.Is(err, ErrNoChat) {
		t.Fatalf("missing shop: err = %v, want ErrNoChat", err)
	}
}
