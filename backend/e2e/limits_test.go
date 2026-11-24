//go:build e2e

package e2e

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/0abdurahmon/seller-panel/backend/internal/config"
)

func TestLoginIsBlockedAfterFiveFailures(t *testing.T) {
	s := newServer(t)
	victim, bystander := s.register(), s.register()

	for i := 1; i <= 5; i++ {
		s.expect(s.login(victim.email, "wrong-password"), http.StatusUnauthorized, "invalid_credentials")
	}
	blocked := s.login(victim.email, victim.password)
	s.expect(blocked, http.StatusTooManyRequests, "rate_limited")
	if secs, err := strconv.Atoi(blocked.header.Get("Retry-After")); err != nil || secs < 1 {
		t.Fatalf("Retry-After = %q, want a number of seconds", blocked.header.Get("Retry-After"))
	}

	s.expect(s.login(bystander.email, bystander.password), http.StatusOK, "")
}

func TestSuccessfulLoginClearsTheFailureCount(t *testing.T) {
	s := newServer(t)
	sel := s.register()
	for round := 0; round < 3; round++ {
		for i := 0; i < 4; i++ {
			s.expect(s.login(sel.email, "wrong-password"), http.StatusUnauthorized, "invalid_credentials")
		}
		s.expect(s.login(sel.email, sel.password), http.StatusOK, "")
	}
}

func TestProxyHeaderIsTrustedOnlyFromTrustedProxies(t *testing.T) {
	t.Run("trusted proxy: the header picks the client", func(t *testing.T) {
		s := newServer(t)
		sel := s.register()
		for i := 0; i < 5; i++ {
			s.login(sel.email, "wrong-password", "X-Real-IP", "198.51.100.1")
		}
		s.expect(s.login(sel.email, sel.password, "X-Real-IP", "198.51.100.1"), http.StatusTooManyRequests, "rate_limited")
		s.expect(s.login(sel.email, sel.password, "X-Real-IP", "198.51.100.2"), http.StatusOK, "")
	})

	t.Run("untrusted peer: a spoofed header does not help", func(t *testing.T) {
		s := newServer(t, func(c *config.Config) { c.TrustedProxies = nil })
		sel := s.register()
		for i := 0; i < 5; i++ {
			s.login(sel.email, "wrong-password", "X-Real-IP", "203.0.113."+strconv.Itoa(i+1))
		}
		s.expect(s.login(sel.email, sel.password, "X-Real-IP", "203.0.113.200"), http.StatusTooManyRequests, "rate_limited")
	})
}

func TestProtectedRoutesAreLimitedPerShop(t *testing.T) {
	s := newServer(t, func(c *config.Config) { c.ShopRPS, c.ShopBurst = 0.001, 4 })
	a, b := s.register(), s.register()

	for i := 1; i <= 4; i++ {
		s.expect(s.do(http.MethodGet, "/products", a.token, nil), http.StatusOK, "")
	}
	limited := s.do(http.MethodGet, "/products", a.token, nil)
	s.expect(limited, http.StatusTooManyRequests, "rate_limited")
	if limited.header.Get("Retry-After") == "" {
		t.Fatal("Retry-After missing")
	}

	s.expect(s.do(http.MethodGet, "/products", b.token, nil), http.StatusOK, "")
	s.expect(s.do(http.MethodGet, "/health", "", nil), http.StatusOK, "")
	s.expect(s.login(a.email, a.password), http.StatusOK, "")
}
