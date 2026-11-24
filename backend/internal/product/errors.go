package product

import "errors"

var (
	ErrValidation = errors.New("validation failed")
	ErrNotFound   = errors.New("product not found")
	ErrSKUTaken   = errors.New("sku already used in this shop")
	ErrArchived   = errors.New("product is archived")
)
