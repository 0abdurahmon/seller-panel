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
	"github.com/0abdurahmon/seller-panel/backend/internal/orderalert"
	"github.com/0abdurahmon/seller-panel/backend/internal/product"
	"github.com/0abdurahmon/seller-panel/backend/internal/realtime"
	"github.com/0abdurahmon/seller-panel/backend/internal/shop"
	"github.com/0abdurahmon/seller-panel/backend/internal/stock"
	"github.com/0abdurahmon/seller-panel/backend/internal/telegram"
	"github.com/0abdurahmon/seller-panel/backend/internal/upload"
)

// Events buffered per order stream before a slow client is dropped.
const (
	streamBuffer = 16
	// Unsent order alerts kept before new ones are dropped.
	orderAlertQueue = 100
)

// App is the HTTP handler and the background work that runs next to it.
type App struct {
	Handler    http.Handler
	background []func(ctx context.Context)
	closer     io.Closer
	hub        *realtime.Hub
}

// Start runs the background work until ctx ends. It returns at once.
func (a *App) Start(ctx context.Context) {
	for _, run := range a.background {
		go run(ctx)
	}
}

// CloseStreams ends the order event streams, which would otherwise hold up the server shutdown.
func (a *App) CloseStreams() {
	a.hub.Close()
}

// Close releases the upload directory. Call it after the server stops.
func (a *App) Close() error {
	return a.closer.Close()
}

// NewApp wires every feature. Telegram polling only runs when a bot token is set.
func NewApp(cfg config.Config, pool *pgxpool.Pool, logger *slog.Logger) (*App, error) {
	tokens, err := auth.NewTokenIssuer(cfg.JWTSecret, cfg.JWTTTL)
	if err != nil {
		return nil, err
	}
	var tg auth.Telegram
	var bot *telegram.Client
	if cfg.TelegramBotToken != "" {
		bot = telegram.NewClient(cfg.TelegramBotToken, cfg.TelegramBotUsername, cfg.TelegramAPIURL)
		tg = bot
	}
	authSvc, err := auth.NewService(auth.NewPgRepository(pool), tokens, tg,
		auth.WithCodeLimits(cfg.CodeCooldown, cfg.CodeAttempts))
	if err != nil {
		return nil, err
	}
	stockSvc := stock.NewService(stock.NewPgRepository(pool))

	hub := realtime.NewHub(cfg.StreamsPerShop, streamBuffer)
	var alerts *orderalert.Alerts
	var publisher order.Publisher = hub
	if bot != nil {
		alerts = orderalert.New(orderalert.NewPgRepository(pool), bot, logger, orderAlertQueue, cfg.BusinessLocation, cfg.PanelURL)
		publisher = fanout{hub, alerts}
	}
	orderSvc, err := order.NewService(order.NewPgRepository(pool), cfg.CommissionPercent, publisher)
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
		_ = uploadRepo.Close()
		return nil, err
	}
	loginGuard := httpx.NewLimiterPer(cfg.LoginAttempts, cfg.LoginWindow)

	var spec http.Handler
	if cfg.OpenAPIFile != "" {
		spec = apispec.NewHandler(cfg.OpenAPIFile)
	}

	productHandler := product.NewHandler(product.NewService(product.NewPgRepository(pool), productPhotos{uploads: uploadSvc}))

	router := httpapi.NewRouter(httpapi.Deps{
		Auth:    auth.NewHandler(authSvc, loginGuard, ipResolver.Of),
		Shop:    shop.NewHandler(shop.NewService(shop.NewPgRepository(pool), shopLogos{uploads: uploadSvc})),
		Catalog: catalog.NewHandler(catalog.NewService(catalog.NewPgRepository(pool))),
		Product: productHandler,
		Tokens:  tokens,
		Stock:   stock.NewHandler(stockSvc, productHandler.Load),
		Balance: balance.NewHandler(balance.NewService(balance.NewPgRepository(pool))),
		Order:   order.NewHandler(orderSvc, hub),

		Dashboard: dashboard.NewHandler(dashboardSvc),

		ShopLimiter:  httpx.NewLimiter(cfg.ShopRPS, cfg.ShopBurst),
		ImageLimiter: httpx.NewLimiterPer(cfg.ImagesPerMinute, time.Minute),
		ShopChecker:  authSvc.OwnsShop,
		DevEndpoints: cfg.Env == config.EnvDevelopment,
		Spec:         spec,
		MaxInFlight:  cfg.MaxRequestsInFlight,
	})
	application := &App{Handler: httpx.Logging(logger)(router), closer: uploadRepo, hub: hub}
	if bot != nil {
		application.background = append(application.background,
			func(ctx context.Context) { bot.Poll(ctx, logger, authSvc.HandleBotMessage) },
			alerts.Run,
		)
	}
	return application, nil
}
