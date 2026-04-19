package observability

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMetricsHandlerExposesRegisteredMetrics(t *testing.T) {
	metrics := NewMetrics(prometheus.NewRegistry())
	metrics.AgentRunsTotal.WithLabelValues("success").Inc()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/metrics", nil)
	metrics.Handler().ServeHTTP(recorder, request)

	body := recorder.Body.String()
	if !strings.Contains(body, "agent_runs_total") {
		t.Fatalf("expected agent_runs_total in metrics output, got %s", body)
	}
}
