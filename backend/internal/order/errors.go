package order

import "errors"

var (
	ErrNotFound          = errors.New("order not found")
	ErrProductNotFound   = errors.New("product not found")
	ErrInsufficientStock = errors.New("not enough available stock")
	ErrInvalidTransition = errors.New("transition not allowed")
	ErrValidation        = errors.New("validation failed")
	ErrNoProducts        = errors.New("no product with available stock to order")
)
