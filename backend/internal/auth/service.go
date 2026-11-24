package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLen = 8
	maxPasswordLen = 72 // bcrypt only uses the first 72 bytes
	maxShopNameLen = 120

	codeDigits      = 6
	codeTTL         = 5 * time.Minute
	codeCooldown    = 60 * time.Second
	codeMaxAttempts = 5
	linkTTL         = 10 * time.Minute
)

// Telegram is what sign-in by code needs from the bot. A nil Telegram disables it.
type Telegram interface {
	BotUsername() string
	SendMessage(ctx context.Context, chatID int64, text string) error
}

// Service holds the auth rules: validation, password hashing, sign-in codes and token issue.
type Service struct {
	repo     Repository
	tokens   *TokenIssuer
	telegram Telegram
	now      func() time.Time
	// dummyHash is compared on unknown emails so that login takes about the
	// same time whether or not the account exists.
	dummyHash []byte
}

// NewService builds the auth service.
func NewService(repo Repository, tokens *TokenIssuer, tg Telegram) (*Service, error) {
	dummy, err := bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("prepare dummy hash: %w", err)
	}
	return &Service{repo: repo, tokens: tokens, telegram: tg, now: time.Now, dummyHash: dummy}, nil
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

	sellerID, shopID, err := s.repo.CreateSellerWithShop(ctx, email, string(hash), shopName)
	if err != nil {
		return TokenResponse{}, err
	}
	return s.issue(sellerID, shopID)
}

// Login checks the password and returns a token.
func (s *Service) Login(ctx context.Context, req LoginRequest) (TokenResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	c, err := s.repo.FindCredentials(ctx, email)
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

// StartTelegramLink returns a one-time link that binds the seller to the bot chat it is opened in.
func (s *Service) StartTelegramLink(ctx context.Context, sellerID int64) (TelegramLinkResponse, error) {
	if s.telegram == nil {
		return TelegramLinkResponse{}, ErrTelegramUnavailable
	}
	raw, err := randomToken(24)
	if err != nil {
		return TelegramLinkResponse{}, err
	}
	expires := s.now().Add(linkTTL)
	if err := s.repo.SaveLinkToken(ctx, sellerID, hashSecret(raw), expires); err != nil {
		return TelegramLinkResponse{}, err
	}
	return TelegramLinkResponse{
		URL:       "https://t.me/" + s.telegram.BotUsername() + "?start=" + raw,
		ExpiresAt: expires,
	}, nil
}

// HandleBotMessage answers a message sent to the bot. "/start <token>" links the chat.
func (s *Service) HandleBotMessage(ctx context.Context, chatID int64, text string) error {
	if s.telegram == nil {
		return ErrTelegramUnavailable
	}
	fields := strings.Fields(text)
	if len(fields) != 2 || fields[0] != "/start" {
		return s.telegram.SendMessage(ctx, chatID,
			"Open the Telegram link from the seller panel to connect this chat for sign-in codes.")
	}

	_, err := s.repo.LinkTelegram(ctx, hashSecret(fields[1]), chatID, s.now())
	switch {
	case errors.Is(err, ErrLinkInvalid):
		return s.telegram.SendMessage(ctx, chatID,
			"This link has expired or was already used. Create a new one in the seller panel.")
	case errors.Is(err, ErrTelegramTaken):
		return s.telegram.SendMessage(ctx, chatID,
			"This Telegram account is already linked to another seller.")
	case err != nil:
		return err
	}
	return s.telegram.SendMessage(ctx, chatID,
		"Telegram is linked. Sign-in codes for your seller panel will be sent here.")
}

// RequestCode sends a sign-in code to the seller's linked chat. It returns nil
// for unknown emails and unlinked sellers too, so the caller cannot tell them apart.
func (s *Service) RequestCode(ctx context.Context, email string) error {
	if s.telegram == nil {
		return ErrTelegramUnavailable
	}
	sellerID, chatID, err := s.repo.FindTelegramChat(ctx, normalizeLookup(email))
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	now := s.now()
	if prev, err := s.repo.FindLoginCode(ctx, sellerID); err == nil && now.Before(prev.CreatedAt.Add(codeCooldown)) {
		return nil
	} else if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}

	code, err := randomDigits(codeDigits)
	if err != nil {
		return err
	}
	if err := s.repo.SaveLoginCode(ctx, sellerID, LoginCode{
		CodeHash:  hashSecret(code),
		CreatedAt: now,
		ExpiresAt: now.Add(codeTTL),
	}); err != nil {
		return err
	}
	return s.telegram.SendMessage(ctx, chatID, fmt.Sprintf(
		"Your sign-in code is %s. It expires in %d minutes. If you did not ask for it, ignore this message.",
		code, int(codeTTL.Minutes())))
}

// VerifyCode checks a sign-in code and returns a token. A code is single use and
// stops working after codeMaxAttempts wrong tries.
func (s *Service) VerifyCode(ctx context.Context, email, code string) (TokenResponse, error) {
	code = strings.TrimSpace(code)
	if len(code) != codeDigits || strings.Trim(code, "0123456789") != "" {
		return TokenResponse{}, ErrInvalidCode
	}
	c, err := s.repo.FindCredentials(ctx, normalizeLookup(email))
	if errors.Is(err, ErrNotFound) {
		return TokenResponse{}, ErrInvalidCode
	}
	if err != nil {
		return TokenResponse{}, err
	}
	lc, err := s.repo.FindLoginCode(ctx, c.SellerID)
	if errors.Is(err, ErrNotFound) {
		return TokenResponse{}, ErrInvalidCode
	}
	if err != nil {
		return TokenResponse{}, err
	}
	if !s.now().Before(lc.ExpiresAt) || lc.Attempts >= codeMaxAttempts {
		if err := s.repo.DeleteLoginCode(ctx, c.SellerID); err != nil {
			return TokenResponse{}, err
		}
		return TokenResponse{}, ErrInvalidCode
	}
	if subtle.ConstantTimeCompare([]byte(hashSecret(code)), []byte(lc.CodeHash)) != 1 {
		if err := s.repo.IncrementLoginAttempts(ctx, c.SellerID); err != nil {
			return TokenResponse{}, err
		}
		return TokenResponse{}, ErrInvalidCode
	}
	if err := s.repo.DeleteLoginCode(ctx, c.SellerID); err != nil {
		return TokenResponse{}, err
	}
	return s.issue(c.SellerID, c.ShopID)
}

// normalizeLookup trims and lower-cases an email the way Register stores it.
func normalizeLookup(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// hashSecret stores codes and link tokens as SHA-256; they are random and short-lived.
func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func randomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func randomDigits(n int) (string, error) {
	limit := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
	v, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return "", fmt.Errorf("random code: %w", err)
	}
	return fmt.Sprintf("%0*d", n, v), nil
}
