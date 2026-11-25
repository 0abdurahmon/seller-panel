package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLimitInFlight(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	h := LimitInFlight(2, func(r *http.Request) bool { return r.URL.Path == "/free" })(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/slow" {
				entered <- struct{}{}
				<-release
			}
			w.WriteHeader(http.StatusNoContent)
		}))
	serve := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec
	}

	done := make(chan int, 2)
	for range 2 {
		go func() { done <- serve("/slow").Code }()
		<-entered
	}

	shed := serve("/x")
	if shed.Code != http.StatusServiceUnavailable || shed.Header().Get("Retry-After") != "1" {
		t.Fatalf("status = %d, Retry-After = %q; want 503 and 1", shed.Code, shed.Header().Get("Retry-After"))
	}
	var e struct{ Code, Message string }
	if err := json.Unmarshal(shed.Body.Bytes(), &e); err != nil || e.Code != "busy" || e.Message == "" {
		t.Fatalf("body = %q", shed.Body.String())
	}
	if got := serve("/free").Code; got != http.StatusNoContent {
		t.Fatalf("exempt request = %d, want 204", got)
	}

	close(release)
	for range 2 {
		if got := <-done; got != http.StatusNoContent {
			t.Fatalf("accepted request = %d, want 204", got)
		}
	}
	if got := serve("/x").Code; got != http.StatusNoContent {
		t.Fatalf("after the slots are free = %d, want 204", got)
	}
}

func TestLimitInFlightFreesTheSlotAfterAPanic(t *testing.T) {
	h := LimitInFlight(1, func(*http.Request) bool { return false })(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/panic" {
				panic("boom")
			}
			w.WriteHeader(http.StatusNoContent)
		}))
	func() {
		defer func() { _ = recover() }()
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/panic", nil))
	}()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}
