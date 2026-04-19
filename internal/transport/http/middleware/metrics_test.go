package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"DND-AI-BOT/internal/observability"
)

func TestMetricsMiddlewareRecordsHTTPRequest(t *testing.T) {
	metrics := observability.NewMetrics(prometheus.NewRegistry())
	handler := NewMetricsMiddleware(metrics)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/sessions/session-1/messages", nil))

	recorder := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := recorder.Body.String()
	if !strings.Contains(body, `http_requests_total{method="POST",route="/sessions/{session_id}/messages",status="202"} 1`) {
		t.Fatalf("expected HTTP request metric, got %s", body)
	}
}

func TestRoutePatternNormalizesSessionIDs(t *testing.T) {
	if got := routePattern("/sessions/session-123/game-state"); got != "/sessions/{session_id}/game-state" {
		t.Fatalf("expected normalized session route, got %q", got)
	}
}
