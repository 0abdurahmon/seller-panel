package realtime

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStreamWritesServerSentEvents(t *testing.T) {
	h := NewHub(5, 4)
	subscribed := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sub, err := h.Subscribe(1)
		if err != nil {
			t.Error(err)
			return
		}
		defer sub.Close()
		close(subscribed)
		Stream(w, r, sub, time.Time{})
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if ct := res.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type = %q", ct)
	}
	if res.Header.Get("X-Accel-Buffering") != "no" {
		t.Fatal("proxies must not buffer the stream")
	}

	<-subscribed
	h.Publish(1, "order.status_changed", event{Hello: "x"})
	r := bufio.NewReader(res.Body)
	for _, want := range []string{"event: order.status_changed\n", "data: {\"hello\":\"x\"}\n", "\n"} {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if line != want {
			t.Fatalf("line = %q, want %q", line, want)
		}
	}
}

func TestStreamEndsAtTokenExpiry(t *testing.T) {
	h := NewHub(5, 4)
	done := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sub, _ := h.Subscribe(1)
		defer sub.Close()
		Stream(w, r, sub, time.Now().Add(50*time.Millisecond))
		close(done)
	}))
	defer srv.Close()

	res, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("stream did not end at token expiry")
	}
}
