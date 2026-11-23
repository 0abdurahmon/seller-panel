package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Business errors. The handler maps them to HTTP status codes.
var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrValidation         = errors.New("validation failed")
)

const (
	minPasswordLen = 8
	maxPasswordLen = 72 // bcrypt only uses the first 72 bytes
	maxShopNameLen = 120
)

// Service holds the auth rules: validation, password hashing and token issue.
type Service struct {
	store  store
	tokens *TokenIssuer
	// dummyHash is compared on unknown emails so that login takes about the
	// same time whether or not the account exists.
	dummyHash []byte
}

// NewService builds the auth service.
func NewService(s store, tokens *TokenIssuer) (*Service, error) {
	dummy, err := bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("prepare dummy hash: %w", err)
	}
	return &Service{store: s, tokens: tokens, dummyHash: dummy}, nil
}

// Register creates a seller and their shop, then returns a token.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (TokenResponse, error) {
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return TokenResponse{}, err
	}
	if err := validatePassword(req.Password); err != nil {
		return TokenResponse{}, err
	}
	shopName := strings.TrimSpace(req.ShopName)
	if shopName == "" || len(shopName) > maxShopNameLen {
		return TokenResponse{}, fmt.Errorf("%w: shop_name must be 1-%d characters", ErrValidation, maxShopNameLen)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("hash password: %w", err)
	}

	sellerID, shopID, err := s.store.CreateSellerWithShop(ctx, email, string(hash), shopName)
	if err != nil {
		return TokenResponse{}, err
	}
	return s.issue(sellerID, shopID)
}

// Login checks the password and returns a token.
func (s *Service) Login(ctx context.Context, req LoginRequest) (TokenResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	c, err := s.store.FindCredentials(ctx, email)
	if errors.Is(err, ErrNotFound) {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(req.Password))
		return TokenResponse{}, ErrInvalidCredentials
	}
	if err != nil {
		return TokenResponse{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(c.PasswordHash), []byte(req.Password)); err != nil {
		return TokenResponse{}, ErrInvalidCredentials
	}
	return s.issue(c.SellerID, c.ShopID)
}

func (s *Service) issue(sellerID, shopID int64) (TokenResponse, error) {
	token, exp, err := s.tokens.Issue(sellerID, shopID)
	if err != nil {
		return TokenResponse{}, err
	}
	return TokenResponse{Token: token, ExpiresAt: exp}, nil
}

func normalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || len(email) > 254 {
		return "", fmt.Errorf("%w: email is not valid", ErrValidation)
	}
	return email, nil
}

func validatePassword(p string) error {
	if len(p) < minPasswordLen {
		return fmt.Errorf("%w: password must be at least %d characters", ErrValidation, minPasswordLen)
	}
	if len(p) > maxPasswordLen {
		return fmt.Errorf("%w: password must be at most %d bytes", ErrValidation, maxPasswordLen)
	}
	return nil
}
