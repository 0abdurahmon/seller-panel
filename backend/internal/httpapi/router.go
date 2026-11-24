package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/balance"
	"github.com/0abdurahmon/seller-panel/backend/internal/catalog"
	"github.com/0abdurahmon/seller-panel/backend/internal/dashboard"
	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
	"github.com/0abdurahmon/seller-panel/backend/internal/order"
	"github.com/0abdurahmon/seller-panel/backend/internal/product"
	"github.com/0abdurahmon/seller-panel/backend/internal/shop"
	"github.com/0abdurahmon/seller-panel/backend/internal/stock"
)

// liveShopFor is how long a shop found for a token is remembered, see auth.RequireLiveShop.
const liveShopFor = 30 * time.Second

// Deps are the handlers and services the router is wired with.
type Deps struct {
	Auth    *auth.Handler
	Shop    *shop.Handler
	Catalog *catalog.Handler
	Product *product.Handler
	Tokens  *auth.TokenIssuer
	Stock   *stock.Handler
	Balance *balance.Handler
	Order   *order.Handler

	Dashboard *dashboard.Handler

	ShopLimiter  *httpx.Limiter
	ImageLimiter *httpx.Limiter
	ShopChecker  auth.ShopChecker
	DevEndpoints bool
	Spec         http.Handler
}

// NewRouter registers all routes. Protected routes go behind auth.RequireAuth.
func NewRouter(d Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("POST /auth/register", d.Auth.Register)
	mux.HandleFunc("POST /auth/login", d.Auth.Login)
	mux.HandleFunc("POST /auth/request-code", d.Auth.RequestCode)
	mux.HandleFunc("POST /auth/verify", d.Auth.Verify)

	requireAuth := auth.RequireAuth(d.Tokens)
	limitPerShop := httpx.RateLimit(d.ShopLimiter, func(r *http.Request) (string, bool) {
		p, ok := auth.PrincipalFrom(r.Context())
		return strconv.FormatInt(p.ShopID, 10), ok
	})
	liveShop := func(next http.Handler) http.Handler { return next }
	if d.ShopChecker != nil {
		liveShop = auth.RequireLiveShop(d.ShopChecker, liveShopFor)
	}
	protected := func(h http.HandlerFunc) http.Handler {
		return requireAuth(liveShop(limitPerShop(h)))
	}
	rateLimitImages := httpx.RateLimit(d.ImageLimiter, func(r *http.Request) (string, bool) {
		p, ok := auth.PrincipalFrom(r.Context())
		return strconv.FormatInt(p.ShopID, 10), ok
	})
	// A request that saves an image is expensive, so it has its own per shop limit; JSON edits do not.
	limitImages := func(next http.HandlerFunc) http.HandlerFunc {
		limited := rateLimitImages(next)
		return func(w http.ResponseWriter, r *http.Request) {
			if httpx.IsMultipart(r) {
				limited.ServeHTTP(w, r)
				return
			}
			next(w, r)
		}
	}
	mux.Handle("POST /auth/telegram/link", protected(d.Auth.LinkTelegram))
	mux.Handle("GET /shop", protected(d.Shop.Get))
	mux.Handle("PUT /shop", protected(limitImages(d.Shop.Update)))
	mux.Handle("GET /categories", protected(d.Catalog.List))
	mux.Handle("GET /products", protected(d.Product.List))
	mux.Handle("POST /products", protected(limitImages(d.Product.Create)))
	mux.Handle("GET /products/{id}", protected(d.Product.Get))
	mux.Handle("PUT /products/{id}", protected(limitImages(d.Product.Update)))
	mux.Handle("POST /products/{id}/archive", protected(d.Product.Archive))
	mux.Handle("PATCH /products/{id}/stock", protected(d.Stock.Update))
	mux.Handle("GET /balance", protected(d.Balance.Summary))
	mux.Handle("GET /balance/entries", protected(d.Balance.Entries))
	mux.Handle("GET /orders", protected(d.Order.List))
	mux.Handle("GET /orders/{id}", protected(d.Order.Get))
	mux.Handle("POST /orders/{id}/transition", protected(d.Order.Transition))
	mux.Handle("GET /dashboard/summary", protected(d.Dashboard.Summary))
	if d.Spec != nil {
		mux.Handle("GET /openapi.json", d.Spec)
	}
	if d.DevEndpoints {
		mux.Handle("POST /dev/simulate-order", protected(d.Order.Simulate))
	}
	return httpx.JSONErrors(mux)
}

func health(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
