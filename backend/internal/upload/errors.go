package upload

import "errors"

var (
	ErrValidation      = errors.New("validation failed")
	ErrNotFound        = errors.New("file not found")
	ErrTooLarge        = errors.New("file too large")
	ErrUnsupportedType = errors.New("unsupported file type")
	ErrBusy            = errors.New("server busy")
)
