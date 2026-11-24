package app

import (
	"context"
	"io"
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
	"github.com/0abdurahmon/seller-panel/backend/internal/telegram"
	"github.com/0abdurahmon/seller-panel/backend/internal/upload"
)

// App is the HTTP handler and the background work that runs next to it.
type App struct {
	Handler http.Handler
	poll    func(ctx context.Context)
}

// Start runs the background work until ctx ends. It returns at once.
func (a *App) Start(ctx context.Context) {
	if a.poll != nil {
		go a.poll(ctx)
	}
}

// NewApp wires every feature. Telegram polling only runs when a bot token is set.
func NewApp(cfg config.Config, pool *pgxpool.Pool, logger *slog.Logger) (*App, error) {
// NewHandler wires the API. The returned Closer releases the upload directory and must be closed after the server stops.
func NewHandler(cfg config.Config, pool *pgxpool.Pool, logger *slog.Logger) (http.Handler, io.Closer, error) {
	tokens, err := auth.NewTokenIssuer(cfg.JWTSecret, cfg.JWTTTL)
	if err != nil {
		return nil, nil, err
	}
	var tg auth.Telegram
	var bot *telegram.Client
	if cfg.TelegramBotToken != "" {
		bot = telegram.NewClient(cfg.TelegramBotToken, cfg.TelegramBotUsername, cfg.TelegramAPIURL)
		tg = bot
	}
	authSvc, err := auth.NewService(auth.NewPgRepository(pool), tokens, tg)
	if err != nil {
		return nil, nil, err
	}
	stockSvc, err := stock.NewService(stock.NewPgRepository(pool), cfg.LowStockThreshold)
	if err != nil {
		return nil, nil, err
	}

	orderSvc, err := order.NewService(order.NewPgRepository(pool), cfg.CommissionPercent)
	if err != nil {
		return nil, nil, err
	}

	dashboardSvc, err := dashboard.NewService(dashboard.NewPgRepository(pool), cfg.BusinessLocation)
	if err != nil {
		return nil, nil, err
	}
	uploadRepo, err := upload.NewFSRepository(cfg.UploadDir)
	if err != nil {
		return nil, nil, err
	}
	uploadSvc := upload.NewService(uploadRepo, cfg.UploadPublicURL)

	ipResolver, err := httpx.NewIPResolver(cfg.TrustedProxies)
	if err != nil {
		_ = uploadRepo.Close()
		return nil, nil, err
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
		Product: product.NewHandler(product.NewService(product.NewPgRepository(pool), productPhotos{uploads: uploadSvc})),
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
	app := &App{Handler: httpx.Logging(logger)(router)}
	if bot != nil {
		app.poll = func(ctx context.Context) { bot.Poll(ctx, logger, authSvc.HandleBotMessage) }
	}
	return httpx.Logging(logger)(router), uploadRepo, nil
}
