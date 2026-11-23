package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port              string
	DatabaseURL       string
	JWTSecret         string
	JWTTTL            time.Duration
	CommissionPercent int
	LowStockThreshold int
}

func Load() (Config, error) {
	cfg := Config{
		Port:              getEnv("BACKEND_PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		CommissionPercent: 10,
		LowStockThreshold: 5,
	}

	var err error
	ttl := getEnv("JWT_TTL", "24h")
	if cfg.JWTTTL, err = time.ParseDuration(ttl); err != nil {
		return Config{}, fmt.Errorf("JWT_TTL must be a duration like 24h: %w", err)
	}
	if cfg.CommissionPercent, err = getInt("COMMISSION_PERCENT", cfg.CommissionPercent); err != nil {
		return Config{}, err
	}
	if cfg.LowStockThreshold, err = getInt("LOW_STOCK_THRESHOLD", cfg.LowStockThreshold); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return n, nil
}
