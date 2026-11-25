package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/realtime"
)

func TestServerTimeouts(t *testing.T) {
	srv := newServer(":0", http.NotFoundHandler(), func() {})
	if srv.ReadHeaderTimeout != 5*time.Second || srv.ReadTimeout != time.Minute || srv.IdleTimeout != 2*time.Minute {
		t.Fatalf("read header %s, read %s, idle %s", srv.ReadHeaderTimeout, srv.ReadTimeout, srv.IdleTimeout)
	}
	if srv.WriteTimeout != 0 {
		t.Fatalf("write timeout = %s, it would cut the event streams", srv.WriteTimeout)
	}
}

func serve(t *testing.T, srv *http.Server) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
	return "http://" + ln.Addr().String()
}

func openStream(t *testing.T, url string) *bufio.Reader {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })
	return bufio.NewReader(res.Body)
}

func streamHandler(t *testing.T, hub *realtime.Hub) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sub, err := hub.Subscribe(1)
		if err != nil {
			t.Error(err)
			return
		}
		defer sub.Close()
		realtime.Stream(w, r, sub, time.Time{})
	})
}

func TestReadTimeoutKeepsAStreamOpen(t *testing.T) {
	hub := realtime.NewHub(1, 1)
	srv := newServer("", streamHandler(t, hub), hub.Close)
	srv.ReadTimeout = 100 * time.Millisecond
	stream := openStream(t, serve(t, srv))

	<-time.After(5 * srv.ReadTimeout)
	hub.Publish(1, "order.created", struct{}{})
	line, err := stream.ReadString('\n')
	if err != nil || line != "event: order.created\n" {
		t.Fatalf("line = %q, %v; want the event after the read timeout", line, err)
	}
}

func TestShutdownDoesNotWaitForOpenStreams(t *testing.T) {
	hub := realtime.NewHub(1, 1)
	srv := newServer("", streamHandler(t, hub), hub.Close)
	stream := openStream(t, serve(t, srv))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown with an open stream: %v", err)
	}
	if line, err := stream.ReadString('\n'); !errors.Is(err, io.EOF) {
		t.Fatalf("line = %q, %v; want the end of the stream", line, err)
	}
}
