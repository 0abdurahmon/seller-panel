package config

import (
	"testing"
	"time"
)

func TestLoadRateLimitDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ShopRPS != 20 || cfg.ShopBurst != 40 || cfg.LoginAttempts != 5 || cfg.LoginWindow != 15*time.Minute {
		t.Fatalf("defaults = %+v", cfg)
	}
	if len(cfg.TrustedProxies) < 2 {
		t.Fatalf("trusted proxies = %v", cfg.TrustedProxies)
	}
}

func TestLoadRateLimitFromEnvironment(t *testing.T) {
	t.Setenv("RATE_LIMIT_SHOP_RPS", "2.5")
	t.Setenv("RATE_LIMIT_SHOP_BURST", "7")
	t.Setenv("RATE_LIMIT_LOGIN_ATTEMPTS", "3")
	t.Setenv("RATE_LIMIT_LOGIN_WINDOW", "1h")
	t.Setenv("TRUSTED_PROXIES", "10.1.0.0/16")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ShopRPS != 2.5 || cfg.ShopBurst != 7 || cfg.LoginAttempts != 3 || cfg.LoginWindow != time.Hour || len(cfg.TrustedProxies) != 1 {
		t.Fatalf("config = %+v", cfg)
	}
}

func TestLoadRejectsBadRateLimitSettings(t *testing.T) {
	cases := map[string]map[string]string{
		"zero rps":        {"RATE_LIMIT_SHOP_RPS": "0"},
		"negative burst":  {"RATE_LIMIT_SHOP_BURST": "-1"},
		"zero attempts":   {"RATE_LIMIT_LOGIN_ATTEMPTS": "0"},
		"bad window":      {"RATE_LIMIT_LOGIN_WINDOW": "soon"},
		"zero window":     {"RATE_LIMIT_LOGIN_WINDOW": "0s"},
		"rps not numeric": {"RATE_LIMIT_SHOP_RPS": "fast"},
	}
	for name, env := range cases {
		t.Run(name, func(t *testing.T) {
			for k, v := range env {
				t.Setenv(k, v)
			}
			if _, err := Load(); err == nil {
				t.Fatal("bad settings accepted")
			}
		})
	}
}

func TestLoadAppEnv(t *testing.T) {
	cfg, err := Load()
	if err != nil || cfg.Env != EnvProduction {
		t.Fatalf("default env = %q, %v; want production so dev endpoints stay off", cfg.Env, err)
	}
	t.Setenv("APP_ENV", "development")
	cfg, err = Load()
	if err != nil || cfg.Env != EnvDevelopment {
		t.Fatalf("env = %q, %v; want development", cfg.Env, err)
	}
	t.Setenv("APP_ENV", "staging")
	if _, err := Load(); err == nil {
		t.Fatal("an unknown APP_ENV was accepted")
	}
}

func TestLoadBusinessTimeZone(t *testing.T) {
	cfg, err := Load()
	if err != nil || cfg.BusinessLocation == nil || cfg.BusinessLocation.String() != "Asia/Tashkent" {
		t.Fatalf("default zone = %v, %v; want Asia/Tashkent", cfg.BusinessLocation, err)
	}
	_, offset := time.Date(2026, 11, 24, 12, 0, 0, 0, cfg.BusinessLocation).Zone()
	if offset != 5*60*60 {
		t.Fatalf("Tashkent offset = %d seconds, want UTC+5", offset)
	}

	t.Setenv("BUSINESS_TIMEZONE", "Europe/Moscow")
	cfg, err = Load()
	if err != nil || cfg.BusinessLocation.String() != "Europe/Moscow" {
		t.Fatalf("zone from the environment = %v, %v; want Europe/Moscow", cfg.BusinessLocation, err)
	}

	t.Setenv("BUSINESS_TIMEZONE", "Mars/Olympus")
	if _, err := Load(); err == nil {
		t.Fatal("an unknown time zone was accepted")
	}
}
