package catalog

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

func (r *PgRepository) List(ctx context.Context) ([]Category, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, parent_id
		   FROM categories
		  ORDER BY COALESCE(parent_id, id), parent_id NULLS FIRST, id`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.ParentID); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return out, nil
}
