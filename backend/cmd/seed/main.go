// Command seed fills an empty development database with a demo shop, products and orders.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/app"
	"github.com/0abdurahmon/seller-panel/backend/internal/config"
	"github.com/0abdurahmon/seller-panel/backend/internal/database"
	"github.com/0abdurahmon/seller-panel/backend/internal/order"
	"github.com/0abdurahmon/seller-panel/backend/internal/seed"
	"github.com/0abdurahmon/seller-panel/backend/internal/upload"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Env != config.EnvDevelopment {
		return fmt.Errorf("refusing to run with APP_ENV=%q: the demo login is public, set APP_ENV=%s to seed a development database", cfg.Env, config.EnvDevelopment)
	}
	if cfg.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	if err := database.Migrate(cfg.DatabaseURL); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	uploads, err := upload.NewFSRepository(cfg.UploadDir)
	if err != nil {
		return err
	}
	defer uploads.Close()

	res, err := seed.Run(ctx, pool, seed.Options{
		Photos:            app.NewProductPhotos(upload.NewService(uploads, cfg.UploadPublicURL)),
		Now:               time.Now(),
		Loc:               cfg.BusinessLocation,
		CommissionPercent: cfg.CommissionPercent,
	})
	if err != nil {
		return fmt.Errorf("%w\nThe database may hold half of the demo data. Start over with an empty database (docker compose down -v)", err)
	}
	if res.AlreadySeeded {
		fmt.Printf("The demo seller %s already exists, nothing was changed.\n", res.Email)
		return nil
	}

	fmt.Printf("Seeded %d products and %d orders.\n", res.Products, res.Orders)
	for _, s := range []order.Status{order.StatusNew, order.StatusAccepted, order.StatusPacked, order.StatusHandedToCourier, order.StatusDelivered, order.StatusCancelled} {
		fmt.Printf("  %-18s %d\n", s, res.ByStatus[s])
	}
	fmt.Printf("Log in with %s / %s\n", res.Email, res.Password)
	return nil
}
