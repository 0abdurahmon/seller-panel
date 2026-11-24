package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimitAnswers429WithShape(t *testing.T) {
	calls := 0
	h := RateLimit(NewLimiter(0.1, 2), func(r *http.Request) (string, bool) { return "k", true })(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls++
			w.WriteHeader(http.StatusNoContent)
		}))

	var last *httptest.ResponseRecorder
	for i := 0; i < 3; i++ {
		last = httptest.NewRecorder()
		h.ServeHTTP(last, httptest.NewRequest(http.MethodGet, "/x", nil))
	}
	if calls != 2 {
		t.Fatalf("handler ran %d times, want 2", calls)
	}
	if last.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", last.Code)
	}
	if last.Header().Get("Retry-After") != "10" {
		t.Fatalf("Retry-After = %q, want 10 (one token at 0.1/s)", last.Header().Get("Retry-After"))
	}
	var e struct{ Code, Message string }
	if err := json.Unmarshal(last.Body.Bytes(), &e); err != nil || e.Code != "rate_limited" || e.Message == "" {
		t.Fatalf("body = %q", last.Body.String())
	}
}

func TestRateLimitSkipsWhenThereIsNoKey(t *testing.T) {
	calls := 0
	h := RateLimit(NewLimiter(0.001, 1), func(*http.Request) (string, bool) { return "", false })(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	for i := 0; i < 5; i++ {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	}
	if calls != 5 {
		t.Fatalf("handler ran %d times, want 5", calls)
	}
}

func TestRetryAfterIsAtLeastOneSecond(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteRateLimited(rec, 10*time.Millisecond)
	if rec.Header().Get("Retry-After") != "1" {
		t.Fatalf("Retry-After = %q, want 1", rec.Header().Get("Retry-After"))
	}
}
