package auth

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

type sentMessage struct {
	chatID int64
	text   string
}

type fakeTelegram struct {
	mu   sync.Mutex
	sent []sentMessage
}

func (f *fakeTelegram) BotUsername() string { return "seller_panel_bot" }

func (f *fakeTelegram) SendMessage(_ context.Context, chatID int64, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, sentMessage{chatID: chatID, text: text})
	return nil
}

func (f *fakeTelegram) messages() []sentMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]sentMessage(nil), f.sent...)
}

// lastCode returns the six digits in the last message sent to Telegram.
func (f *fakeTelegram) lastCode(t *testing.T) string {
	t.Helper()
	msgs := f.messages()
	if len(msgs) == 0 {
		t.Fatal("no message was sent")
	}
	code := regexp.MustCompile(`\b\d{6}\b`).FindString(msgs[len(msgs)-1].text)
	if code == "" {
		t.Fatalf("no code in %q", msgs[len(msgs)-1].text)
	}
	return code
}

type otpFixture struct {
	svc    *Service
	repo   *memRepository
	tg     *fakeTelegram
	clock  *time.Time
	seller int64
	email  string
}

const linkedChat = int64(555001)

// newOTPFixture makes a seller whose Telegram chat is already linked.
func newOTPFixture(t *testing.T) otpFixture {
	t.Helper()
	repo := newMemRepository()
	tokens, err := NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tg := &fakeTelegram{}
	svc, err := NewService(repo, tokens, tg)
	if err != nil {
		t.Fatal(err)
	}
	clock := time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return clock }

	email := "otp@example.com"
	sellerID, _, err := repo.CreateSellerWithShop(context.Background(), email, "hash", "Shop")
	if err != nil {
		t.Fatal(err)
	}
	repo.chats[sellerID] = linkedChat
	return otpFixture{svc: svc, repo: repo, tg: tg, clock: &clock, seller: sellerID, email: email}
}

func (f otpFixture) advance(d time.Duration) { *f.clock = f.clock.Add(d) }

func TestRequestCodeSendsSixDigitsToLinkedChat(t *testing.T) {
	f := newOTPFixture(t)
	if err := f.svc.RequestCode(context.Background(), " OTP@example.com "); err != nil {
		t.Fatal(err)
	}
	msgs := f.tg.messages()
	if len(msgs) != 1 || msgs[0].chatID != linkedChat {
		t.Fatalf("messages = %+v", msgs)
	}
	code := f.tg.lastCode(t)
	stored, _ := f.repo.FindLoginCode(context.Background(), f.seller)
	if stored.CodeHash != hashSecret(code) {
		t.Fatal("stored hash does not match the code sent")
	}
	if strings.Contains(stored.CodeHash, code) {
		t.Fatal("the code is stored in plain text")
	}
}

func TestRequestCodeSaysWhenNoChatIsLinked(t *testing.T) {
	f := newOTPFixture(t)
	ctx := context.Background()
	if _, _, err := f.repo.CreateSellerWithShop(ctx, "unlinked@example.com", "hash", "Other"); err != nil {
		t.Fatal(err)
	}
	for _, email := range []string{"nobody@example.com", "unlinked@example.com"} {
		if err := f.svc.RequestCode(ctx, email); !errors.Is(err, ErrTelegramNotLinked) {
			t.Fatalf("%s: err = %v, want ErrTelegramNotLinked", email, err)
		}
	}
	if n := len(f.tg.messages()); n != 0 {
		t.Fatalf("%d messages sent, want 0", n)
	}
}

