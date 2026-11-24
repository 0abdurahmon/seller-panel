package shop

import "errors"

var (
	ErrValidation = errors.New("validation failed")
	ErrNotFound   = errors.New("shop not found")
)
