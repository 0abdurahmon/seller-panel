package auth

import (
	"context"
	"errors"
	"sync"
)

// ErrEmailTaken is returned when the email is already registered.
var ErrEmailTaken = errors.New("email already registered")

// ErrNotFound is returned when no seller has the given email.
var ErrNotFound = errors.New("seller not found")

// Credentials is what login needs to check a password and issue a token.
type Credentials struct {
	SellerID     int64
	ShopID       int64
	PasswordHash string
}

// store is the persistence the service needs.
// The PostgreSQL implementation waits for the sellers/shops schema to be
// approved; memStore is used until then and in tests.
type store interface {
	CreateSellerWithShop(ctx context.Context, email, passwordHash, shopName string) (sellerID, shopID int64, err error)
	FindCredentials(ctx context.Context, email string) (Credentials, error)
}

// memStore keeps accounts in memory. Data is lost on restart.
type memStore struct {
	mu     sync.Mutex
	nextID int64
	byMail map[string]Credentials
}

// NewMemStore returns an in-memory store.
func NewMemStore() store {
	return &memStore{byMail: make(map[string]Credentials)}
}

func (s *memStore) CreateSellerWithShop(_ context.Context, email, passwordHash, _ string) (int64, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, taken := s.byMail[email]; taken {
		return 0, 0, ErrEmailTaken
	}
	s.nextID++
	c := Credentials{SellerID: s.nextID, ShopID: s.nextID, PasswordHash: passwordHash}
	s.byMail[email] = c
	return c.SellerID, c.ShopID, nil
}

func (s *memStore) FindCredentials(_ context.Context, email string) (Credentials, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.byMail[email]
	if !ok {
		return Credentials{}, ErrNotFound
	}
	return c, nil
}
