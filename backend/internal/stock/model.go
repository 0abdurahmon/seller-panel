// Package stock changes product stock and computes the stock label (see D5).
package stock

import "time"

// Label is the stock state shown next to a product.
type Label string

const (
	InStock    Label = "in_stock"
	LowStock   Label = "low_stock"
	OutOfStock Label = "out_of_stock"
)

// LabelFor: out_of_stock at 0, low_stock up to threshold, otherwise in_stock.
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

// Level is a product's stock with derived available and label.
type Level struct {
	ProductID int64     `json:"product_id"`
	Stock     int       `json:"stock"`
	Reserved  int       `json:"reserved"`
	Available int       `json:"available"`
	Label     Label     `json:"stock_label"`
	UpdatedAt time.Time `json:"updated_at"`
}
