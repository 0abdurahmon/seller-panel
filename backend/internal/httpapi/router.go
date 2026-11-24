package httpapi

import (
	"net/http"
	"strconv"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/balance"
	"github.com/0abdurahmon/seller-panel/backend/internal/catalog"
	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
	"github.com/0abdurahmon/seller-panel/backend/internal/order"
	"github.com/0abdurahmon/seller-panel/backend/internal/product"
	"github.com/0abdurahmon/seller-panel/backend/internal/shop"
	"github.com/0abdurahmon/seller-panel/backend/internal/stock"
)

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

	ShopLimiter  *httpx.Limiter
	DevEndpoints bool
}

// NewRouter registers all routes. Protected routes go behind auth.RequireAuth.
func NewRouter(d Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("POST /auth/register", d.Auth.Register)
	mux.HandleFunc("POST /auth/login", d.Auth.Login)

	requireAuth := auth.RequireAuth(d.Tokens)
	limitPerShop := httpx.RateLimit(d.ShopLimiter, func(r *http.Request) (string, bool) {
		p, ok := auth.PrincipalFrom(r.Context())
		return strconv.FormatInt(p.ShopID, 10), ok
	})
	protected := func(h http.HandlerFunc) http.Handler {
		return requireAuth(limitPerShop(h))
	}
	mux.Handle("GET /shop", protected(d.Shop.Get))
	mux.Handle("PUT /shop", protected(d.Shop.Update))
	mux.Handle("GET /categories", protected(d.Catalog.List))
	mux.Handle("GET /products", protected(d.Product.List))
	mux.Handle("POST /products", protected(d.Product.Create))
	mux.Handle("GET /products/{id}", protected(d.Product.Get))
	mux.Handle("PUT /products/{id}", protected(d.Product.Update))
	mux.Handle("POST /products/{id}/archive", protected(d.Product.Archive))
	mux.Handle("PATCH /products/{id}/stock", protected(d.Stock.Update))
	mux.Handle("GET /balance", protected(d.Balance.Summary))
	mux.Handle("GET /balance/entries", protected(d.Balance.Entries))
	mux.Handle("GET /orders", protected(d.Order.List))
	mux.Handle("GET /orders/{id}", protected(d.Order.Get))
	mux.Handle("POST /orders/{id}/transition", protected(d.Order.Transition))
	if d.DevEndpoints {
		mux.Handle("POST /dev/simulate-order", protected(d.Order.Simulate))
	}
	return httpx.JSONErrors(mux)
}

func health(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
