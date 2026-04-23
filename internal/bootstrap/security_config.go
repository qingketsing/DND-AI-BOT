package bootstrap

import (
	"net/http"
	"os"
	"strconv"
	"strings"
)

type SecurityConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxBodyBytes     int64
	TrustedProxies   []string
	Cookie           CookieConfig
	Metrics          MetricsAccessConfig
}

type CookieConfig struct {
	Secure   bool
	SameSite http.SameSite
	Domain   string
}

type MetricsAccessConfig struct {
	Enabled      bool
	AllowedCIDRs []string
	BearerToken  string
}

func LoadSecurityConfigFromEnv() SecurityConfig {
	return SecurityConfig{
		AllowedOrigins: splitCSV(os.Getenv("CORS_ALLOWED_ORIGINS")),
		AllowedMethods: envCSV("CORS_ALLOWED_METHODS", []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodDelete,
			http.MethodOptions,
		}),
		AllowedHeaders: envCSV("CORS_ALLOWED_HEADERS", []string{
			"Content-Type",
			"X-Request-ID",
		}),
		AllowCredentials: envBool("CORS_ALLOW_CREDENTIALS", true),
		MaxBodyBytes:     envInt64("HTTP_MAX_BODY_BYTES", 1<<20),
		TrustedProxies: envCSV("TRUSTED_PROXIES", []string{
			"127.0.0.1/32",
			"::1/128",
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
		}),
		Cookie: CookieConfig{
			Secure:   envBool("AUTH_COOKIE_SECURE", false),
			Domain:   strings.TrimSpace(os.Getenv("AUTH_COOKIE_DOMAIN")),
			SameSite: parseSameSite(os.Getenv("AUTH_COOKIE_SAMESITE"), http.SameSiteLaxMode),
		},
		Metrics: MetricsAccessConfig{
			Enabled: envBool("METRICS_ENABLED", true),
			AllowedCIDRs: envCSV("METRICS_ALLOWED_CIDRS", []string{
				"127.0.0.1/32",
				"::1/128",
				"10.0.0.0/8",
				"172.16.0.0/12",
				"192.168.0.0/16",
			}),
			BearerToken: strings.TrimSpace(os.Getenv("METRICS_BEARER_TOKEN")),
		},
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func envCSV(name string, fallback []string) []string {
	values := splitCSV(os.Getenv(name))
	if len(values) == 0 {
		return append([]string(nil), fallback...)
	}
	return values
}

func envBool(name string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt64(name string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseSameSite(value string, fallback http.SameSite) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "default":
		return http.SameSiteDefaultMode
	case "lax", "":
		if strings.TrimSpace(value) == "" {
			return fallback
		}
		return http.SameSiteLaxMode
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return fallback
	}
}
