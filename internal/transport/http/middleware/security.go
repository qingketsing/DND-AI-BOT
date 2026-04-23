package middleware

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"

	"DND-AI-BOT/internal/transport/http/dto"
)

type SecurityConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxBodyBytes     int64
	TrustedProxies   []string
}

type MetricsAccessConfig struct {
	Enabled      bool
	AllowedCIDRs []string
	BearerToken  string
}

type clientIPContextKey struct{}

func NewSecurityMiddleware(config SecurityConfig) func(http.Handler) http.Handler {
	allowedOrigins := makeSet(config.AllowedOrigins)
	allowedMethods := strings.Join(nonEmptyOrDefault(config.AllowedMethods, []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodOptions}), ", ")
	allowedHeaders := strings.Join(nonEmptyOrDefault(config.AllowedHeaders, []string{"Content-Type", requestIDHeader}), ", ")
	trustedProxyNets := parseCIDRs(config.TrustedProxies)
	maxBodyBytes := config.MaxBodyBytes
	if maxBodyBytes <= 0 {
		maxBodyBytes = 1 << 20
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			setSecurityHeaders(w)
			if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" && originAllowed(origin, allowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", appendVary(w.Header().Get("Vary"), "Origin"))
				w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
				w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
				if config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if r.ContentLength > maxBodyBytes {
				writeSecurityError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body is too large")
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
			r = r.WithContext(WithClientIP(r.Context(), resolveClientIP(r, trustedProxyNets)))
			next.ServeHTTP(w, r)
		})
	}
}

func WithClientIP(ctx context.Context, clientIP string) context.Context {
	return context.WithValue(ctx, clientIPContextKey{}, strings.TrimSpace(clientIP))
}

func ClientIPFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	if value, ok := r.Context().Value(clientIPContextKey{}).(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	ip := requestRemoteIP(r)
	if ip == nil {
		return ""
	}
	return ip.String()
}

func NewMetricsAccessMiddleware(config MetricsAccessConfig) func(http.Handler) http.Handler {
	allowedNets := parseCIDRs(config.AllowedCIDRs)
	bearerToken := strings.TrimSpace(config.BearerToken)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !config.Enabled {
				http.NotFound(w, r)
				return
			}
			if bearerToken != "" && r.Header.Get("Authorization") == "Bearer "+bearerToken {
				next.ServeHTTP(w, r)
				return
			}
			ip := requestRemoteIP(r)
			for _, network := range allowedNets {
				if network.Contains(ip) {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeSecurityError(w, http.StatusForbidden, "metrics_forbidden", "metrics access is forbidden")
		})
	}
}

func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
}

func makeSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result[value] = struct{}{}
		}
	}
	return result
}

func originAllowed(origin string, allowed map[string]struct{}) bool {
	if len(allowed) == 0 {
		return false
	}
	if _, ok := allowed[origin]; ok {
		return true
	}
	_, ok := allowed["*"]
	return ok
}

func appendVary(current string, value string) string {
	if current == "" {
		return value
	}
	for _, part := range strings.Split(current, ",") {
		if strings.EqualFold(strings.TrimSpace(part), value) {
			return current
		}
	}
	return current + ", " + value
}

func nonEmptyOrDefault(values []string, fallback []string) []string {
	if len(values) == 0 {
		return append([]string(nil), fallback...)
	}
	return values
}

func parseCIDRs(values []string) []*net.IPNet {
	networks := make([]*net.IPNet, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if ip := net.ParseIP(value); ip != nil {
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			_, network, err := net.ParseCIDR(ip.String() + "/" + strconv.Itoa(bits))
			if err == nil {
				networks = append(networks, network)
			}
			continue
		}
		_, network, err := net.ParseCIDR(value)
		if err == nil {
			networks = append(networks, network)
		}
	}
	return networks
}

func requestRemoteIP(r *http.Request) net.IP {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	return net.ParseIP(host)
}

func resolveClientIP(r *http.Request, trustedProxyNets []*net.IPNet) string {
	remoteIP := requestRemoteIP(r)
	if remoteIP == nil {
		return ""
	}
	if !ipInAnyNet(remoteIP, trustedProxyNets) {
		return remoteIP.String()
	}
	for _, candidate := range forwardedIPs(r.Header.Get("X-Forwarded-For")) {
		if ip := net.ParseIP(candidate); ip != nil {
			return ip.String()
		}
	}
	if ip := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); ip != nil {
		return ip.String()
	}
	return remoteIP.String()
}

func forwardedIPs(value string) []string {
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

func ipInAnyNet(ip net.IP, networks []*net.IPNet) bool {
	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func writeSecurityError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(dto.NewErrorResponse(code, message))
}
