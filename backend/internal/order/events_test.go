package order

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/realtime"
)

type recorded struct {
	shopID int64
	name   string
	data   any
}

type recorder struct {
	mu     sync.Mutex
	events []recorded
}

func (r *recorder) Publish(shopID int64, name string, data any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, recorded{shopID, name, data})
}

func (r *recorder) all() []recorded {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]recorded(nil), r.events...)
}

func newPublishingService(t *testing.T, repo *memRepository, pub Publisher) *Service {
	t.Helper()
	svc, err := NewService(repo, 10, pub)
	if err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC) }
	return svc
}

func TestSimulatePublishesCreated(t *testing.T) {
	repo := newMemRepository()
	repo.AddPricedProduct(1, shopA, 10, 0, "Phone", 5000)
	rec := &recorder{}
	if _, err := newPublishingService(t, repo, rec).Simulate(context.Background(), shopA); err != nil {
		t.Fatal(err)
	}
	evs := rec.all()
	if len(evs) != 1 || evs[0].name != EventOrderCreated || evs[0].shopID != shopA {
		t.Fatalf("events = %+v", evs)
	}
	if s, ok := evs[0].data.(Summary); !ok || s.Status != StatusNew || s.Number == "" {
		t.Fatalf("created data = %+v", evs[0].data)
	}
}

func TestTransitionPublishesStatusChangeAfterCommit(t *testing.T) {
	repo := newMemRepository()
	repo.AddOrder(order(1, shopA, StatusNew, 0))
	rec := &recorder{}
	if _, err := newPublishingService(t, repo, rec).Transition(context.Background(), shopA, 1,
		TransitionRequest{To: StatusAccepted}); err != nil {
		t.Fatal(err)
	}
	evs := rec.all()
	if len(evs) != 1 || evs[0].name != EventOrderStatusChanged {
		t.Fatalf("events = %+v", evs)
	}
	if c, ok := evs[0].data.(StatusChange); !ok || c.ID != 1 || c.Status != StatusAccepted {
		t.Fatalf("status data = %+v", evs[0].data)
	}
}

func TestFailedChangesPublishNothing(t *testing.T) {
	ctx := context.Background()
	repo := newMemRepository()
	repo.AddOrder(order(1, shopA, StatusDelivered, 0))
	rec := &recorder{}
	svc := newPublishingService(t, repo, rec)
	if _, err := svc.Transition(ctx, shopA, 1, TransitionRequest{To: StatusAccepted}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("err = %v", err)
	}
	if _, err := svc.Transition(ctx, shopA, 2, TransitionRequest{To: StatusAccepted}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing order err = %v", err)
	}
	if n := len(rec.all()); n != 0 {
		t.Fatalf("%d events for failed changes, want 0", n)
	}
}

func TestEventsStreamDeliversStatusChange(t *testing.T) {
	repo := newMemRepository()
	repo.AddOrder(order(1, shopA, StatusNew, 0))
	hub := realtime.NewHub(5, 16)
	svc := newPublishingService(t, repo, hub)
	tokens, err := auth.NewTokenIssuer("test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(auth.RequireAuth(tokens)(http.HandlerFunc(NewHandler(svc, hub).Events)))
	defer srv.Close()

	token, _, err := tokens.Issue(7, shopA)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}

	// The headers are only sent after the subscription exists, so the change below is not missed.
	if _, err := svc.Transition(ctx, shopA, 1, TransitionRequest{To: StatusAccepted}); err != nil {
		t.Fatal(err)
	}
	r := bufio.NewReader(res.Body)
	var event, data string
	for event == "" || data == "" {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		switch {
		case strings.HasPrefix(line, "event: "):
			event = strings.TrimSpace(strings.TrimPrefix(line, "event: "))
		case strings.HasPrefix(line, "data: "):
			data = strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		}
	}
	if event != EventOrderStatusChanged {
		t.Fatalf("event = %q", event)
	}
	var c StatusChange
	if err := json.Unmarshal([]byte(data), &c); err != nil || c.Status != StatusAccepted || c.ID != 1 {
		t.Fatalf("data = %s", data)
	}
}

func TestEventsStreamNeedsToken(t *testing.T) {
	hub := realtime.NewHub(5, 16)
	tokens, _ := auth.NewTokenIssuer("test-secret", time.Hour)
	srv := httptest.NewServer(auth.RequireAuth(tokens)(http.HandlerFunc(NewHandler(newPublishingService(t, newMemRepository(), hub), hub).Events)))
	defer srv.Close()
	res, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", res.StatusCode)
	}
}

func TestEventsStreamLimitPerShop(t *testing.T) {
	hub := realtime.NewHub(1, 16)
	tokens, _ := auth.NewTokenIssuer("test-secret", time.Hour)
	srv := httptest.NewServer(auth.RequireAuth(tokens)(http.HandlerFunc(NewHandler(newPublishingService(t, newMemRepository(), hub), hub).Events)))
	defer srv.Close()
	token, _, _ := tokens.Issue(7, shopA)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	open := func() (*http.Response, error) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		return http.DefaultClient.Do(req)
	}
	first, err := open()
	if err != nil {
		t.Fatal(err)
	}
	defer first.Body.Close()
	second, err := open()
	if err != nil {
		t.Fatal(err)
	}
	defer second.Body.Close()
	if second.StatusCode != http.StatusTooManyRequests || second.Header.Get("Retry-After") == "" {
		t.Fatalf("second stream status = %d, want 429 with Retry-After", second.StatusCode)
	}
}
