package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIPResolver(t *testing.T) {
	r, err := NewIPResolver([]string{"127.0.0.1/32", "172.16.0.0/12", " ", "::1/128"})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, remote, realIP, want string
	}{
		{"direct client", "203.0.113.9:5555", "", "203.0.113.9"},
		{"direct client cannot spoof", "203.0.113.9:5555", "1.2.3.4", "203.0.113.9"},
		{"trusted proxy passes the real IP", "172.18.0.5:41000", "198.51.100.7", "198.51.100.7"},
		{"trusted proxy without header", "172.18.0.5:41000", "", "172.18.0.5"},
		{"trusted proxy with a junk header", "172.18.0.5:41000", "not-an-ip", "172.18.0.5"},
		{"loopback proxy", "127.0.0.1:80", " 198.51.100.8 ", "198.51.100.8"},
		{"ipv6 loopback proxy", "[::1]:80", "198.51.100.9", "198.51.100.9"},
		{"remote without a port", "203.0.113.10", "", "203.0.113.10"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = c.remote
		if c.realIP != "" {
			req.Header.Set("X-Real-IP", c.realIP)
		}
		if got := r.Of(req); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestIPResolverRejectsBadCIDR(t *testing.T) {
	if _, err := NewIPResolver([]string{"10.0.0.0/99"}); err == nil {
		t.Fatal("bad CIDR accepted")
	}
}
