// Package stock owns the stock rules of a product: the manual stock change from
// the product list, the stock label (in stock / low stock / out of stock), and
// the reserve, release and commit operations that orders run inside their
// transaction. Products themselves (title, price, status) live in the product
// feature; this package only touches the stock and reserved columns.
package stock

import "time"

// Label is the stock state shown next to a product.
type Label string

const (
	InStock    Label = "in_stock"
	LowStock   Label = "low_stock"
	OutOfStock Label = "out_of_stock"
)

// LabelFor returns the label for the available quantity:
// out_of_stock when available is 0, low_stock when 0 < available <= threshold,
// in_stock otherwise.
func LabelFor(available, lowStockThreshold int) Label {
	switch {
	case available <= 0:
		return OutOfStock
	case available <= lowStockThreshold:
		return LowStock
	default:
		return InStock
	}
}

// Level is the stock of one product as returned by PATCH /products/{id}/stock.
type Level struct {
	ProductID int64     `json:"product_id"`
	Stock     int       `json:"stock"`
	Reserved  int       `json:"reserved"`
	Available int       `json:"available"`
	Label     Label     `json:"stock_label"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpdateRequest is the body of PATCH /products/{id}/stock. Stock is the new
// absolute quantity, not a delta, so a repeated request gives the same result.
// It is a pointer so a missing field is told apart from 0.
type UpdateRequest struct {
	Stock *int `json:"stock"`
}
