package app

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"DND-AI-BOT/internal/observability"
)

func TestRecordAgentPhaseAndPromptSegmentMetrics(t *testing.T) {
	t.Parallel()

	metrics := observability.NewMetrics(prometheus.NewRegistry())
	recordAgentPhase(metrics, "preloaded_context_build", "success", time.Now())
	recordAgentPromptSegmentChars(metrics, "final_system_prompt", "青稞")

	recorder := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(recorder, httptest.NewRequest("GET", "/metrics", nil))
	body := recorder.Body.String()
	for _, expected := range []string{
		"agent_phase_duration_seconds",
		`phase="preloaded_context_build"`,
		"agent_prompt_segment_chars",
		`segment="final_system_prompt"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in metrics output, got %s", expected, body)
		}
	}
}
