package dashboard

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

func (r *PgRepository) Summary(ctx context.Context, shopID int64, w Window) (Summary, error) {
	var s Summary
	err := r.db.QueryRow(ctx,
		`SELECT count(*) FILTER (WHERE created_at >= $2 AND created_at < $3),
		        count(*) FILTER (WHERE status IN ('new', 'accepted', 'packed')),
		        coalesce(sum(total) FILTER (WHERE status = 'delivered' AND status_changed_at >= $4 AND status_changed_at < $3), 0)::bigint,
		        coalesce(sum(total) FILTER (WHERE status = 'delivered' AND status_changed_at >= $5 AND status_changed_at < $3), 0)::bigint
		   FROM orders
		  WHERE shop_id = $1`,
		shopID, w.TodayStart, w.TodayEnd, w.From7D, w.From30D,
	).Scan(&s.TodayOrders, &s.WaitingForAction, &s.Revenue7D, &s.Revenue30D)
	if err != nil {
		return Summary{}, fmt.Errorf("count orders: %w", err)
	}

	rows, err := r.db.Query(ctx,
		`SELECT oi.product_id,
		        p.title,
		        sum(oi.quantity)::bigint AS quantity_sold,
		        sum(oi.price_snapshot * oi.quantity)::bigint AS revenue
		   FROM orders o
		   JOIN order_items oi ON oi.order_id = o.id
		   JOIN products p ON p.id = oi.product_id AND p.shop_id = o.shop_id
		  WHERE o.shop_id = $1
		    AND o.status = 'delivered'
		    AND o.status_changed_at >= $2
		    AND o.status_changed_at < $3
		  GROUP BY oi.product_id, p.title
		  ORDER BY quantity_sold DESC, revenue DESC, oi.product_id
		  LIMIT 5`,
		shopID, w.From30D, w.TodayEnd)
	if err != nil {
		return Summary{}, fmt.Errorf("top products: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var t TopProduct
		if err := rows.Scan(&t.ProductID, &t.Title, &t.QuantitySold, &t.Revenue); err != nil {
			return Summary{}, fmt.Errorf("scan top product: %w", err)
		}
		s.TopProducts = append(s.TopProducts, t)
	}
	if err := rows.Err(); err != nil {
		return Summary{}, fmt.Errorf("top products: %w", err)
	}
	return s, nil
}
