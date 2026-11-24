package balance

import (
	"context"
	"fmt"

	"github.com/0abdurahmon/seller-panel/backend/internal/database"
)

type PgRepository struct {
	db database.DBTX
}

func NewPgRepository(db database.DBTX) *PgRepository {
	return &PgRepository{db: db}
}

func (r *PgRepository) Summary(ctx context.Context, shopID int64) (Summary, error) {
	var s Summary
	err := r.db.QueryRow(ctx,
		`SELECT coalesce(sum(amount), 0)::bigint,
		        coalesce(sum(amount) FILTER (WHERE type = 'sale'), 0)::bigint,
		        coalesce(sum(amount) FILTER (WHERE type = 'commission'), 0)::bigint
		   FROM balance_entries
		  WHERE shop_id = $1`,
		shopID,
	).Scan(&s.Available, &s.Earned, &s.Commission)
	if err != nil {
		return Summary{}, fmt.Errorf("sum balance: %w", err)
	}
	return s, nil
}

func (r *PgRepository) Entries(ctx context.Context, shopID int64, limit, offset int) ([]Entry, int, error) {
	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT count(*) FROM balance_entries WHERE shop_id = $1`, shopID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count balance entries: %w", err)
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, order_id, type::text, amount, created_at
		   FROM balance_entries
		  WHERE shop_id = $1
		  ORDER BY created_at DESC, id DESC
		  LIMIT $2 OFFSET $3`,
		shopID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list balance entries: %w", err)
	}
	defer rows.Close()

	var out []Entry
	for rows.Next() {
		var e Entry
		var typ string
		if err := rows.Scan(&e.ID, &e.OrderID, &typ, &e.Amount, &e.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan balance entry: %w", err)
		}
		e.Type = EntryType(typ)
		e.CreatedAt = e.CreatedAt.UTC()
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list balance entries: %w", err)
	}
	return out, total, nil
}
