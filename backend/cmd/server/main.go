package main

import (
	"log"
	"net/http"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/auth"
	"github.com/0abdurahmon/seller-panel/backend/internal/config"
	"github.com/0abdurahmon/seller-panel/backend/internal/httpapi"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	tokens, err := auth.NewTokenIssuer(cfg.JWTSecret, cfg.JWTTTL)
	if err != nil {
		log.Fatal(err)
	}
	authSvc, err := auth.NewService(auth.NewMemStore(), tokens)
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: httpapi.NewRouter(httpapi.Deps{
			Auth:   auth.NewHandler(authSvc),
			Tokens: tokens,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on :%s", cfg.Port)
	log.Fatal(srv.ListenAndServe())
}
