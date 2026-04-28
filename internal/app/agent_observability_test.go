package app

import (
	"bytes"
	"log/slog"
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

func TestLogAgentLatencyBreakdownLogsWhenModeAll(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buffer, nil))

	logAgentLatencyBreakdown(logger, "session-1", agentLatencyBreakdown{
		Total:        time.Millisecond,
		RuntimeTotal: time.Millisecond,
	}, AgentLatencyBreakdownLogConfig{
		Mode:      AgentLatencyBreakdownLogAll,
		Threshold: time.Hour,
	})

	logOutput := buffer.String()
	for _, expected := range []string{"agent latency breakdown", "session_id=session-1", "total_ms=1", "runtime_total_ms=1"} {
		if !strings.Contains(logOutput, expected) {
			t.Fatalf("expected log output to contain %q, got %s", expected, logOutput)
		}
	}
}

func TestLogAgentLatencyBreakdownSkipsFastRunWhenModeSlow(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buffer, nil))

	logAgentLatencyBreakdown(logger, "session-1", agentLatencyBreakdown{
		Total: time.Millisecond,
	}, AgentLatencyBreakdownLogConfig{
		Mode:      AgentLatencyBreakdownLogSlow,
		Threshold: time.Hour,
	})

	if logOutput := strings.TrimSpace(buffer.String()); logOutput != "" {
		t.Fatalf("expected no log output, got %s", logOutput)
	}
}
