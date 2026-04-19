package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"DND-AI-BOT/internal/observability"
)

func TestRequestIDMiddlewareUsesIncomingHeader(t *testing.T) {
	middleware := NewRequestIDMiddleware()
	var captured string
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = observability.RequestIDFromContext(r.Context())
	}))

	request := httptest.NewRequest(http.MethodGet, "/ping", nil)
	request.Header.Set("X-Request-ID", "req-incoming")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if captured != "req-incoming" {
		t.Fatalf("expected request id from header, got %q", captured)
	}
	if recorder.Header().Get("X-Request-ID") != "req-incoming" {
		t.Fatalf("expected response request id header, got %q", recorder.Header().Get("X-Request-ID"))
	}
}

func TestRequestIDMiddlewareGeneratesMissingID(t *testing.T) {
	middleware := NewRequestIDMiddleware()
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if observability.RequestIDFromContext(r.Context()) == "" {
			t.Fatal("expected generated request id in context")
		}
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ping", nil))

	if recorder.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected generated request id response header")
	}
}
