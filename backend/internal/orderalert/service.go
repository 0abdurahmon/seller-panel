// Package orderalert sends a Telegram message to the seller for each new order.
// It is an order.Publisher: Publish only queues the alert, so a slow Telegram never delays a request.
package orderalert

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/order"
)

const (
	sendTimeout = 10 * time.Second
	// maxLines keeps a large order far below the 4096 characters Telegram accepts.
	maxLines   = 10
	buttonText = "Открыть заказ"
)

// Sender posts HTML formatted text to a Telegram chat, with a button that opens buttonURL unless it is empty.
type Sender interface {
	SendHTML(ctx context.Context, chatID int64, text, buttonText, buttonURL string) error
}

type alert struct {
	shopID  int64
	summary order.Summary
}

// Alerts queues order alerts and sends them one at a time.
type Alerts struct {
	repo     Repository
	sender   Sender
	logger   *slog.Logger
	queue    chan alert
	location *time.Location
	panelURL string
}

// New returns alerts that keep up to queueSize unsent messages. Run must be started for them to go out.
// Times are written in location, and an empty panelURL leaves the button out.
func New(repo Repository, sender Sender, logger *slog.Logger, queueSize int, location *time.Location, panelURL string) *Alerts {
	return &Alerts{repo: repo, sender: sender, logger: logger, queue: make(chan alert, queueSize), location: location, panelURL: panelURL}
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
	lines, err := a.repo.Lines(ctx, al.shopID, al.summary.ID)
	if err != nil {
		a.logger.Error("order alert: items not read, sending without them", "order", al.summary.Number, "error", err)
		lines = nil
	}
	var link string
	if a.panelURL != "" {
		link = a.panelURL + "/orders?order=" + strconv.FormatInt(al.summary.ID, 10)
	}
	sendCtx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	if err := a.sender.SendHTML(sendCtx, chat, a.message(al.summary, lines), buttonText, link); err != nil {
		a.logger.Error("order alert not sent", "order", al.summary.Number, "error", err)
	}
}

// message writes the alert in Telegram HTML, in Russian like the panel. Every value from data is escaped.
func (a *Alerts) message(s order.Summary, lines []Line) string {
	var b strings.Builder
	fmt.Fprintf(&b, "🛒 <b>Новый заказ %s</b>\n%s\n\n👤 Покупатель: %s\n\n",
		html.EscapeString(s.Number), s.CreatedAt.In(a.location).Format("02.01.2006, 15:04"), html.EscapeString(s.BuyerName))
	if len(lines) > 0 {
		b.WriteString("📦 Состав:\n")
		for i, l := range lines[:min(len(lines), maxLines)] {
			fmt.Fprintf(&b, "%d. %s\n   %d × %s = %s сум\n", i+1, html.EscapeString(l.Title),
				l.Quantity, formatSom(l.Price), formatSom(l.Price*int64(l.Quantity)))
		}
		if rest := len(lines) - maxLines; rest > 0 {
			fmt.Fprintf(&b, "и ещё %d\n", rest)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "💰 <b>Итого: %s сум</b>\n\nОткройте панель продавца, чтобы принять заказ.", formatSom(s.Total))
	return b.String()
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
