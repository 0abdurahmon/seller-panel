package auth

import (
	"context"
	"sync"
	"time"
)

type memRepository struct {
	mu         sync.Mutex
	nextID     int64
	byMail     map[string]Credentials
	chats      map[int64]int64
	linkTokens map[string]linkToken
	codes      map[int64]LoginCode
}

func newMemRepository() *memRepository {
	return &memRepository{
		byMail:     make(map[string]Credentials),
		chats:      make(map[int64]int64),
		linkTokens: make(map[string]linkToken),
		codes:      make(map[int64]LoginCode),
	}
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

func (r *memRepository) ShopOwnedBy(_ context.Context, sellerID, shopID int64) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.byMail {
		if c.SellerID == sellerID && c.ShopID == shopID {
			return true, nil
		}
	}
	return false, nil
}

type linkToken struct {
	sellerID  int64
	expiresAt time.Time
}

func (r *memRepository) FindTelegramChat(_ context.Context, email string) (int64, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.byMail[email]
	if !ok || r.chats[c.SellerID] == 0 {
		return 0, 0, ErrNotFound
	}
	return c.SellerID, r.chats[c.SellerID], nil
}

func (r *memRepository) SaveLinkToken(_ context.Context, sellerID int64, tokenHash string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for hash, t := range r.linkTokens {
		if t.sellerID == sellerID {
			delete(r.linkTokens, hash)
		}
	}
	r.linkTokens[tokenHash] = linkToken{sellerID: sellerID, expiresAt: expiresAt}
	return nil
}

func (r *memRepository) LinkTelegram(_ context.Context, tokenHash string, chatID int64, now time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.linkTokens[tokenHash]
	if !ok || !now.Before(t.expiresAt) {
		return 0, ErrLinkInvalid
	}
	for seller, chat := range r.chats {
		if chat == chatID && seller != t.sellerID {
			return 0, ErrTelegramTaken
		}
	}
	delete(r.linkTokens, tokenHash)
	r.chats[t.sellerID] = chatID
	return t.sellerID, nil
}

func (r *memRepository) FindLoginCode(_ context.Context, sellerID int64) (LoginCode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.codes[sellerID]
	if !ok {
		return LoginCode{}, ErrNotFound
	}
	return c, nil
}

func (r *memRepository) SaveLoginCode(_ context.Context, sellerID int64, c LoginCode) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c.Attempts = 0
	r.codes[sellerID] = c
	return nil
}

func (r *memRepository) IncrementLoginAttempts(_ context.Context, sellerID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := r.codes[sellerID]
	c.Attempts++
	r.codes[sellerID] = c
	return nil
}

func (r *memRepository) DeleteLoginCode(_ context.Context, sellerID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.codes, sellerID)
	return nil
}
