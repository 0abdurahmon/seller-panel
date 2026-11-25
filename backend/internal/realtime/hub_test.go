package realtime

import (
	"errors"
	"testing"
)

type event struct {
	Hello string `json:"hello"`
}

func TestPublishReachesOnlyTheShop(t *testing.T) {
	h := NewHub(5, 4)
	mine, _ := h.Subscribe(1)
	other, _ := h.Subscribe(2)
	defer mine.Close()
	defer other.Close()

	h.Publish(1, "order.created", event{Hello: "a"})
	msg := <-mine.Messages()
	if msg.Name != "order.created" || string(msg.Data) != `{"hello":"a"}` {
		t.Fatalf("message = %+v", msg)
	}
	select {
	case m := <-other.Messages():
		t.Fatalf("other shop got %+v", m)
	default:
	}
}

func TestLimitPerShop(t *testing.T) {
	h := NewHub(2, 4)
	a, _ := h.Subscribe(1)
	b, err := h.Subscribe(1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Subscribe(1); !errors.Is(err, ErrTooManyStreams) {
		t.Fatalf("third stream err = %v, want ErrTooManyStreams", err)
	}
	if _, err := h.Subscribe(2); err != nil {
		t.Fatalf("another shop was refused: %v", err)
	}
	a.Close()
	if _, err := h.Subscribe(1); err != nil {
		t.Fatalf("after a close the shop can open again: %v", err)
	}
	b.Close()
}

func TestSlowSubscriberIsDroppedNotBlocking(t *testing.T) {
	h := NewHub(5, 1)
	slow, _ := h.Subscribe(1)
	h.Publish(1, "a", event{})
	h.Publish(1, "b", event{}) // buffer full: the subscriber is dropped

	first := <-slow.Messages()
	if first.Name != "a" {
		t.Fatalf("first = %q", first.Name)
	}
	if _, ok := <-slow.Messages(); ok {
		t.Fatal("dropped subscriber's channel should be closed")
	}
	slow.Close() // already dropped, must not panic
}

func TestCloseIsIdempotentAndPublishAfterCloseIsSafe(t *testing.T) {
	h := NewHub(5, 1)
	s, _ := h.Subscribe(1)
	s.Close()
	s.Close()
	h.Publish(1, "a", event{})
}
