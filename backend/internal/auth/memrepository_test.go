package auth

import (
	"context"
	"sync"
)

type memRepository struct {
	mu     sync.Mutex
	nextID int64
	byMail map[string]Credentials
}

func newMemRepository() *memRepository {
	return &memRepository{byMail: make(map[string]Credentials)}
}

func (r *memRepository) CreateSellerWithShop(_ context.Context, email, passwordHash, _ string) (int64, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, taken := r.byMail[email]; taken {
		return 0, 0, ErrEmailTaken
	}
	r.nextID++
	c := Credentials{SellerID: r.nextID, ShopID: r.nextID, PasswordHash: passwordHash}
	r.byMail[email] = c
	return c.SellerID, c.ShopID, nil
}

func (r *memRepository) FindCredentials(_ context.Context, email string) (Credentials, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.byMail[email]
	if !ok {
		return Credentials{}, ErrNotFound
	}
	return c, nil
}
