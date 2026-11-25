package orderalert

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/order"
)

type memRepository struct {
	chats map[int64]int64
}

func (m memRepository) ChatForShop(_ context.Context, shopID int64) (int64, error) {
	chat, ok := m.chats[shopID]
	if !ok {
		return 0, ErrNoChat
	}
	return chat, nil
}

type sent struct {
	chatID int64
	text   string
}

type fakeSender struct {
	mu    sync.Mutex
	err   error
	calls []sent
	got   chan sent
}

func (f *fakeSender) SendMessage(_ context.Context, chatID int64, text string) error {
	f.mu.Lock()
	f.calls = append(f.calls, sent{chatID, text})
	f.mu.Unlock()
	if f.got != nil {
		f.got <- sent{chatID, text}
	}
	return f.err
}

func (f *fakeSender) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func created(number string, total int64) order.Summary {
	return order.Summary{ID: 1, Number: number, Status: order.StatusNew, BuyerName: "Aziz Karimov", Total: total}
}

// start runs the worker and returns the sender, which reports each message on got.
func start(t *testing.T, repo Repository, sender *fakeSender) (*Alerts, context.CancelFunc) {
	t.Helper()
	a := New(repo, sender, quiet(), 4)
	ctx, cancel := context.WithCancel(context.Background())
	go a.Run(ctx)
	t.Cleanup(cancel)
	return a, cancel
}

func TestNewOrderIsSentToTheShopsChat(t *testing.T) {
	sender := &fakeSender{got: make(chan sent, 1)}
	a, _ := start(t, memRepository{chats: map[int64]int64{7: 555}}, sender)

	a.Publish(7, order.EventOrderCreated, created("#100245", 4500000))
	select {
	case m := <-sender.got:
		if m.chatID != 555 || !strings.Contains(m.text, "#100245") || !strings.Contains(m.text, "4 500 000 so'm") {
			t.Fatalf("message = %+v", m)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no alert sent")
	}
}

func TestOtherEventsAndUnlinkedShopsSendNothing(t *testing.T) {
	sender := &fakeSender{}
	a, _ := start(t, memRepository{chats: map[int64]int64{7: 555}}, sender)

	a.Publish(7, order.EventOrderStatusChanged, order.StatusChange{ID: 1})
	a.Publish(8, order.EventOrderCreated, created("#1", 10))
	time.Sleep(100 * time.Millisecond)
	if n := sender.count(); n != 0 {
		t.Fatalf("%d messages sent, want 0", n)
	}
}

func TestSendFailureDoesNotStopLaterAlerts(t *testing.T) {
	sender := &fakeSender{err: context.DeadlineExceeded, got: make(chan sent, 2)}
	a, _ := start(t, memRepository{chats: map[int64]int64{7: 555}}, sender)

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
	a := New(memRepository{}, &fakeSender{}, quiet(), 1)
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
