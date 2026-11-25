//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/config"
)

func TestOrderAlertJourney(t *testing.T) {
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
	s.expect(s.login(sel.email, sel.password), http.StatusOK, "")
	link := s.do(http.MethodPost, "/auth/telegram/link", sel.token, nil)
	s.expect(link, http.StatusCreated, "")
	var linkBody struct {
		URL string `json:"url"`
	}
	link.decode(&linkBody)
	token, ok := strings.CutPrefix(linkBody.URL, "https://t.me/panel_bot?start=")
	if !ok {
		t.Fatalf("link = %q", linkBody.URL)
	}
	chatID := time.Now().UnixNano()%1_000_000_000 + 2_000_000 // unique per run, the database is reused
	bot.queue(chatID, "/start "+token)
	bot.waitFor(t, "Telegram is linked")

	s.activate(sel.token, s.createProduct(sel.token, s.categoryID(sel.token), 6))
	var o orderJSON
	created := s.do(http.MethodPost, "/dev/simulate-order", sel.token, nil)
	s.expect(created, http.StatusCreated, "")
	created.decode(&o)
	msg := bot.waitFor(t, "New order "+o.Number)
	if !strings.Contains(msg, o.BuyerName) || !regexp.MustCompile(`so'm\. Open the seller panel`).MatchString(msg) {
		t.Fatalf("alert = %q", msg)
	}

	// A seller without a linked chat gets no alert for the same kind of order.
	other := s.register()
	s.activate(other.token, s.createProduct(other.token, s.categoryID(other.token), 6))
	var unlinked orderJSON
	otherOrder := s.do(http.MethodPost, "/dev/simulate-order", other.token, nil)
	s.expect(otherOrder, http.StatusCreated, "")
	otherOrder.decode(&unlinked)
	time.Sleep(300 * time.Millisecond)
	for _, m := range bot.messages() {
		if strings.Contains(m, unlinked.Number) {
			t.Fatalf("unlinked seller's order was announced: %q", m)
		}
	}
}
