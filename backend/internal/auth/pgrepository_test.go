package auth

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/database/dbtest"
)

// These tests need a real PostgreSQL. Set TEST_DATABASE_URL (a database named *_test) to run them;
// without it they are skipped. They run in their own database, and clear it before each test.
func newPgService(t *testing.T) (*Service, *TokenIssuer) {
	t.Helper()
	return newPgServiceWith(t, nil)
}

func newPgServiceWith(t *testing.T, tg Telegram) (*Service, *TokenIssuer) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool := dbtest.NewPool(t, "auth")
	if _, err := pool.Exec(ctx, `TRUNCATE shops, sellers RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}

	tokens, err := NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(NewPgRepository(pool), tokens, tg)
	if err != nil {
		t.Fatal(err)
	}
	return svc, tokens
}

func TestPgRegisterCreatesSellerAndShop(t *testing.T) {
	svc, tokens := newPgService(t)
	ctx := context.Background()

	reg, err := svc.Register(ctx, RegisterRequest{Email: "pg@example.com", Password: "correct-horse", ShopName: "PG Shop"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	claims, err := tokens.Parse(reg.Token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.SellerID == 0 || claims.ShopID == 0 {
		t.Fatalf("claims = %+v, want non-zero ids", claims)
	}

	login, err := svc.Login(ctx, LoginRequest{Email: "pg@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	loginClaims, _ := tokens.Parse(login.Token)
	if loginClaims.SellerID != claims.SellerID || loginClaims.ShopID != claims.ShopID {
		t.Fatalf("login ids %+v differ from register ids %+v", loginClaims, claims)
	}
}

func TestPgDuplicateEmailIsRejectedByDatabase(t *testing.T) {
	svc, _ := newPgService(t)
	ctx := context.Background()
	req := RegisterRequest{Email: "dup@example.com", Password: "correct-horse", ShopName: "A"}
	if _, err := svc.Register(ctx, req); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Register(ctx, req)
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
}

func TestPgRejectedRegisterCreatesNoExtraShop(t *testing.T) {
	svc, _ := newPgService(t)
	ctx := context.Background()
	if _, err := svc.Register(ctx, RegisterRequest{Email: "one@example.com", Password: "correct-horse", ShopName: "One"}); err != nil {
		t.Fatal(err)
	}
	// The duplicate fails inside the transaction; the shop insert must roll back.
	if _, err := svc.Register(ctx, RegisterRequest{Email: "one@example.com", Password: "correct-horse", ShopName: "Two"}); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
	var shops int
	pool := svc.repo.(*PgRepository).pool
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM shops`).Scan(&shops); err != nil {
		t.Fatal(err)
	}
	if shops != 1 {
		t.Fatalf("shops = %d, want 1 (rolled-back register left a shop behind)", shops)
	}
}

