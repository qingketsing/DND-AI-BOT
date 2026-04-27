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

func TestMetricsHandlerExposesLatencyBreakdownMetrics(t *testing.T) {
	metrics := NewMetrics(prometheus.NewRegistry())
	metrics.AgentPhaseDuration.WithLabelValues("warmup_build", "success").Observe(0.1)
	metrics.RAGPhaseDuration.WithLabelValues("rules", "embedding", "success").Observe(0.2)
	metrics.AgentPromptSegmentChars.WithLabelValues("final_system_prompt").Observe(1200)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/metrics", nil)
	metrics.Handler().ServeHTTP(recorder, request)

	body := recorder.Body.String()
	for _, expected := range []string{
		"agent_phase_duration_seconds",
		`phase="warmup_build"`,
		"rag_phase_duration_seconds",
		`phase="embedding"`,
		"agent_prompt_segment_chars",
		`segment="final_system_prompt"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in metrics output, got %s", expected, body)
		}
	}
}
