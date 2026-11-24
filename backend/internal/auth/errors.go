package auth

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrValidation         = errors.New("validation failed")
	ErrEmailTaken         = errors.New("email already registered")
	ErrNotFound           = errors.New("seller not found")
	ErrInvalidToken       = errors.New("invalid token")
)
