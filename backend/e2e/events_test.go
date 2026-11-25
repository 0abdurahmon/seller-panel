//go:build e2e

package e2e

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

type sseEvent struct {
	Name string
	Data string
}

// openStream opens GET /orders/events. Events arrive on the channel until the stream ends.
func (s *server) openStream(token string) (*http.Response, <-chan sseEvent, context.CancelFunc) {
	s.t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url+"/orders/events", nil)
	if err != nil {
		cancel()
		s.t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		cancel()
		s.t.Fatal(err)
	}
	events := make(chan sseEvent, 16)
	go func() {
		defer close(events)
		defer res.Body.Close()
		r := bufio.NewReader(res.Body)
		var ev sseEvent
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			switch {
			case strings.HasPrefix(line, "event: "):
				ev.Name = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				ev.Data = strings.TrimPrefix(line, "data: ")
			case line == "" && ev.Name != "":
				events <- ev
				ev = sseEvent{}
			}
		}
	}()
	return res, events, cancel
}

// nextEvent waits for the next event with the given name.
func (s *server) nextEvent(events <-chan sseEvent, name string) sseEvent {
	s.t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				s.t.Fatalf("stream closed before %s", name)
			}
			if ev.Name == name {
				return ev
			}
		case <-deadline:
			s.t.Fatalf("no %s event within 5 seconds", name)
		}
	}
}

func TestOrderEventsJourney(t *testing.T) {
	s := newServer(t)
	sel := s.register()
	other := s.register()
	s.activate(sel.token, s.createProduct(sel.token, s.categoryID(sel.token), 6))

	mine, myEvents, cancelMine := s.openStream(sel.token)
	defer cancelMine()
	if mine.StatusCode != http.StatusOK {
		t.Fatalf("stream status = %d", mine.StatusCode)
	}
	_, otherEvents, cancelOther := s.openStream(other.token)
	defer cancelOther()

	var o orderJSON
	created := s.do(http.MethodPost, "/dev/simulate-order", sel.token, nil)
	s.expect(created, http.StatusCreated, "")
	created.decode(&o)

	var newOrder struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(s.nextEvent(myEvents, "order.created").Data), &newOrder); err != nil {
		t.Fatal(err)
	}
	if newOrder.ID != o.ID || newOrder.Status != "new" {
		t.Fatalf("order.created = %+v, want order %d as new", newOrder, o.ID)
	}

	s.expect(s.transition(sel.token, o.ID, "accepted", ""), http.StatusOK, "")
	var change struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(s.nextEvent(myEvents, "order.status_changed").Data), &change); err != nil {
		t.Fatal(err)
	}
	if change.ID != o.ID || change.Status != "accepted" {
		t.Fatalf("order.status_changed = %+v", change)
	}

	select {
	case ev, ok := <-otherEvents:
		if ok {
			t.Fatalf("another seller's stream got %+v", ev)
		}
	case <-time.After(300 * time.Millisecond):
	}

	var cancels []context.CancelFunc
	for i := 0; i < 4; i++ {
		_, _, cancel := s.openStream(sel.token)
		cancels = append(cancels, cancel)
	}
	defer func() {
		for _, c := range cancels {
			c()
		}
	}()
	sixth, _, cancelSixth := s.openStream(sel.token)
	defer cancelSixth()
	if sixth.StatusCode != http.StatusTooManyRequests || sixth.Header.Get("Retry-After") == "" {
		t.Fatalf("sixth stream status = %d, want 429 with Retry-After", sixth.StatusCode)
	}
}

func TestOrderEventsNeedAToken(t *testing.T) {
	s := newServer(t)
	s.expect(s.do(http.MethodGet, "/orders/events", "", nil), http.StatusUnauthorized, "unauthorized")
}
