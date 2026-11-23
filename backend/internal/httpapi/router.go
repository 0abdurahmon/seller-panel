package httpapi

import (
	"net/http"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
	"github.com/0abdurahmon/seller-panel/backend/internal/stock"
)

// Deps are the handlers and services the router is wired with.
type Deps struct {
	Auth   *auth.Handler
	Tokens *auth.TokenIssuer
	Stock  *stock.Handler
}

// NewRouter registers all routes. Protected routes go behind auth.RequireAuth.
func NewRouter(d Deps) http.Handler {
	protected := auth.RequireAuth(d.Tokens)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("POST /auth/register", d.Auth.Register)
	mux.HandleFunc("POST /auth/login", d.Auth.Login)
	mux.Handle("PATCH /products/{id}/stock", protected(http.HandlerFunc(d.Stock.Update)))
	return mux
}

func health(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
