package app

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0abdurahmon/seller-panel/backend/internal/apispec"
	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/balance"
	"github.com/0abdurahmon/seller-panel/backend/internal/catalog"
	"github.com/0abdurahmon/seller-panel/backend/internal/config"
	"github.com/0abdurahmon/seller-panel/backend/internal/dashboard"
	"github.com/0abdurahmon/seller-panel/backend/internal/httpapi"
	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
	"github.com/0abdurahmon/seller-panel/backend/internal/order"
	"github.com/0abdurahmon/seller-panel/backend/internal/product"
	"github.com/0abdurahmon/seller-panel/backend/internal/shop"
	"github.com/0abdurahmon/seller-panel/backend/internal/stock"
	"github.com/0abdurahmon/seller-panel/backend/internal/upload"
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

	orderSvc, err := order.NewService(order.NewPgRepository(pool), cfg.CommissionPercent)
	if err != nil {
		return nil, err
	}

	dashboardSvc, err := dashboard.NewService(dashboard.NewPgRepository(pool), cfg.BusinessLocation)
	if err != nil {
		return nil, err
	}
	uploadRepo, err := upload.NewFSRepository(cfg.UploadDir)
	if err != nil {
		return nil, err
	}
	uploadSvc := upload.NewService(uploadRepo, cfg.UploadPublicURL)

	ipResolver, err := httpx.NewIPResolver(cfg.TrustedProxies)
	if err != nil {
		return nil, err
	}
	loginGuard := httpx.NewLimiterPer(cfg.LoginAttempts, cfg.LoginWindow)

	var spec http.Handler
	if cfg.OpenAPIFile != "" {
		spec = apispec.NewHandler(cfg.OpenAPIFile)
	}

	router := httpapi.NewRouter(httpapi.Deps{
		Auth:    auth.NewHandler(authSvc, loginGuard, ipResolver.Of),
		Shop:    shop.NewHandler(shop.NewService(shop.NewPgRepository(pool))),
		Catalog: catalog.NewHandler(catalog.NewService(catalog.NewPgRepository(pool))),
		Product: product.NewHandler(product.NewService(product.NewPgRepository(pool), uploadSvc), stockLabel),
		Tokens:  tokens,
		Stock:   stock.NewHandler(stockSvc),
		Balance: balance.NewHandler(balance.NewService(balance.NewPgRepository(pool))),
		Order:   order.NewHandler(orderSvc),
		Upload:  upload.NewHandler(uploadSvc),

		Dashboard: dashboard.NewHandler(dashboardSvc),

		ShopLimiter:   httpx.NewLimiter(cfg.ShopRPS, cfg.ShopBurst),
		UploadLimiter: httpx.NewLimiterPer(cfg.UploadsPerMinute, time.Minute),
		DevEndpoints:  cfg.Env == config.EnvDevelopment,
		Spec:          spec,
	})
	return httpx.Logging(logger)(router), nil
}
