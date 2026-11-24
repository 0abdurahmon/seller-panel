package app

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/catalog"
	"github.com/0abdurahmon/seller-panel/backend/internal/config"
	"github.com/0abdurahmon/seller-panel/backend/internal/httpapi"
	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
	"github.com/0abdurahmon/seller-panel/backend/internal/product"
	"github.com/0abdurahmon/seller-panel/backend/internal/shop"
	"github.com/0abdurahmon/seller-panel/backend/internal/stock"
)

func NewHandler(cfg config.Config, pool *pgxpool.Pool, logger *slog.Logger) (http.Handler, error) {
	tokens, err := auth.NewTokenIssuer(cfg.JWTSecret, cfg.JWTTTL)
	if err != nil {
		return nil, err
	}
	authSvc, err := auth.NewService(auth.NewPgRepository(pool), tokens)
	if err != nil {
		return nil, err
	}
	stockSvc, err := stock.NewService(stock.NewPgRepository(pool), cfg.LowStockThreshold)
	if err != nil {
		return nil, err
	}
	stockLabel := func(available int) string { return string(stockSvc.Label(available)) }

	ipResolver, err := httpx.NewIPResolver(cfg.TrustedProxies)
	if err != nil {
		return nil, err
	}
	loginGuard := httpx.NewLimiterPer(cfg.LoginAttempts, cfg.LoginWindow)

	router := httpapi.NewRouter(httpapi.Deps{
		Auth:    auth.NewHandler(authSvc, loginGuard, ipResolver.Of),
		Shop:    shop.NewHandler(shop.NewService(shop.NewPgRepository(pool))),
		Catalog: catalog.NewHandler(catalog.NewService(catalog.NewPgRepository(pool))),
		Product: product.NewHandler(product.NewService(product.NewPgRepository(pool)), stockLabel),
		Tokens:  tokens,
		Stock:   stock.NewHandler(stockSvc),

		ShopLimiter: httpx.NewLimiter(cfg.ShopRPS, cfg.ShopBurst),
	})
	return httpx.Logging(logger)(router), nil
}
