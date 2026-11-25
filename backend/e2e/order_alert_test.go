//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"net/http"
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
		c.PanelURL = "http://panel.test"
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
	msg := bot.waitFor(t, "Новый заказ "+o.Number)
	for _, want := range []string{o.BuyerName, "Состав:", "Phone SKU-", "сум", "Откройте панель продавца"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("alert lacks %q: %q", want, msg)
		}
	}
	var detail struct {
		BuyerPhone      string `json:"buyer_phone"`
		DeliveryAddress string `json:"delivery_address"`
	}
	var list struct {
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
	}
	s.do(http.MethodGet, "/orders", sel.token, nil).decode(&list)
	if len(list.Items) != 1 {
		t.Fatalf("orders = %+v", list)
	}
	s.do(http.MethodGet, fmt.Sprintf("/orders/%d", list.Items[0].ID), sel.token, nil).decode(&detail)
	if detail.BuyerPhone == "" || strings.Contains(msg, detail.BuyerPhone) || strings.Contains(msg, detail.DeliveryAddress) {
		t.Fatalf("the alert must not carry the phone or the address: %q", msg)
	}
	sent := bot.sendRequest(t, msg)
	if sent["parse_mode"] != "HTML" {
		t.Fatalf("parse_mode = %v", sent["parse_mode"])
	}
	want := fmt.Sprintf("map[inline_keyboard:[[map[text:Открыть заказ url:http://panel.test/orders?order=%d]]]]", list.Items[0].ID)
	if got := fmt.Sprint(sent["reply_markup"]); got != want {
		t.Fatalf("reply_markup = %s, want %s", got, want)
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
