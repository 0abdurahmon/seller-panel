package httpx

import "net/http"

// LimitInFlight serves at most limit requests at once and answers the others at once with 503 busy.
func LimitInFlight(limit int, exempt func(*http.Request) bool) func(http.Handler) http.Handler {
	slots := make(chan struct{}, limit)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if exempt(r) {
				next.ServeHTTP(w, r)
				return
			}
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
				next.ServeHTTP(w, r)
			default:
				w.Header().Set("Retry-After", "1")
				WriteError(w, http.StatusServiceUnavailable, "busy", "server is busy, try again shortly")
			}
		})
	}
}
