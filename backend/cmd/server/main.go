package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/config"
	"github.com/0abdurahmon/seller-panel/backend/internal/database"
	"github.com/0abdurahmon/seller-panel/backend/internal/httpapi"
	"github.com/0abdurahmon/seller-panel/backend/internal/shop"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if err := database.Migrate(cfg.DatabaseURL); err != nil {
		log.Fatal(err)
	}
	pool, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	tokens, err := auth.NewTokenIssuer(cfg.JWTSecret, cfg.JWTTTL)
	if err != nil {
		log.Fatal(err)
	}
	authSvc, err := auth.NewService(auth.NewPgStore(pool), tokens)
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: httpapi.NewRouter(httpapi.Deps{
			Auth:   auth.NewHandler(authSvc),
			Shop:   shop.NewHandler(shop.NewService(shop.NewPgRepository(pool))),
			Tokens: tokens,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on :%s", cfg.Port)
	log.Fatal(srv.ListenAndServe())
}
