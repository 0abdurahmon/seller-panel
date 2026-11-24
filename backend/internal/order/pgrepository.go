package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0abdurahmon/seller-panel/backend/internal/database"
)

const orderColumns = `id, shop_id, '#' || number, status::text, buyer_name, buyer_phone, delivery_address,
	total, cancel_reason, created_at, status_changed_at`

type PgRepository struct {
	pool *pgxpool.Pool
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

func (r *PgRepository) List(ctx context.Context, shopID int64, f Filter) (Page[Summary], error) {
	var status any
	if f.Status != nil {
		status = string(*f.Status)
	}
	const where = `shop_id = $1 AND ($2::order_status IS NULL OR status = $2::order_status)`

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE `+where, shopID, status).Scan(&total); err != nil {
		return Page[Summary]{}, fmt.Errorf("count orders: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, '#' || number, status::text, buyer_name, total, created_at
		   FROM orders
		  WHERE `+where+`
		  ORDER BY created_at DESC, id DESC
		  LIMIT $3 OFFSET $4`,
		shopID, status, f.PageSize, (f.Page-1)*f.PageSize)
	if err != nil {
		return Page[Summary]{}, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	items := []Summary{}
	for rows.Next() {
		var s Summary
		var st string
		if err := rows.Scan(&s.ID, &s.Number, &st, &s.BuyerName, &s.Total, &s.CreatedAt); err != nil {
			return Page[Summary]{}, fmt.Errorf("scan order: %w", err)
		}
		s.Status = Status(st)
		s.CreatedAt = s.CreatedAt.UTC()
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return Page[Summary]{}, fmt.Errorf("list orders: %w", err)
	}
	return Page[Summary]{Items: items, Page: f.Page, PageSize: f.PageSize, Total: total}, nil
}

func (r *PgRepository) Get(ctx context.Context, shopID, orderID int64) (Order, error) {
	return loadOrder(ctx, r.pool, shopID, orderID, false, "id")
}

func (r *PgRepository) SimulatableProducts(ctx context.Context, shopID int64) ([]ProductRef, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, title, price, stock - reserved
		   FROM products
		  WHERE shop_id = $1 AND status = 'active' AND stock - reserved > 0
		  ORDER BY id`, shopID)
	if err != nil {
		return nil, fmt.Errorf("list products to order: %w", err)
	}
	defer rows.Close()

	var out []ProductRef
	for rows.Next() {
		var p ProductRef
		if err := rows.Scan(&p.ID, &p.Title, &p.Price, &p.Available); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list products to order: %w", err)
	}
	return out, nil
}

func (r *PgRepository) Create(ctx context.Context, shopID int64, n NewOrder, at time.Time) (Order, error) {
	var created Order
	err := database.InTx(ctx, r.pool, func(tx pgx.Tx) error {
		var total int64
		for _, it := range n.Items {
			total += it.Price * int64(it.Quantity)
		}
		var id int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO orders (shop_id, buyer_name, buyer_phone, delivery_address, total, created_at, status_changed_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $6) RETURNING id`,
			shopID, n.BuyerName, n.BuyerPhone, n.DeliveryAddress, total, at).Scan(&id); err != nil {
			return fmt.Errorf("insert order: %w", err)
		}
		for _, it := range n.Items {
			if _, err := tx.Exec(ctx,
				`INSERT INTO order_items (order_id, product_id, title_snapshot, price_snapshot, quantity) VALUES ($1, $2, $3, $4, $5)`,
				id, it.ProductID, it.Title, it.Price, it.Quantity); err != nil {
				return fmt.Errorf("insert order item: %w", err)
			}
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO order_status_history (order_id, from_status, to_status, created_at) VALUES ($1, NULL, 'new', $2)`,
			id, at); err != nil {
			return fmt.Errorf("insert first history row: %w", err)
		}
		o, err := loadOrder(ctx, tx, shopID, id, false, "id")
		if err != nil {
			return err
		}
		created = o
		return nil
	})
	if err != nil {
		return Order{}, err
	}
	return created, nil
}

func (r *PgRepository) InTx(ctx context.Context, fn func(tx TxRepository) error) error {
	return database.InTx(ctx, r.pool, func(tx pgx.Tx) error {
		return fn(&pgTx{tx: tx})
	})
}

type pgTx struct {
	tx pgx.Tx
}

func (t *pgTx) LockOrder(ctx context.Context, shopID, orderID int64) (Order, error) {
	return loadOrder(ctx, t.tx, shopID, orderID, true, "product_id")
}

func (t *pgTx) Reserve(ctx context.Context, shopID, productID int64, qty int) error {
	tag, err := t.tx.Exec(ctx,
		`UPDATE products
		    SET reserved = reserved + $3, updated_at = now()
		  WHERE id = $1 AND shop_id = $2 AND stock - reserved >= $3`,
		productID, shopID, qty)
	if err != nil {
		return fmt.Errorf("reserve stock: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return nil
	}
	var exists bool
	if err := t.tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM products WHERE id = $1 AND shop_id = $2)`, productID, shopID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check product: %w", err)
	}
	if !exists {
		return ErrProductNotFound
	}
	return ErrInsufficientStock
}

