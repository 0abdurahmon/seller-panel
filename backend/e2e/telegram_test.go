//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/config"
)

const fakeBotToken = "e2e-bot-token"

// fakeBot is a Telegram Bot API stand-in: it hands out queued updates and records sent messages.
type fakeBot struct {
	url     string
	mu      sync.Mutex
	pending []map[string]any
	sent    []string
	bodies  []map[string]any
}

func newFakeBot(t *testing.T) *fakeBot {
	t.Helper()
	b := &fakeBot{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()
		switch r.URL.Path {
		case "/bot" + fakeBotToken + "/getUpdates":
			result := b.pending
			b.pending = nil
			if len(result) == 0 {
				time.Sleep(20 * time.Millisecond) // keep the poll loop from spinning
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": result})
		case "/bot" + fakeBotToken + "/sendMessage":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			text, _ := body["text"].(string)
			b.sent = append(b.sent, text)
			b.bodies = append(b.bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": map[string]any{"message_id": 1}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	b.url = srv.URL
	return b
}

func (b *fakeBot) queue(chatID int64, text string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pending = append(b.pending, map[string]any{
		"update_id": len(b.sent) + len(b.pending) + 1,
		"message":   map[string]any{"chat": map[string]any{"id": chatID}, "text": text},
	})
}

func (b *fakeBot) messages() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.sent...)
}

// sendRequest returns the whole sendMessage request that carried text.
func (b *fakeBot) sendRequest(t *testing.T, text string) map[string]any {
	t.Helper()
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, body := range b.bodies {
		if body["text"] == text {
			return body
		}
	}
	t.Fatalf("no request with the text %q", text)
	return nil
}

// waitFor returns the first message containing want, or fails after a few seconds.
func (b *fakeBot) waitFor(t *testing.T, want string) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, m := range b.messages() {
			if strings.Contains(m, want) {
				return m
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no bot message containing %q, got %q", want, b.messages())
	return ""
}

func TestTelegramSignInJourney(t *testing.T) {
	bot := newFakeBot(t)
	s := newServer(t, func(c *config.Config) {
		c.TelegramBotToken = fakeBotToken
		c.TelegramBotUsername = "panel_bot"
		c.TelegramAPIURL = bot.url
	})
	ctx, stop := context.WithCancel(context.Background())
	t.Cleanup(stop)
	s.app.Start(ctx)

	sel := s.register()
	login := s.login(sel.email, sel.password)
	s.expect(login, http.StatusOK, "")
	var tok struct {
		Token string `json:"token"`
	}
	login.decode(&tok)

	s.expect(s.do(http.MethodPost, "/auth/telegram/link", "", nil), http.StatusUnauthorized, "unauthorized")
	link := s.do(http.MethodPost, "/auth/telegram/link", tok.Token, nil)
	s.expect(link, http.StatusCreated, "")
	var linkBody struct {
		URL string `json:"url"`
	}
	link.decode(&linkBody)
	linkToken, ok := strings.CutPrefix(linkBody.URL, "https://t.me/panel_bot?start=")
	if !ok {
		t.Fatalf("link = %q", linkBody.URL)
	}

	chatID := time.Now().UnixNano()%1_000_000_000 + 1_000_000 // unique per run, the database is reused
	bot.queue(chatID, "/start "+linkToken)
	bot.waitFor(t, "Telegram is linked")

	s.expect(s.do(http.MethodPost, "/auth/request-code", "", map[string]string{"email": sel.email}), http.StatusAccepted, "")
	code := regexp.MustCompile(`\b\d{6}\b`).FindString(bot.waitFor(t, "sign-in code"))

	s.expect(s.do(http.MethodPost, "/auth/request-code", "", map[string]string{"email": "nobody@example.com"}),
		http.StatusConflict, "telegram_not_linked")
	verify := s.do(http.MethodPost, "/auth/verify", "", map[string]string{"email": sel.email, "code": code})
	s.expect(verify, http.StatusOK, "")
	var otpTok struct {
		Token string `json:"token"`
	}
	verify.decode(&otpTok)
	if shop := s.do(http.MethodGet, "/shop", otpTok.Token, nil); !strings.Contains(string(shop.body), sel.shopName) {
		t.Fatalf("shop with code token = %s", shop.body)
	}

	s.expect(s.do(http.MethodPost, "/auth/verify", "", map[string]string{"email": sel.email, "code": code}), http.StatusUnauthorized, "invalid_code")
	for _, m := range bot.messages() {
		if strings.Contains(m, "nobody@example.com") {
			t.Fatalf("message sent for an unknown email: %q", m)
		}
	}
}

func TestSignInCodeIsUnavailableWithoutTelegram(t *testing.T) {
	s := newServer(t)
	s.expect(s.do(http.MethodPost, "/auth/request-code", "", map[string]string{"email": "a@example.com"}),
		http.StatusServiceUnavailable, "telegram_unavailable")
	s.expect(s.do(http.MethodPost, "/auth/verify", "", map[string]string{"email": "a@example.com", "code": "123456"}),
		http.StatusUnauthorized, "invalid_code")
}