func TestPgTelegramSignInFlow(t *testing.T) {
	tg := &fakeTelegram{}
	svc, tokens := newPgServiceWith(t, tg)
	ctx := context.Background()
	for _, email := range []string{"otp-pg@example.com", "otp-pg-2@example.com"} {
		if _, err := svc.Register(ctx, RegisterRequest{Email: email, Password: "long-enough-pw", ShopName: "Shop"}); err != nil {
			t.Fatal(err)
		}
	}

	link, err := svc.StartTelegramLink(ctx, sellerIDOf(t, svc, "otp-pg@example.com"))
	if err != nil {
		t.Fatal(err)
	}
	token := strings.TrimPrefix(link.URL, "https://t.me/seller_panel_bot?start=")
	const chat = int64(9_001)
	if err := svc.HandleBotMessage(ctx, chat, "/start "+token); err != nil {
		t.Fatal(err)
	}

	other, err := svc.StartTelegramLink(ctx, sellerIDOf(t, svc, "otp-pg-2@example.com"))
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.HandleBotMessage(ctx, chat, "/start "+strings.TrimPrefix(other.URL, "https://t.me/seller_panel_bot?start=")); err != nil {
		t.Fatal(err)
	}
	if last := tg.messages()[len(tg.messages())-1].text; !strings.Contains(last, "already linked to another seller") {
		t.Fatalf("second seller reply = %q", last)
	}

	if err := svc.RequestCode(ctx, "otp-pg@example.com"); err != nil {
		t.Fatal(err)
	}
	res, err := svc.VerifyCode(ctx, "otp-pg@example.com", tg.lastCode(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tokens.Parse(res.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.VerifyCode(ctx, "otp-pg@example.com", tg.lastCode(t)); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("reused code: err = %v", err)
	}
}

func newPgLinkedSeller(t *testing.T) (*Service, *fakeTelegram, string) {
	t.Helper()
	tg := &fakeTelegram{}
	svc, _ := newPgServiceWith(t, tg)
	ctx := context.Background()
	email := "race@example.com"
	if _, err := svc.Register(ctx, RegisterRequest{Email: email, Password: "correct-horse", ShopName: "Race"}); err != nil {
		t.Fatal(err)
	}
	link, err := svc.StartTelegramLink(ctx, sellerIDOf(t, svc, email))
	if err != nil {
		t.Fatal(err)
	}
	_, token, _ := strings.Cut(link.URL, "start=")
	if err := svc.HandleBotMessage(ctx, linkedChat, "/start "+token); err != nil {
		t.Fatal(err)
	}
	return svc, tg, email
}

// parallel runs fn n times at once and returns the errors.
func parallel(n int, fn func() error) []error {
	errs := make([]error, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs[i] = fn()
		}()
	}
	close(start)
	wg.Wait()
	return errs
}

func succeeded(errs []error) int {
	n := 0
	for _, err := range errs {
		if err == nil {
			n++
		}
	}
	return n
}

func TestPgParallelRequestsSendOneCode(t *testing.T) {
	svc, tg, email := newPgLinkedSeller(t)
	before := len(tg.messages())
	errs := parallel(20, func() error { return svc.RequestCode(context.Background(), email) })
	if n := succeeded(errs); n != 1 {
		t.Fatalf("%d requests succeeded, want 1", n)
	}
	for _, err := range errs {
		var cooldown *CooldownError
		if err != nil && !errors.As(err, &cooldown) {
			t.Fatalf("err = %v, want CooldownError", err)
		}
	}
	if n := len(tg.messages()) - before; n != 1 {
		t.Fatalf("%d messages sent, want 1", n)
	}
}

func TestPgParallelCorrectCodeGivesOneToken(t *testing.T) {
	svc, tg, email := newPgLinkedSeller(t)
	ctx := context.Background()
	if err := svc.RequestCode(ctx, email); err != nil {
		t.Fatal(err)
	}
	code := tg.lastCode(t)
	errs := parallel(20, func() error {
		_, err := svc.VerifyCode(ctx, email, code)
		return err
	})
	if n := succeeded(errs); n != 1 {
		t.Fatalf("%d tokens issued for one code, want 1", n)
	}
}

func TestPgParallelWrongCodesCannotPassTheAttemptLimit(t *testing.T) {
	svc, tg, email := newPgLinkedSeller(t)
	ctx := context.Background()
	if err := svc.RequestCode(ctx, email); err != nil {
		t.Fatal(err)
	}
	wrong := "000000"
	if tg.lastCode(t) == wrong {
		wrong = "111111"
	}
	parallel(40, func() error {
		_, err := svc.VerifyCode(ctx, email, wrong)
		return err
	})
	stored, err := svc.repo.FindLoginCode(ctx, sellerIDOf(t, svc, email))
	if err != nil {
		t.Fatal(err)
	}
	if stored.Attempts != codeMaxAttempts {
		t.Fatalf("%d guesses were compared, want %d", stored.Attempts, codeMaxAttempts)
	}
}

func sellerIDOf(t *testing.T, svc *Service, email string) int64 {
	t.Helper()
	c, err := svc.repo.FindCredentials(context.Background(), email)
	if err != nil {
		t.Fatal(err)
	}
	return c.SellerID
}

func TestPgOwnsShop(t *testing.T) {
	svc, tokens := newPgService(t)
	ctx := context.Background()
	a, err := svc.Register(ctx, RegisterRequest{Email: "owns-a@example.com", Password: "correct-horse", ShopName: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Register(ctx, RegisterRequest{Email: "owns-b@example.com", Password: "correct-horse", ShopName: "B"})
	if err != nil {
		t.Fatal(err)
	}
	ca, _ := tokens.Parse(a.Token)
	cb, _ := tokens.Parse(b.Token)

	for name, c := range map[string]struct {
		seller, shop int64
		want         bool
	}{
		"own shop":                     {ca.SellerID, ca.ShopID, true},
		"the other seller's own shop":  {cb.SellerID, cb.ShopID, true},
		"a shop of another seller":     {ca.SellerID, cb.ShopID, false},
		"a shop that does not exist":   {ca.SellerID, ca.ShopID + 1000, false},
		"a seller that does not exist": {ca.SellerID + 1000, ca.ShopID, false},
	} {
		got, err := svc.OwnsShop(ctx, c.seller, c.shop)
		if err != nil || got != c.want {
			t.Errorf("%s: %v, %v; want %v", name, got, err, c.want)
		}
	}
}
