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
	if cfg.CodeCooldown != time.Minute || cfg.CodeAttempts != 5 || cfg.StreamsPerShop != 5 {
		t.Fatalf("code and stream defaults = %+v", cfg)
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
	t.Setenv("RATE_LIMIT_CODE_COOLDOWN", "90s")
	t.Setenv("RATE_LIMIT_CODE_ATTEMPTS", "3")
	t.Setenv("RATE_LIMIT_STREAMS_PER_SHOP", "8")
	t.Setenv("TRUSTED_PROXIES", "10.1.0.0/16")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ShopRPS != 2.5 || cfg.ShopBurst != 7 || cfg.LoginAttempts != 3 || cfg.LoginWindow != time.Hour || len(cfg.TrustedProxies) != 1 {
		t.Fatalf("config = %+v", cfg)
	}
	if cfg.CodeCooldown != 90*time.Second || cfg.CodeAttempts != 3 || cfg.StreamsPerShop != 8 {
		t.Fatalf("code and stream config = %+v", cfg)
	}
}

func TestLoadRejectsBadRateLimitSettings(t *testing.T) {
	cases := map[string]map[string]string{
		"zero rps":            {"RATE_LIMIT_SHOP_RPS": "0"},
		"negative burst":      {"RATE_LIMIT_SHOP_BURST": "-1"},
		"zero attempts":       {"RATE_LIMIT_LOGIN_ATTEMPTS": "0"},
		"bad window":          {"RATE_LIMIT_LOGIN_WINDOW": "soon"},
		"zero window":         {"RATE_LIMIT_LOGIN_WINDOW": "0s"},
		"rps not numeric":     {"RATE_LIMIT_SHOP_RPS": "fast"},
		"zero code attempts":  {"RATE_LIMIT_CODE_ATTEMPTS": "0"},
		"negative cooldown":   {"RATE_LIMIT_CODE_COOLDOWN": "-1s"},
		"cooldown not a time": {"RATE_LIMIT_CODE_COOLDOWN": "minute"},
		"zero streams":        {"RATE_LIMIT_STREAMS_PER_SHOP": "0"},
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

func TestLoadOpenAPIFile(t *testing.T) {
	cfg, err := Load()
	if err != nil || cfg.OpenAPIFile != "" {
		t.Fatalf("default = %q, %v; want empty so the endpoint is off", cfg.OpenAPIFile, err)
	}
	t.Setenv("OPENAPI_FILE", "/app/api/openapi.yaml")
	cfg, err = Load()
	if err != nil || cfg.OpenAPIFile != "/app/api/openapi.yaml" {
		t.Fatalf("OpenAPIFile = %q, %v", cfg.OpenAPIFile, err)
	}
}

func TestLoadPanelURL(t *testing.T) {
	cfg, err := Load()
	if err != nil || cfg.PanelURL != "" {
		t.Fatalf("default = %q, %v; want empty so alerts carry no button", cfg.PanelURL, err)
	}
	for in, want := range map[string]string{
		"http://192.168.0.93:3000":   "http://192.168.0.93:3000",
		"https://panel.example.com/": "https://panel.example.com",
	} {
		t.Setenv("PANEL_URL", in)
		if cfg, err = Load(); err != nil || cfg.PanelURL != want {
			t.Fatalf("PANEL_URL %q = %q, %v; want %q", in, cfg.PanelURL, err, want)
		}
	}
	for _, bad := range []string{"192.168.0.93:3000", "ftp://panel.example.com", "http://", "http://host/?a=1", "http://host/#x"} {
		t.Setenv("PANEL_URL", bad)
		if _, err := Load(); err == nil {
			t.Errorf("PANEL_URL %q accepted", bad)
		}
	}
}

func TestLoadUploadSettings(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.UploadDir != "./uploads" || cfg.UploadPublicURL != "/api/uploads" || cfg.ImagesPerMinute != 20 {
		t.Fatalf("defaults = %+v", cfg)
	}

	t.Setenv("UPLOAD_DIR", "/data/files")
	t.Setenv("UPLOAD_PUBLIC_URL", "https://cdn.example.com/u")
	t.Setenv("RATE_LIMIT_IMAGES_PER_MINUTE", "5")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.UploadDir != "/data/files" || cfg.UploadPublicURL != "https://cdn.example.com/u" || cfg.ImagesPerMinute != 5 {
		t.Fatalf("config = %+v", cfg)
	}

	t.Setenv("RATE_LIMIT_IMAGES_PER_MINUTE", "0")
	if _, err := Load(); err == nil {
		t.Fatal("zero uploads per minute must be rejected")
	}
}
