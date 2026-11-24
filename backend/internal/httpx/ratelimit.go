package httpx

import (
	"math"
	"net/http"
	"strconv"
	"time"
)

func WriteRateLimited(w http.ResponseWriter, retryAfter time.Duration) {
	seconds := int(math.Ceil(retryAfter.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests, try again later")
}

func RateLimit(l *Limiter, key func(*http.Request) (string, bool)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if k, ok := key(r); ok {
				if allowed, wait := l.Allow(k); !allowed {
					WriteRateLimited(w, wait)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
