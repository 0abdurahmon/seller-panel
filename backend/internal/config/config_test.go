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
