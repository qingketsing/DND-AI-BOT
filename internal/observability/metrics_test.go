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
	metrics.RuntimeModelCallDuration.WithLabelValues("success", "final_reply").Observe(0.3)
	metrics.RuntimeToolStepDuration.WithLabelValues("search_rules", "success").Observe(0.4)
	metrics.RuntimeStepDuration.WithLabelValues("success", "tool_request").Observe(0.5)
	metrics.RuntimeModelCallsPerRun.WithLabelValues("success").Observe(2)
	metrics.RuntimeToolStepsPerRun.WithLabelValues("success").Observe(1)

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
		"runtime_model_call_duration_seconds",
		"runtime_tool_step_duration_seconds",
		"runtime_step_duration_seconds",
		"runtime_model_calls_per_run",
		"runtime_tool_steps_per_run",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in metrics output, got %s", expected, body)
		}
	}
}