func TestRequestCodeCooldown(t *testing.T) {
	f := newOTPFixture(t)
	ctx := context.Background()
	if err := f.svc.RequestCode(ctx, f.email); err != nil {
		t.Fatal(err)
	}
	f.advance(codeCooldown - time.Second)
	if err := f.svc.RequestCode(ctx, f.email); err != nil {
		t.Fatal(err)
	}
	if n := len(f.tg.messages()); n != 1 {
		t.Fatalf("%d messages during cooldown, want 1", n)
	}
	f.advance(2 * time.Second)
	if err := f.svc.RequestCode(ctx, f.email); err != nil {
		t.Fatal(err)
	}
	if n := len(f.tg.messages()); n != 2 {
		t.Fatalf("%d messages after cooldown, want 2", n)
	}
}

func TestVerifyCodeIssuesTokenOnce(t *testing.T) {
	f := newOTPFixture(t)
	ctx := context.Background()
	if err := f.svc.RequestCode(ctx, f.email); err != nil {
		t.Fatal(err)
	}
	code := f.tg.lastCode(t)

	res, err := f.svc.VerifyCode(ctx, f.email, code)
	if err != nil {
		t.Fatal(err)
	}
	if res.Token == "" {
		t.Fatal("empty token")
	}
	if _, err := f.svc.VerifyCode(ctx, f.email, code); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("reused code: err = %v, want ErrInvalidCode", err)
	}
}

func TestVerifyCodeLocksAfterMaxAttempts(t *testing.T) {
	f := newOTPFixture(t)
	ctx := context.Background()
	if err := f.svc.RequestCode(ctx, f.email); err != nil {
		t.Fatal(err)
	}
	code := f.tg.lastCode(t)
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	for i := 0; i < codeMaxAttempts; i++ {
		if _, err := f.svc.VerifyCode(ctx, f.email, wrong); !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("wrong code %d: err = %v", i+1, err)
		}
	}
	if _, err := f.svc.VerifyCode(ctx, f.email, code); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("right code after lockout: err = %v, want ErrInvalidCode", err)
	}
}

func TestVerifyCodeExpires(t *testing.T) {
	f := newOTPFixture(t)
	ctx := context.Background()
	if err := f.svc.RequestCode(ctx, f.email); err != nil {
		t.Fatal(err)
	}
	code := f.tg.lastCode(t)
	f.advance(codeTTL)
	if _, err := f.svc.VerifyCode(ctx, f.email, code); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("expired code: err = %v, want ErrInvalidCode", err)
	}
}

func TestVerifyCodeRejectsMalformedAndUnknown(t *testing.T) {
	f := newOTPFixture(t)
	ctx := context.Background()
	if err := f.svc.RequestCode(ctx, f.email); err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"", "12345", "1234567", "12a456", "-23456"} {
		if _, err := f.svc.VerifyCode(ctx, f.email, code); !errors.Is(err, ErrInvalidCode) {
			t.Errorf("code %q: err = %v, want ErrInvalidCode", code, err)
		}
	}
	if _, err := f.svc.VerifyCode(ctx, "nobody@example.com", "123456"); !errors.Is(err, ErrInvalidCode) {
		t.Errorf("unknown email: err = %v, want ErrInvalidCode", err)
	}
}

func TestTelegramLinkFlow(t *testing.T) {
	f := newOTPFixture(t)
	ctx := context.Background()
	f.repo.chats = map[int64]int64{}

	link, err := f.svc.StartTelegramLink(ctx, f.seller)
	if err != nil {
		t.Fatal(err)
	}
	token, ok := strings.CutPrefix(link.URL, "https://t.me/seller_panel_bot?start=")
	if !ok || len(token) < 20 {
		t.Fatalf("link = %q", link.URL)
	}

	if err := f.svc.HandleBotMessage(ctx, 777, "/start "+token); err != nil {
		t.Fatal(err)
	}
	if f.repo.chats[f.seller] != 777 {
		t.Fatalf("chat = %d, want 777", f.repo.chats[f.seller])
	}
	if last := f.tg.messages()[0].text; !strings.Contains(last, "Telegram is linked") {
		t.Fatalf("reply = %q", last)
	}

	if err := f.svc.HandleBotMessage(ctx, 888, "/start "+token); err != nil {
		t.Fatal(err)
	}
	if last := f.tg.messages()[1].text; !strings.Contains(last, "expired or was already used") {
		t.Fatalf("reused link reply = %q", last)
	}
}

