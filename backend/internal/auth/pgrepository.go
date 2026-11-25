package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0abdurahmon/seller-panel/backend/internal/database"
)

const uniqueViolation = "23505"

type PgRepository struct {
	pool *pgxpool.Pool
}

// NewPgRepository returns the PostgreSQL-backed repository for sellers and shops.
func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

func (s *PgRepository) CreateSellerWithShop(ctx context.Context, email, passwordHash, shopName string) (int64, int64, error) {
	var sellerID, shopID int64
	err := database.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`INSERT INTO sellers (email, password_hash) VALUES ($1, $2) RETURNING id`,
			email, passwordHash,
		).Scan(&sellerID); err != nil {
			return err
		}
		return tx.QueryRow(ctx,
			`INSERT INTO shops (seller_id, name) VALUES ($1, $2) RETURNING id`,
			sellerID, shopName,
		).Scan(&shopID)
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return 0, 0, ErrEmailTaken
		}
		return 0, 0, fmt.Errorf("create seller: %w", err)
	}
	return sellerID, shopID, nil
}

func (s *PgRepository) FindCredentials(ctx context.Context, email string) (Credentials, error) {
	var c Credentials
	err := s.pool.QueryRow(ctx,
		`SELECT se.id, sh.id, se.password_hash
		   FROM sellers se
		   JOIN shops sh ON sh.seller_id = se.id
		  WHERE se.email = $1`,
		email,
	).Scan(&c.SellerID, &c.ShopID, &c.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Credentials{}, ErrNotFound
	}
	if err != nil {
		return Credentials{}, fmt.Errorf("find seller: %w", err)
	}
	return c, nil
}

func (s *PgRepository) FindTelegramChat(ctx context.Context, email string) (int64, int64, error) {
	var sellerID, chatID int64
	err := s.pool.QueryRow(ctx,
		`SELECT id, telegram_chat_id FROM sellers
		  WHERE email = $1 AND telegram_chat_id IS NOT NULL`,
		email,
	).Scan(&sellerID, &chatID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, ErrNotFound
	}
	if err != nil {
		return 0, 0, fmt.Errorf("find telegram chat: %w", err)
	}
	return sellerID, chatID, nil
}

func (s *PgRepository) SaveLinkToken(ctx context.Context, sellerID int64, tokenHash string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO telegram_link_tokens (token_hash, seller_id, expires_at) VALUES ($1, $2, $3)
		 ON CONFLICT (seller_id) DO UPDATE SET token_hash = EXCLUDED.token_hash, expires_at = EXCLUDED.expires_at`,
		tokenHash, sellerID, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("save link token: %w", err)
	}
	return nil
}

func (s *PgRepository) LinkTelegram(ctx context.Context, tokenHash string, chatID int64, now time.Time) (int64, error) {
	var sellerID int64
	err := database.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`DELETE FROM telegram_link_tokens WHERE token_hash = $1 AND expires_at > $2 RETURNING seller_id`,
			tokenHash, now,
		).Scan(&sellerID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrLinkInvalid
		}
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE sellers SET telegram_chat_id = $1 WHERE id = $2`, chatID, sellerID)
		return err
	})
	if errors.Is(err, ErrLinkInvalid) {
		return 0, ErrLinkInvalid
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return 0, ErrTelegramTaken
	}
	if err != nil {
		return 0, fmt.Errorf("link telegram: %w", err)
	}
	return sellerID, nil
}

func (s *PgRepository) FindLoginCode(ctx context.Context, sellerID int64) (LoginCode, error) {
	var c LoginCode
	err := s.pool.QueryRow(ctx,
		`SELECT code_hash, attempts, created_at, expires_at FROM login_codes WHERE seller_id = $1`,
		sellerID,
	).Scan(&c.CodeHash, &c.Attempts, &c.CreatedAt, &c.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return LoginCode{}, ErrNotFound
	}
	if err != nil {
		return LoginCode{}, fmt.Errorf("find login code: %w", err)
	}
	return c, nil
}

func (s *PgRepository) SaveLoginCode(ctx context.Context, sellerID int64, c LoginCode, replaceBefore time.Time) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`INSERT INTO login_codes (seller_id, code_hash, attempts, created_at, expires_at) VALUES ($1, $2, 0, $3, $4)
		 ON CONFLICT (seller_id) DO UPDATE
		   SET code_hash = EXCLUDED.code_hash, attempts = 0,
		       created_at = EXCLUDED.created_at, expires_at = EXCLUDED.expires_at
		   WHERE login_codes.created_at <= $5`,
		sellerID, c.CodeHash, c.CreatedAt, c.ExpiresAt, replaceBefore,
	)
	if err != nil {
		return false, fmt.Errorf("save login code: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (s *PgRepository) ClaimLoginAttempt(ctx context.Context, sellerID int64, maxAttempts int, now time.Time) (string, error) {
	var hash string
	err := s.pool.QueryRow(ctx,
		`UPDATE login_codes SET attempts = attempts + 1
		  WHERE seller_id = $1 AND attempts < $2 AND expires_at > $3
		  RETURNING code_hash`,
		sellerID, maxAttempts, now,
	).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("count login attempt: %w", err)
	}
	return hash, nil
}

func (s *PgRepository) DeleteLoginCode(ctx context.Context, sellerID int64, codeHash string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM login_codes WHERE seller_id = $1 AND code_hash = $2`, sellerID, codeHash)
	if err != nil {
		return false, fmt.Errorf("delete login code: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PgRepository) ShopOwnedBy(ctx context.Context, sellerID, shopID int64) (bool, error) {
	var owned bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM shops WHERE id = $1 AND seller_id = $2)`, shopID, sellerID).Scan(&owned)
	if err != nil {
		return false, fmt.Errorf("check the shop of a seller: %w", err)
	}
	return owned, nil
}
