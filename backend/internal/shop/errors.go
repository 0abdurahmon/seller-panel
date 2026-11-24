package shop

import "errors"

var (
	ErrValidation = errors.New("validation failed")
	ErrNotFound   = errors.New("shop not found")

	ErrFileTooLarge     = errors.New("logo is too large")
	ErrUnsupportedMedia = errors.New("unsupported logo type")
	ErrBusy             = errors.New("server busy")
)
