package httpx

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

type IPResolver struct {
	trusted []*net.IPNet
}

func NewIPResolver(cidrs []string) (*IPResolver, error) {
	r := &IPResolver{}
	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			return nil, fmt.Errorf("trusted proxy %q: %w", c, err)
		}
		r.trusted = append(r.trusted, n)
	}
	return r, nil
}

func (r *IPResolver) Of(req *http.Request) string {
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		host = req.RemoteAddr
	}
	peer := net.ParseIP(host)
	if peer == nil || !r.isTrusted(peer) {
		return host
	}
	if real := net.ParseIP(strings.TrimSpace(req.Header.Get("X-Real-IP"))); real != nil {
		return real.String()
	}
	return peer.String()
}

func (r *IPResolver) isTrusted(ip net.IP) bool {
	for _, n := range r.trusted {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
