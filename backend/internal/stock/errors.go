package stock

import "errors"

var (
	ErrValidation    = errors.New("validation failed")
	ErrNotFound      = errors.New("product not found")
	ErrArchived      = errors.New("product is archived")
	ErrStockChanged  = errors.New("stock was changed by someone else")
	ErrBelowReserved = errors.New("stock cannot be lower than reserved")
)
