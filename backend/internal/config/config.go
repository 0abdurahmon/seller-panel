package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultTrustedProxies = "127.0.0.1/32,::1/128,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"

type Config struct {
	Port              string
	DatabaseURL       string
	JWTSecret         string
	JWTTTL            time.Duration
	CommissionPercent int
	LowStockThreshold int
	ShopRPS           float64
	ShopBurst         int
	LoginAttempts     int
	LoginWindow       time.Duration
	TrustedProxies    []string
}

func Load() (Config, error) {
	cfg := Config{
		Port:              getEnv("BACKEND_PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		CommissionPercent: 10,
		LowStockThreshold: 5,
		ShopRPS:           20,
		ShopBurst:         40,
		LoginAttempts:     5,
		TrustedProxies:    strings.Split(getEnv("TRUSTED_PROXIES", defaultTrustedProxies), ","),
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
	if cfg.ShopRPS, err = getFloat("RATE_LIMIT_SHOP_RPS", cfg.ShopRPS); err != nil {
		return Config{}, err
	}
	if cfg.ShopBurst, err = getInt("RATE_LIMIT_SHOP_BURST", cfg.ShopBurst); err != nil {
		return Config{}, err
	}
	if cfg.LoginAttempts, err = getInt("RATE_LIMIT_LOGIN_ATTEMPTS", cfg.LoginAttempts); err != nil {
		return Config{}, err
	}
	window := getEnv("RATE_LIMIT_LOGIN_WINDOW", "15m")
	if cfg.LoginWindow, err = time.ParseDuration(window); err != nil {
		return Config{}, fmt.Errorf("RATE_LIMIT_LOGIN_WINDOW must be a duration like 15m: %w", err)
	}
	if cfg.ShopRPS <= 0 || cfg.ShopBurst < 1 || cfg.LoginAttempts < 1 || cfg.LoginWindow <= 0 {
		return Config{}, fmt.Errorf("rate limit settings must be positive")
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

func getFloat(key string, fallback float64) (float64, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number: %w", key, err)
	}
	return n, nil
}
