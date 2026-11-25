// Package realtime fans out events to the open streams of one shop at a time.
// It is in-process: several backend instances would need a shared bus instead.
package realtime

import (
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
)

var ErrTooManyStreams = errors.New("too many open streams for this shop")

// Message is one event ready to write to a stream.
type Message struct {
	Name string
	Data []byte
}

// Hub keeps the open subscriptions of every shop.
type Hub struct {
	mu         sync.Mutex
	maxPerShop int
	buffer     int
	shops      map[int64]map[*Subscription]struct{}
}

// Subscription receives the events of one shop until it is closed or dropped.
type Subscription struct {
	hub    *Hub
	shopID int64
	ch     chan Message
}

// NewHub returns a hub allowing maxPerShop open streams per shop, each buffering up to buffer events.
func NewHub(maxPerShop, buffer int) *Hub {
	return &Hub{maxPerShop: maxPerShop, buffer: buffer, shops: make(map[int64]map[*Subscription]struct{})}
}

// Subscribe opens a stream for a shop. It fails with ErrTooManyStreams at the limit.
func (h *Hub) Subscribe(shopID int64) (*Subscription, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	subs := h.shops[shopID]
	if len(subs) >= h.maxPerShop {
		return nil, ErrTooManyStreams
	}
	if subs == nil {
		subs = make(map[*Subscription]struct{})
		h.shops[shopID] = subs
	}
	s := &Subscription{hub: h, shopID: shopID, ch: make(chan Message, h.buffer)}
	subs[s] = struct{}{}
	return s, nil
}

// Publish sends an event to every stream of the shop. A stream that cannot keep up is dropped:
// its channel closes and the client reconnects and reloads. Publish never blocks.
func (h *Hub) Publish(shopID int64, name string, data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		slog.Error("realtime event not encoded", "event", name, "error", err)
		return
	}
	msg := Message{Name: name, Data: raw}

	h.mu.Lock()
	defer h.mu.Unlock()
	subs := h.shops[shopID]
	for s := range subs {
		select {
		case s.ch <- msg:
		default:
			delete(subs, s)
			close(s.ch)
		}
	}
	if len(subs) == 0 {
		delete(h.shops, shopID)
	}
}

// Messages returns the events of the subscription. The channel closes when the subscription ends.
func (s *Subscription) Messages() <-chan Message {
	return s.ch
}

// Close ends the subscription. Calling it again, or after the hub dropped it, does nothing.
func (s *Subscription) Close() {
	h := s.hub
	h.mu.Lock()
	defer h.mu.Unlock()
	subs, ok := h.shops[s.shopID]
	if !ok {
		return
	}
	if _, ok := subs[s]; !ok {
		return
	}
	delete(subs, s)
	if len(subs) == 0 {
		delete(h.shops, s.shopID)
	}
	close(s.ch)
}
