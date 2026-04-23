package bootstrap

import (
	"net/http"
	"testing"
)

func TestLoadSecurityConfigFromEnvParsesProductionSecurityOptions(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com, https://admin.example.com")
	t.Setenv("HTTP_MAX_BODY_BYTES", "2048")
	t.Setenv("AUTH_COOKIE_SECURE", "true")
	t.Setenv("AUTH_COOKIE_DOMAIN", "example.com")
	t.Setenv("AUTH_COOKIE_SAMESITE", "none")
	t.Setenv("METRICS_ENABLED", "true")
	t.Setenv("METRICS_ALLOWED_CIDRS", "127.0.0.1/32,10.0.0.0/8")
	t.Setenv("METRICS_BEARER_TOKEN", "metrics-token")
	t.Setenv("TRUSTED_PROXIES", "127.0.0.1/32,172.16.0.0/12")

	config := LoadSecurityConfigFromEnv()

	if len(config.AllowedOrigins) != 2 || config.AllowedOrigins[0] != "https://app.example.com" || config.AllowedOrigins[1] != "https://admin.example.com" {
		t.Fatalf("expected allowed origins to be parsed, got %+v", config.AllowedOrigins)
	}
	if config.MaxBodyBytes != 2048 {
		t.Fatalf("expected max body bytes 2048, got %d", config.MaxBodyBytes)
	}
	if !config.Cookie.Secure {
		t.Fatal("expected secure cookie")
	}
	if config.Cookie.Domain != "example.com" {
		t.Fatalf("expected cookie domain example.com, got %q", config.Cookie.Domain)
	}
	if config.Cookie.SameSite != http.SameSiteNoneMode {
		t.Fatalf("expected SameSite=None, got %v", config.Cookie.SameSite)
	}
	if !config.Metrics.Enabled {
		t.Fatal("expected metrics access to be enabled")
	}
	if len(config.Metrics.AllowedCIDRs) != 2 {
		t.Fatalf("expected two metrics CIDRs, got %+v", config.Metrics.AllowedCIDRs)
	}
	if config.Metrics.BearerToken != "metrics-token" {
		t.Fatalf("expected metrics bearer token, got %q", config.Metrics.BearerToken)
	}
	if len(config.TrustedProxies) != 2 {
		t.Fatalf("expected trusted proxies to be parsed, got %+v", config.TrustedProxies)
	}
}

func TestLoadSecurityConfigFromEnvUsesSafeDefaults(t *testing.T) {
	config := LoadSecurityConfigFromEnv()

	if config.MaxBodyBytes != 1<<20 {
		t.Fatalf("expected default max body bytes 1MiB, got %d", config.MaxBodyBytes)
	}
	if !config.AllowCredentials {
		t.Fatal("expected credentials to be allowed for cookie auth")
	}
	if config.Cookie.Secure {
		t.Fatal("expected secure cookie to be disabled by default for local development")
	}
	if config.Cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite=Lax by default, got %v", config.Cookie.SameSite)
	}
	if !config.Metrics.Enabled {
		t.Fatal("expected metrics endpoint enabled by default")
	}
}
