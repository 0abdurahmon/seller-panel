package product

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

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
	pool *pgxpool.Pool
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

func (r *PgRepository) Create(ctx context.Context, shopID int64, p Product) (Product, error) {
	var created Product
	err := database.InTx(ctx, r.pool, func(tx pgx.Tx) error {
		var err error
		created, err = scanWrite(tx.QueryRow(ctx,
			`INSERT INTO products (shop_id, category_id, title, description, price, sku, stock, status)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 RETURNING `+columns,
			shopID, p.CategoryID, p.Title, p.Description, p.Price, p.SKU, p.Stock, p.Status))
		if err != nil {
			return err
		}
		return replacePhotos(ctx, tx, &created, p.Photos)
	})
	if err != nil {
		return Product{}, err
	}
	return created, nil
}

func (r *PgRepository) Get(ctx context.Context, shopID, id int64) (Product, error) {
	p, err := scanRead(r.pool.QueryRow(ctx,
		`SELECT `+columns+` FROM products WHERE id = $1 AND shop_id = $2`, id, shopID))
	if err != nil {
		return Product{}, err
	}
	items := []Product{p}
	if err := loadPhotos(ctx, r.pool, items); err != nil {
		return Product{}, err
	}
	return items[0], nil
}

func (r *PgRepository) Update(ctx context.Context, shopID, id int64, change func(Product) (Product, error)) (Product, error) {
	var updated Product
	err := database.InTx(ctx, r.pool, func(tx pgx.Tx) error {
		current, err := scanRead(tx.QueryRow(ctx,
			`SELECT `+columns+` FROM products WHERE id = $1 AND shop_id = $2 FOR NO KEY UPDATE`, id, shopID))
		if err != nil {
			return err
		}
		locked := []Product{current}
		if err := loadPhotos(ctx, tx, locked); err != nil {
			return err
		}
		p, err := change(locked[0])
		if err != nil {
			return err
		}
		updated, err = scanWrite(tx.QueryRow(ctx,
			`UPDATE products
			    SET category_id = $3, title = $4, description = $5, price = $6, sku = $7, status = $8, updated_at = now()
			  WHERE id = $1 AND shop_id = $2
			RETURNING `+columns,
			id, shopID, p.CategoryID, p.Title, p.Description, p.Price, p.SKU, p.Status))
		if err != nil {
			return err
		}
		return replacePhotos(ctx, tx, &updated, p.Photos)
	})
	if err != nil {
		return Product{}, err
	}
	return updated, nil
}

// replacePhotos stores photos in order (position 0 is the cover) and sets them on p.
func replacePhotos(ctx context.Context, db database.DBTX, p *Product, photos []string) error {
	if _, err := db.Exec(ctx, `DELETE FROM product_photos WHERE product_id = $1`, p.ID); err != nil {
		return fmt.Errorf("clear photos: %w", err)
	}
	if len(photos) > 0 {
		if _, err := db.Exec(ctx,
			`INSERT INTO product_photos (product_id, url, position)
			 SELECT $1, u, ord - 1 FROM unnest($2::text[]) WITH ORDINALITY AS t(u, ord)`,
			p.ID, photos); err != nil {
			return fmt.Errorf("insert photos: %w", err)
		}
	}
	p.Photos = append([]string{}, photos...)
	return nil
}

// loadPhotos fills Photos of every product with one query.
func loadPhotos(ctx context.Context, db database.DBTX, items []Product) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]int64, len(items))
	index := make(map[int64]int, len(items))
	for i := range items {
		ids[i] = items[i].ID
		index[items[i].ID] = i
		items[i].Photos = []string{}
	}
	rows, err := db.Query(ctx,
		`SELECT product_id, url FROM product_photos WHERE product_id = ANY($1) ORDER BY product_id, position`, ids)
	if err != nil {
		return fmt.Errorf("load photos: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var url string
		if err := rows.Scan(&id, &url); err != nil {
			return fmt.Errorf("scan photo: %w", err)
		}
		i := index[id]
		items[i].Photos = append(items[i].Photos, url)
	}
	return rows.Err()
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
		  AND ($3::bigint IS NULL OR category_id IN (
		      WITH RECURSIVE tree AS (
		          SELECT id FROM categories WHERE id = $3
		          UNION
		          SELECT c.id FROM categories c JOIN tree ON c.parent_id = tree.id
		      )
		      SELECT id FROM tree))
		  AND ($4::text IS NULL OR title ILIKE '%' || $4 || '%' OR sku ILIKE '%' || $4 || '%')`

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM products WHERE `+where,
		shopID, status, category, query).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	rows, err := r.pool.Query(ctx,
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
	if err := loadPhotos(ctx, r.pool, out); err != nil {
		return nil, 0, err
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
