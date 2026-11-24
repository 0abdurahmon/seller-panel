// Package dashboard builds the home page numbers of a shop (see D9).
package dashboard

import "time"

// Summary is the home page of a shop; money is in so'm.
type Summary struct {
	TodayOrders      int          `json:"today_orders"`
	WaitingForAction int          `json:"waiting_for_action"`
	Revenue7D        int64        `json:"revenue_7d"`
	Revenue30D       int64        `json:"revenue_30d"`
	TopProducts      []TopProduct `json:"top_products"`
}

// TopProduct is a product ranked by units in orders delivered in the last 30 days.
type TopProduct struct {
	ProductID    int64  `json:"product_id"`
	Title        string `json:"title"`
	QuantitySold int64  `json:"quantity_sold"`
	Revenue      int64  `json:"revenue"`
}

// Window holds the time boundaries of one summary. Every window ends at TodayEnd,
// and the bounds are half open: from included, to excluded.
type Window struct {
	TodayStart time.Time
	TodayEnd   time.Time
	From7D     time.Time
	From30D    time.Time
}