func TestTelegramLinkExpires(t *testing.T) {
	f := newOTPFixture(t)
	ctx := context.Background()
	link, err := f.svc.StartTelegramLink(ctx, f.seller)
	if err != nil {
		t.Fatal(err)
	}
	token := strings.TrimPrefix(link.URL, "https://t.me/seller_panel_bot?start=")
	f.advance(linkTTL)
	if err := f.svc.HandleBotMessage(ctx, 999, "/start "+token); err != nil {
		t.Fatal(err)
	}
	if f.repo.chats[f.seller] != linkedChat {
		t.Fatalf("expired link changed the chat to %d", f.repo.chats[f.seller])
	}
}

func TestTelegramChatCannotBelongToTwoSellers(t *testing.T) {
	f := newOTPFixture(t)
	ctx := context.Background()
	other, _, err := f.repo.CreateSellerWithShop(ctx, "second@example.com", "hash", "Second")
	if err != nil {
		t.Fatal(err)
	}
	link, err := f.svc.StartTelegramLink(ctx, other)
	if err != nil {
		t.Fatal(err)
	}
	token := strings.TrimPrefix(link.URL, "https://t.me/seller_panel_bot?start=")
	if err := f.svc.HandleBotMessage(ctx, linkedChat, "/start "+token); err != nil {
		t.Fatal(err)
	}
	if last := f.tg.messages()[0].text; !strings.Contains(last, "already linked to another seller") {
		t.Fatalf("reply = %q", last)
	}
	if f.repo.chats[other] != 0 {
		t.Fatal("second seller got the chat")
	}
}

func TestTelegramIsOptional(t *testing.T) {
	repo := newMemRepository()
	tokens, _ := NewTokenIssuer("test-secret", time.Hour)
	svc, err := NewService(repo, tokens, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := svc.StartTelegramLink(ctx, 1); !errors.Is(err, ErrTelegramUnavailable) {
		t.Errorf("link: err = %v", err)
	}
	if err := svc.RequestCode(ctx, "a@example.com"); !errors.Is(err, ErrTelegramUnavailable) {
		t.Errorf("request code: err = %v", err)
	}
	if _, err := svc.Login(ctx, LoginRequest{Email: "a@example.com", Password: "x"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("password login still works without telegram: err = %v", err)
	}
}

func TestCodeLimitsCanBeConfigured(t *testing.T) {
	f := newOTPFixture(t)
	tokens, err := NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(f.repo, tokens, f.tg, WithCodeLimits(10*time.Second, 2))
	if err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return *f.clock }
	ctx := context.Background()

	if err := svc.RequestCode(ctx, f.email); err != nil {
		t.Fatal(err)
	}
	f.advance(9 * time.Second)
	if err := svc.RequestCode(ctx, f.email); err != nil {
		t.Fatal(err)
	}
	if n := len(f.tg.messages()); n != 1 {
		t.Fatalf("%d messages inside a 10s cooldown, want 1", n)
	}
	f.advance(2 * time.Second)
	if err := svc.RequestCode(ctx, f.email); err != nil {
		t.Fatal(err)
	}
	if n := len(f.tg.messages()); n != 2 {
		t.Fatalf("%d messages after the cooldown, want 2", n)
	}

	code := f.tg.lastCode(t)
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	for i := 0; i < 2; i++ {
		if _, err := svc.VerifyCode(ctx, f.email, wrong); !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("wrong code %d: err = %v", i+1, err)
		}
	}
	if _, err := svc.VerifyCode(ctx, f.email, code); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("right code after 2 wrong tries: err = %v, want ErrInvalidCode", err)
	}
}