func (t *pgTx) Release(ctx context.Context, shopID, productID int64, qty int) error {
	return t.updateStock(ctx, "release stock",
		`UPDATE products SET reserved = reserved - $3, updated_at = now() WHERE id = $1 AND shop_id = $2`,
		shopID, productID, qty)
}

func (t *pgTx) Deliver(ctx context.Context, shopID, productID int64, qty int) error {
	return t.updateStock(ctx, "deliver stock",
		`UPDATE products SET stock = stock - $3, reserved = reserved - $3, updated_at = now() WHERE id = $1 AND shop_id = $2`,
		shopID, productID, qty)
}

func (t *pgTx) updateStock(ctx context.Context, what, sql string, shopID, productID int64, qty int) error {
	tag, err := t.tx.Exec(ctx, sql, productID, shopID, qty)
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrProductNotFound
	}
	return nil
}

func (t *pgTx) SetStatus(ctx context.Context, shopID, orderID int64, to Status, reason *string, at time.Time) error {
	tag, err := t.tx.Exec(ctx,
		`INSERT INTO order_status_history (order_id, from_status, to_status, reason, created_at)
		 SELECT id, status, $3::order_status, $4, $5 FROM orders WHERE id = $1 AND shop_id = $2`,
		orderID, shopID, string(to), reason, at)
	if err != nil {
		return fmt.Errorf("write status history: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if _, err := t.tx.Exec(ctx,
		`UPDATE orders SET status = $3::order_status, cancel_reason = $4, status_changed_at = $5
		  WHERE id = $1 AND shop_id = $2`,
		orderID, shopID, string(to), reason, at); err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	return nil
}

func (t *pgTx) AddBalance(ctx context.Context, entries []BalanceEntry) error {
	for _, e := range entries {
		if _, err := t.tx.Exec(ctx,
			`INSERT INTO balance_entries (shop_id, order_id, type, amount) VALUES ($1, $2, $3::balance_entry_type, $4)`,
			e.ShopID, e.OrderID, e.Type, e.Amount); err != nil {
			return fmt.Errorf("add balance entry: %w", err)
		}
	}
	return nil
}

func loadOrder(ctx context.Context, db database.DBTX, shopID, orderID int64, lock bool, itemOrder string) (Order, error) {
	query := `SELECT ` + orderColumns + ` FROM orders WHERE id = $1 AND shop_id = $2`
	if lock {
		query += ` FOR UPDATE`
	}
	var o Order
	var status string
	err := db.QueryRow(ctx, query, orderID, shopID).Scan(
		&o.ID, &o.ShopID, &o.Number, &status, &o.BuyerName, &o.BuyerPhone, &o.DeliveryAddress,
		&o.Total, &o.CancelReason, &o.CreatedAt, &o.StatusChangedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, fmt.Errorf("load order: %w", err)
	}
	o.Status = Status(status)
	o.CreatedAt = o.CreatedAt.UTC()
	o.StatusChangedAt = o.StatusChangedAt.UTC()

	if o.Items, err = loadItems(ctx, db, orderID, itemOrder); err != nil {
		return Order{}, err
	}
	if o.History, err = loadHistory(ctx, db, orderID); err != nil {
		return Order{}, err
	}
	return o, nil
}

func loadItems(ctx context.Context, db database.DBTX, orderID int64, orderBy string) ([]Item, error) {
	rows, err := db.Query(ctx,
		`SELECT id, product_id, title_snapshot, price_snapshot, quantity
		   FROM order_items WHERE order_id = $1 ORDER BY `+orderBy, orderID)
	if err != nil {
		return nil, fmt.Errorf("load order items: %w", err)
	}
	defer rows.Close()

	items := []Item{}
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.ProductID, &it.Title, &it.Price, &it.Quantity); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load order items: %w", err)
	}
	return items, nil
}

func loadHistory(ctx context.Context, db database.DBTX, orderID int64) ([]HistoryEntry, error) {
	rows, err := db.Query(ctx,
		`SELECT from_status::text, to_status::text, reason, created_at
		   FROM order_status_history WHERE order_id = $1 ORDER BY created_at, id`, orderID)
	if err != nil {
		return nil, fmt.Errorf("load order history: %w", err)
	}
	defer rows.Close()

	history := []HistoryEntry{}
	for rows.Next() {
		var from *string
		var to string
		var h HistoryEntry
		if err := rows.Scan(&from, &to, &h.Reason, &h.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan order history: %w", err)
		}
		if from != nil {
			s := Status(*from)
			h.FromStatus = &s
		}
		h.ToStatus = Status(to)
		h.CreatedAt = h.CreatedAt.UTC()
		history = append(history, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load order history: %w", err)
	}
	return history, nil
}
