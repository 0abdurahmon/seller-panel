package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims are the values every protected request is scoped by.
type Claims struct {
	SellerID int64 `json:"seller_id"`
	ShopID   int64 `json:"shop_id"`
	jwt.RegisteredClaims
}

// TokenIssuer signs and verifies HS256 JWTs.
type TokenIssuer struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewTokenIssuer returns an issuer. The secret must not be empty.
func NewTokenIssuer(secret string, ttl time.Duration) (*TokenIssuer, error) {
	if secret == "" {
		return nil, errors.New("JWT_SECRET is required")
	}
	return &TokenIssuer{secret: []byte(secret), ttl: ttl, now: time.Now}, nil
}

// Issue returns a signed token and its expiry time.
func (t *TokenIssuer) Issue(sellerID, shopID int64) (string, time.Time, error) {
	now := t.now()
	exp := now.Add(t.ttl)
	claims := Claims{
		SellerID: sellerID,
		ShopID:   shopID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, exp, nil
}

// Parse verifies the signature and expiry and returns the claims.
func (t *TokenIssuer) Parse(raw string) (Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(raw, &claims, func(tok *jwt.Token) (any, error) {
		return t.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(t.now),
	)
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}
