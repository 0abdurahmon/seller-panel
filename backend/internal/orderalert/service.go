// Package orderalert sends a Telegram message to the seller for each new order.
// It is an order.Publisher: Publish only queues the alert, so a slow Telegram never delays a request.
package orderalert

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/order"
)

const sendTimeout = 10 * time.Second

// Sender posts a text to a Telegram chat.
type Sender interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
}

type alert struct {
	shopID  int64
	summary order.Summary
}

// Alerts queues order alerts and sends them one at a time.
type Alerts struct {
	repo   Repository
	sender Sender
	logger *slog.Logger
	queue  chan alert
}

// New returns alerts that keep up to queueSize unsent messages. Run must be started for them to go out.
func New(repo Repository, sender Sender, logger *slog.Logger, queueSize int) *Alerts {
	return &Alerts{repo: repo, sender: sender, logger: logger, queue: make(chan alert, queueSize)}
}

// Publish queues an alert for order.created. Other events are ignored. A full queue drops the alert and logs it.
func (a *Alerts) Publish(shopID int64, name string, data any) {
	if name != order.EventOrderCreated {
		return
	}
	s, ok := data.(order.Summary)
	if !ok {
		return
	}
	select {
	case a.queue <- alert{shopID: shopID, summary: s}:
	default:
		a.logger.Warn("order alert dropped, queue full", "order", s.Number)
	}
}

// Run sends queued alerts until ctx ends.
func (a *Alerts) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case al := <-a.queue:
			a.send(ctx, al)
		}
	}
}

func (a *Alerts) send(ctx context.Context, al alert) {
	chat, err := a.repo.ChatForShop(ctx, al.shopID)
	if errors.Is(err, ErrNoChat) {
		return
	}
	if err != nil {
		a.logger.Error("order alert: chat lookup failed", "order", al.summary.Number, "error", err)
		return
	}
	sendCtx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	text := fmt.Sprintf("New order %s from %s: %s so'm. Open the seller panel to accept it.",
		al.summary.Number, al.summary.BuyerName, formatSom(al.summary.Total))
	if err := a.sender.SendMessage(sendCtx, chat, text); err != nil {
		a.logger.Error("order alert not sent", "order", al.summary.Number, "error", err)
	}
}

// formatSom writes an amount in so'm with spaces between the thousands, for example 4 500 000.
func formatSom(amount int64) string {
	digits := strconv.FormatInt(amount, 10)
	sign := ""
	if strings.HasPrefix(digits, "-") {
		sign, digits = "-", digits[1:]
	}
	var b strings.Builder
	for i, c := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(c)
	}
	return sign + b.String()
}
