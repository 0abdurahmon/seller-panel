package auth

import "time"

// RegisterRequest creates a seller and their shop in one step.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	ShopName string `json:"shop_name"`
}

// LoginRequest is the body of POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// TokenResponse is returned by register and login.
type TokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// RequestCodeRequest is the body of POST /auth/request-code.
type RequestCodeRequest struct {
	Email string `json:"email"`
}

// VerifyRequest is the body of POST /auth/verify.
type VerifyRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// TelegramLinkResponse is returned by POST /auth/telegram/link.
type TelegramLinkResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}
