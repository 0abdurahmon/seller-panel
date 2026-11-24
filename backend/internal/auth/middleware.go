package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
)

type principalKey struct{}

// Principal is the authenticated seller. Handlers read it from the request
// context and never from the body or the URL.
type Principal struct {
	SellerID int64
	ShopID   int64
}

// PrincipalFrom returns the principal set by RequireAuth.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}

// MustPrincipal returns the principal set by RequireAuth. If it is missing it
// writes a 401 and reports false, so handlers can return right away.
func MustPrincipal(w http.ResponseWriter, r *http.Request) (Principal, bool) {
	p, ok := PrincipalFrom(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing principal")
	}
	return p, ok
}

// RequireAuth rejects requests without a valid Bearer token with 401.
func RequireAuth(tokens *TokenIssuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || raw == "" {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
				return
			}
			claims, err := tokens.Parse(raw)
			if err != nil {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
				return
			}
			p := Principal{SellerID: claims.SellerID, ShopID: claims.ShopID}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, p)))
		})
	}
}
