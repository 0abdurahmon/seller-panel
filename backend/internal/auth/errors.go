package auth

import "errors"

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrValidation          = errors.New("validation failed")
	ErrEmailTaken          = errors.New("email already registered")
	ErrNotFound            = errors.New("seller not found")
	ErrInvalidToken        = errors.New("invalid token")
	ErrInvalidCode         = errors.New("sign-in code is wrong or expired")
	ErrTelegramUnavailable = errors.New("telegram sign-in is not configured")
	ErrLinkInvalid         = errors.New("telegram link is unknown, expired or used")
	ErrTelegramTaken       = errors.New("telegram chat already linked to another seller")
)
