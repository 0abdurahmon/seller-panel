// Package stock changes product stock (see D5).
package stock

// UpdateRequest: new absolute stock and the stock seen before editing; both required.
type UpdateRequest struct {
	Stock         *int `json:"stock"`
	ExpectedStock *int `json:"expected_stock"`
}
