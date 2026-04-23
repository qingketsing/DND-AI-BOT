package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityMiddlewareAllowsConfiguredCORSOrigin(t *testing.T) {
	handler := NewSecurityMiddleware(SecurityConfig{
		AllowedOrigins:   []string{"https://app.example.com"},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowedHeaders:   []string{"Content-Type", "X-Request-ID"},
		AllowCredentials: true,
		MaxBodyBytes:     1024,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodOptions, "/sessions", nil)
	request.Header.Set("Origin", "https://app.example.com")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected preflight status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Fatalf("expected allowed origin header, got %q", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
	if recorder.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("expected credentials header, got %q", recorder.Header().Get("Access-Control-Allow-Credentials"))
	}
	if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("expected nosniff security header, got %q", recorder.Header().Get("X-Content-Type-Options"))
	}
}

func TestSecurityMiddlewareRejectsOversizedRequestBody(t *testing.T) {
	handler := NewSecurityMiddleware(SecurityConfig{MaxBodyBytes: 4})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader("12345"))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, recorder.Code)
	}
}

func TestSecurityMiddlewareDoesNotAllowUnknownOrigin(t *testing.T) {
	handler := NewSecurityMiddleware(SecurityConfig{
		AllowedOrigins: []string{"https://app.example.com"},
		MaxBodyBytes:   1024,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/sessions", nil)
	request.Header.Set("Origin", "https://evil.example.com")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected unknown origin not to be allowed, got %q", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestSecurityMiddlewareStoresForwardedClientIPWhenRemoteIsTrustedProxy(t *testing.T) {
	handler := NewSecurityMiddleware(SecurityConfig{
		MaxBodyBytes:   1024,
		TrustedProxies: []string{"127.0.0.1/32"},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := ClientIPFromRequest(r); got != "203.0.113.10" {
			t.Fatalf("expected forwarded client ip, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/sessions", nil)
	request.RemoteAddr = "127.0.0.1:12345"
	request.Header.Set("X-Forwarded-For", "203.0.113.10, 127.0.0.1")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestMetricsAccessMiddlewareRequiresAllowedCIDROrBearerToken(t *testing.T) {
	handler := NewMetricsAccessMiddleware(MetricsAccessConfig{
		Enabled:      true,
		AllowedCIDRs: []string{"127.0.0.1/32"},
		BearerToken:  "secret-token",
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	deniedRequest := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	deniedRequest.RemoteAddr = "203.0.113.10:12345"
	deniedRecorder := httptest.NewRecorder()
	handler.ServeHTTP(deniedRecorder, deniedRequest)
	if deniedRecorder.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden without allowed CIDR or token, got %d", deniedRecorder.Code)
	}

	tokenRequest := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	tokenRequest.RemoteAddr = "203.0.113.10:12345"
	tokenRequest.Header.Set("Authorization", "Bearer secret-token")
	tokenRecorder := httptest.NewRecorder()
	handler.ServeHTTP(tokenRecorder, tokenRequest)
	if tokenRecorder.Code != http.StatusOK {
		t.Fatalf("expected bearer token to allow metrics, got %d", tokenRecorder.Code)
	}

	localRequest := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	localRequest.RemoteAddr = "127.0.0.1:12345"
	localRecorder := httptest.NewRecorder()
	handler.ServeHTTP(localRecorder, localRequest)
	if localRecorder.Code != http.StatusOK {
		t.Fatalf("expected allowed CIDR to allow metrics, got %d", localRecorder.Code)
	}
}
