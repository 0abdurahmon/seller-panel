package product

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/0abdurahmon/seller-panel/backend/internal/database"
)

const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"
	skuConstraint       = "products_shop_sku_key"
	columns             = `id, shop_id, category_id, title, description, price, sku, stock, reserved, status, created_at, updated_at`
)

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

type PgRepository struct {
	db database.DBTX
}

func NewPgRepository(db database.DBTX) *PgRepository {
	return &PgRepository{db: db}
}

func (r *PgRepository) Create(ctx context.Context, shopID int64, p Product) (Product, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO products (shop_id, category_id, title, description, price, sku, stock, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING `+columns,
		shopID, p.CategoryID, p.Title, p.Description, p.Price, p.SKU, p.Stock, p.Status)
	return scanWrite(row)
}

func (r *PgRepository) Get(ctx context.Context, shopID, id int64) (Product, error) {
	return scanRead(r.db.QueryRow(ctx,
		`SELECT `+columns+` FROM products WHERE id = $1 AND shop_id = $2`, id, shopID))
}

func (r *PgRepository) Update(ctx context.Context, shopID int64, p Product) (Product, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE products
		    SET category_id = $3, title = $4, description = $5, price = $6, sku = $7, status = $8, updated_at = now()
		  WHERE id = $1 AND shop_id = $2
		RETURNING `+columns,
		p.ID, shopID, p.CategoryID, p.Title, p.Description, p.Price, p.SKU, p.Status)
	return scanWrite(row)
}

func (r *PgRepository) List(ctx context.Context, shopID int64, f ListFilter) ([]Product, int, error) {
	var status, category, query any
	if f.Status != "" {
		status = string(f.Status)
	}
	if f.CategoryID != 0 {
		category = f.CategoryID
	}
	if f.Query != "" {
		query = likeEscaper.Replace(f.Query)
	}
	const where = `shop_id = $1
		  AND ($2::product_status IS NULL OR status = $2)
		  AND ($3::bigint IS NULL OR category_id = $3)
		  AND ($4::text IS NULL OR title ILIKE '%' || $4 || '%' OR sku ILIKE '%' || $4 || '%')`

	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM products WHERE `+where,
		shopID, status, category, query).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+columns+` FROM products WHERE `+where+`
		  ORDER BY created_at DESC, id DESC
		  LIMIT $5 OFFSET $6`,
		shopID, status, category, query, f.PageSize, (f.Page-1)*f.PageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	var out []Product
	for rows.Next() {
		p, err := scanRead(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	return out, total, nil
}

func scanRead(row pgx.Row) (Product, error) {
	var p Product
	err := row.Scan(&p.ID, &p.ShopID, &p.CategoryID, &p.Title, &p.Description, &p.Price, &p.SKU,
		&p.Stock, &p.Reserved, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	if err != nil {
		return Product{}, fmt.Errorf("scan product: %w", err)
	}
	p.CreatedAt = p.CreatedAt.UTC()
	p.UpdatedAt = p.UpdatedAt.UTC()
	return p, nil
}

func scanWrite(row pgx.Row) (Product, error) {
	p, err := scanRead(row)
	if err == nil || errors.Is(err, ErrNotFound) {
		return p, err
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch {
		case pgErr.Code == uniqueViolation && pgErr.ConstraintName == skuConstraint:
			return Product{}, ErrSKUTaken
		case pgErr.Code == foreignKeyViolation && strings.Contains(pgErr.ConstraintName, "category"):
			return Product{}, fmt.Errorf("%w: category does not exist", ErrValidation)
		}
	}
	return Product{}, err
}
