package orderalert

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/order"
)

type memRepository struct {
	chats    map[int64]int64
	lines    []Line
	linesErr error
}

func (m memRepository) ChatForShop(_ context.Context, shopID int64) (int64, error) {
	chat, ok := m.chats[shopID]
	if !ok {
		return 0, ErrNoChat
	}
	return chat, nil
}

func (m memRepository) Lines(_ context.Context, shopID, _ int64) ([]Line, error) {
	if _, ok := m.chats[shopID]; !ok {
		return nil, ErrNotFound
	}
	return m.lines, m.linesErr
}

type sent struct {
	chatID     int64
	text       string
	buttonText string
	buttonURL  string
}

type fakeSender struct {
	mu    sync.Mutex
	err   error
	calls []sent
	got   chan sent
}

func (f *fakeSender) SendHTML(_ context.Context, chatID int64, text, buttonText, buttonURL string) error {
	m := sent{chatID, text, buttonText, buttonURL}
	f.mu.Lock()
	f.calls = append(f.calls, m)
	f.mu.Unlock()
	if f.got != nil {
		f.got <- m
	}
	return f.err
}

func (f *fakeSender) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

var tashkent = time.FixedZone("Asia/Tashkent", 5*60*60)

const panelURL = "http://192.168.0.93:3000"

func created(number string, total int64) order.Summary {
	return order.Summary{ID: 42, Number: number, Status: order.StatusNew, BuyerName: "Aziz Karimov", Total: total,
		CreatedAt: time.Date(2026, 11, 25, 8, 20, 0, 0, time.UTC)}
}

// start runs the worker and returns the sender, which reports each message on got.
func start(t *testing.T, repo Repository, sender *fakeSender, panel string) (*Alerts, context.CancelFunc) {
	t.Helper()
	a := New(repo, sender, quiet(), 4, tashkent, panel)
	ctx, cancel := context.WithCancel(context.Background())
	go a.Run(ctx)
	t.Cleanup(cancel)
	return a, cancel
}

func alertFor(t *testing.T, repo memRepository, panel string, s order.Summary) sent {
	t.Helper()
	sender := &fakeSender{got: make(chan sent, 1)}
	a, _ := start(t, repo, sender, panel)
	a.Publish(7, order.EventOrderCreated, s)
	select {
	case m := <-sender.got:
		return m
	case <-time.After(5 * time.Second):
		t.Fatal("no alert sent")
		return sent{}
	}
}

func shopSeven(lines ...Line) memRepository {
	return memRepository{chats: map[int64]int64{7: 555}, lines: lines}
}

func TestNewOrderIsSentToTheShopsChatWithItsDetails(t *testing.T) {
	repo := shopSeven(
		Line{Title: "Apple AirPods 2", Quantity: 3, Price: 1_650_000},
		Line{Title: "Men's cotton T-shirt", Quantity: 2, Price: 120_000},
	)
	m := alertFor(t, repo, panelURL, created("#100459", 5_190_000))

	want := `🛒 <b>Новый заказ #100459</b>
25.11.2026, 13:20

👤 Покупатель: Aziz Karimov

📦 Состав:
1. Apple AirPods 2
   3 × 1 650 000 = 4 950 000 сум
2. Men&#39;s cotton T-shirt
   2 × 120 000 = 240 000 сум

💰 <b>Итого: 5 190 000 сум</b>

Откройте панель продавца, чтобы принять заказ.`
	if m.chatID != 555 || m.text != want {
		t.Fatalf("chat %d, text:\n%s\nwant:\n%s", m.chatID, m.text, want)
	}
	if m.buttonText != "Открыть заказ" || m.buttonURL != panelURL+"/orders?order=42" {
		t.Fatalf("button = %q %q", m.buttonText, m.buttonURL)
	}
}

func TestNoButtonWithoutAPanelAddress(t *testing.T) {
	m := alertFor(t, shopSeven(Line{Title: "Phone", Quantity: 1, Price: 10}), "", created("#1", 10))
	if m.buttonURL != "" {
		t.Fatalf("button = %q", m.buttonURL)
	}
}

func TestBuyerAndProductTextCannotAddMarkup(t *testing.T) {
	s := created("#1", 10)
	s.BuyerName = `<a href="https://evil.example.com">Aziz</a>`
	m := alertFor(t, shopSeven(Line{Title: "<b>Phone</b> & case", Quantity: 1, Price: 10}), panelURL, s)
	if strings.Contains(m.text, "<a ") || strings.Contains(m.text, "<b>Phone") {
		t.Fatalf("markup from data reached the message:\n%s", m.text)
	}
	if !strings.Contains(m.text, "&lt;b&gt;Phone&lt;/b&gt; &amp; case") {
		t.Fatalf("title is not escaped:\n%s", m.text)
	}
}

func TestLongOrderListsTenItemsAndCountsTheRest(t *testing.T) {
	var lines []Line
	for i := 1; i <= 13; i++ {
		lines = append(lines, Line{Title: fmt.Sprintf("Item %02d", i), Quantity: 1, Price: 10})
	}
	m := alertFor(t, shopSeven(lines...), panelURL, created("#1", 130))
	if !strings.Contains(m.text, "10. Item 10") || strings.Contains(m.text, "Item 11") || !strings.Contains(m.text, "и ещё 3\n") {
		t.Fatalf("text:\n%s", m.text)
	}
}

func TestAlertGoesOutWhenTheItemsCannotBeRead(t *testing.T) {
	repo := shopSeven()
	repo.linesErr = errors.New("database is away")
	m := alertFor(t, repo, panelURL, created("#100245", 4500000))
	if strings.Contains(m.text, "Состав") || !strings.Contains(m.text, "#100245") || !strings.Contains(m.text, "4 500 000 сум") {
		t.Fatalf("text:\n%s", m.text)
	}
}

func TestOtherEventsAndUnlinkedShopsSendNothing(t *testing.T) {
	sender := &fakeSender{}
	a, _ := start(t, shopSeven(), sender, panelURL)

	a.Publish(7, order.EventOrderStatusChanged, order.StatusChange{ID: 1})
	a.Publish(8, order.EventOrderCreated, created("#1", 10))
	time.Sleep(100 * time.Millisecond)
	if n := sender.count(); n != 0 {
		t.Fatalf("%d messages sent, want 0", n)
	}
}

func TestSendFailureDoesNotStopLaterAlerts(t *testing.T) {
	sender := &fakeSender{err: context.DeadlineExceeded, got: make(chan sent, 2)}
	a, _ := start(t, shopSeven(), sender, panelURL)

	a.Publish(7, order.EventOrderCreated, created("#1", 10))
	a.Publish(7, order.EventOrderCreated, created("#2", 20))
	for i := 0; i < 2; i++ {
		select {
		case <-sender.got:
		case <-time.After(5 * time.Second):
			t.Fatalf("alert %d not attempted after a failure", i+1)
		}
	}
}

func TestFullQueueDropsWithoutBlocking(t *testing.T) {
	a := New(memRepository{}, &fakeSender{}, quiet(), 1, tashkent, "")
	done := make(chan struct{})
	go func() {
		for i := 0; i < 50; i++ {
			a.Publish(7, order.EventOrderCreated, created("#1", 10))
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Publish blocked on a full queue")
	}
}

func TestFormatSom(t *testing.T) {
	for in, want := range map[int64]string{0: "0", 999: "999", 1000: "1 000", 4500000: "4 500 000", -12345: "-12 345"} {
		if got := formatSom(in); got != want {
			t.Errorf("formatSom(%d) = %q, want %q", in, got, want)
		}
	}
}
