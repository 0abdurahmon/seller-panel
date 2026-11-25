package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // zone names must resolve on any machine, including Windows without a zone database
)

const (
	EnvProduction  = "production"
	EnvDevelopment = "development"
)

// DefaultBusinessTimeZone defines "today" on the dashboard (see D10).
const DefaultBusinessTimeZone = "Asia/Tashkent"

const defaultTrustedProxies = "127.0.0.1/32,::1/128,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"

type Config struct {
	Env                 string
	Port                string
	OpenAPIFile         string
	DatabaseURL         string
	JWTSecret           string
	JWTTTL              time.Duration
	CommissionPercent   int
	ShopRPS             float64
	ShopBurst           int
	LoginAttempts       int
	LoginWindow         time.Duration
	CodeCooldown        time.Duration
	CodeAttempts        int
	StreamsPerShop      int
	MaxRequestsInFlight int
	TrustedProxies      []string
	TelegramBotToken    string
	TelegramBotUsername string
	TelegramAPIURL      string
	BusinessLocation    *time.Location
	UploadDir           string
	UploadPublicURL     string
	ImagesPerMinute     int
	PanelURL            string
}

func Load() (Config, error) {
	cfg := Config{
		Env:                 getEnv("APP_ENV", EnvProduction),
		Port:                getEnv("BACKEND_PORT", "8080"),
		OpenAPIFile:         os.Getenv("OPENAPI_FILE"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		CommissionPercent:   10,
		ShopRPS:             20,
		ShopBurst:           40,
		LoginAttempts:       5,
		TrustedProxies:      strings.Split(getEnv("TRUSTED_PROXIES", defaultTrustedProxies), ","),
		TelegramBotToken:    os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramBotUsername: strings.TrimPrefix(os.Getenv("TELEGRAM_BOT_USERNAME"), "@"),
		TelegramAPIURL:      getEnv("TELEGRAM_API_URL", "https://api.telegram.org"),
		UploadDir:           getEnv("UPLOAD_DIR", "./uploads"),
		UploadPublicURL:     getEnv("UPLOAD_PUBLIC_URL", "/api/uploads"),
		ImagesPerMinute:     20,
		CodeAttempts:        5,
		StreamsPerShop:      5,
		MaxRequestsInFlight: 200,
	}

	if cfg.Env != EnvProduction && cfg.Env != EnvDevelopment {
		return Config{}, fmt.Errorf("APP_ENV must be %q or %q, got %q", EnvProduction, EnvDevelopment, cfg.Env)
	}

	var err error
	ttl := getEnv("JWT_TTL", "24h")
	if cfg.JWTTTL, err = time.ParseDuration(ttl); err != nil {
		return Config{}, fmt.Errorf("JWT_TTL must be a duration like 24h: %w", err)
	}
	if cfg.CommissionPercent, err = getInt("COMMISSION_PERCENT", cfg.CommissionPercent); err != nil {
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
	if cfg.ImagesPerMinute, err = getInt("RATE_LIMIT_IMAGES_PER_MINUTE", cfg.ImagesPerMinute); err != nil {
		return Config{}, err
	}
	if cfg.LoginWindow, err = getDuration("RATE_LIMIT_LOGIN_WINDOW", 15*time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.CodeCooldown, err = getDuration("RATE_LIMIT_CODE_COOLDOWN", 60*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.CodeAttempts, err = getInt("RATE_LIMIT_CODE_ATTEMPTS", cfg.CodeAttempts); err != nil {
		return Config{}, err
	}
	if cfg.StreamsPerShop, err = getInt("RATE_LIMIT_STREAMS_PER_SHOP", cfg.StreamsPerShop); err != nil {
		return Config{}, err
	}
	if cfg.ShopRPS <= 0 || cfg.ShopBurst < 1 || cfg.LoginAttempts < 1 || cfg.LoginWindow <= 0 || cfg.ImagesPerMinute < 1 ||
		cfg.CodeAttempts < 1 || cfg.CodeCooldown < 0 || cfg.StreamsPerShop < 1 {
		return Config{}, fmt.Errorf("rate limit settings must be positive, and the code cooldown must not be negative")
	}
	if cfg.MaxRequestsInFlight, err = getInt("MAX_REQUESTS_IN_FLIGHT", cfg.MaxRequestsInFlight); err != nil {
		return Config{}, err
	}
	if cfg.MaxRequestsInFlight < 1 {
		return Config{}, fmt.Errorf("MAX_REQUESTS_IN_FLIGHT must be at least 1, got %d", cfg.MaxRequestsInFlight)
	}
	if cfg.TelegramBotToken != "" && cfg.TelegramBotUsername == "" {
		return Config{}, fmt.Errorf("TELEGRAM_BOT_USERNAME is required when TELEGRAM_BOT_TOKEN is set")
	}
	zone := getEnv("BUSINESS_TIMEZONE", DefaultBusinessTimeZone)
	if cfg.BusinessLocation, err = time.LoadLocation(zone); err != nil {
		return Config{}, fmt.Errorf("BUSINESS_TIMEZONE must be a zone name like %s: %w", DefaultBusinessTimeZone, err)
	}
	if cfg.PanelURL, err = panelURL(os.Getenv("PANEL_URL")); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// panelURL checks the address the seller opens the panel at, links are built by appending a path to it.
func panelURL(v string) (string, error) {
	if v == "" {
		return "", nil
	}
	u, err := url.Parse(v)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("PANEL_URL must be an address like http://192.168.0.10:3000, got %q", v)
	}
	return strings.TrimRight(v, "/"), nil
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

func getDuration(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration like %s: %w", key, fallback, err)
	}
	return d, nil
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
