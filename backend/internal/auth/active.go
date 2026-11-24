package auth

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
)

// ShopChecker says whether a seller still owns the shop named in a token.
type ShopChecker func(ctx context.Context, sellerID, shopID int64) (bool, error)

const maxRememberedShops = 10_000

// RequireLiveShop answers 401 to a valid token whose seller or shop no longer exists, for example after the
// database was reset, so a client signs in again instead of getting empty lists and a 500 on every write.
// It runs after RequireAuth. A shop that was found is remembered for ttl, so the check is not a query per request;
// one that was not found is checked again every time.
func RequireLiveShop(check ShopChecker, ttl time.Duration) func(http.Handler) http.Handler {
	return requireLiveShop(check, ttl, time.Now)
}

func requireLiveShop(check ShopChecker, ttl time.Duration, now func() time.Time) func(http.Handler) http.Handler {
	var mu sync.Mutex
	until := map[Principal]time.Time{}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := MustPrincipal(w, r)
			if !ok {
				return
			}
			mu.Lock()
			expires, known := until[p]
			mu.Unlock()
			if !known || !now().Before(expires) {
				live, err := check(r.Context(), p.SellerID, p.ShopID)
				if err != nil {
					slog.ErrorContext(r.Context(), "check the shop of a token", "error", err)
					httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
					return
				}
				mu.Lock()
				if !live {
					delete(until, p)
					mu.Unlock()
					httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "this account no longer exists, sign in again")
					return
				}
				if len(until) >= maxRememberedShops {
					for k, t := range until {
						if !now().Before(t) {
							delete(until, k)
						}
					}
				}
				until[p] = now().Add(ttl)
				mu.Unlock()
			}
			next.ServeHTTP(w, r)
		})
	}
}
