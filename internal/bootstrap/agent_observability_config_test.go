package bootstrap

import (
	"testing"
	"time"
)

func TestLoadAgentObservabilityConfigDefaultsToSlowTenSeconds(t *testing.T) {
	t.Setenv("AGENT_LATENCY_BREAKDOWN_LOG_MODE", "")
	t.Setenv("AGENT_LATENCY_BREAKDOWN_THRESHOLD_MS", "")

	config := LoadAgentObservabilityConfigFromEnv()

	if config.LatencyBreakdownLogConfig.Mode != AgentLatencyBreakdownLogSlow {
		t.Fatalf("expected slow mode, got %q", config.LatencyBreakdownLogConfig.Mode)
	}
	if config.LatencyBreakdownLogConfig.Threshold != 10*time.Second {
		t.Fatalf("expected 10s threshold, got %s", config.LatencyBreakdownLogConfig.Threshold)
	}
}

func TestLoadAgentObservabilityConfigParsesAllModeAndThreshold(t *testing.T) {
	t.Setenv("AGENT_LATENCY_BREAKDOWN_LOG_MODE", "all")
	t.Setenv("AGENT_LATENCY_BREAKDOWN_THRESHOLD_MS", "250")

	config := LoadAgentObservabilityConfigFromEnv()

	if config.LatencyBreakdownLogConfig.Mode != AgentLatencyBreakdownLogAll {
		t.Fatalf("expected all mode, got %q", config.LatencyBreakdownLogConfig.Mode)
	}
	if config.LatencyBreakdownLogConfig.Threshold != 250*time.Millisecond {
		t.Fatalf("expected 250ms threshold, got %s", config.LatencyBreakdownLogConfig.Threshold)
	}
}

func TestLoadAgentObservabilityConfigFallsBackOnInvalidValues(t *testing.T) {
	t.Setenv("AGENT_LATENCY_BREAKDOWN_LOG_MODE", "verbose")
	t.Setenv("AGENT_LATENCY_BREAKDOWN_THRESHOLD_MS", "-1")

	config := LoadAgentObservabilityConfigFromEnv()

	if config.LatencyBreakdownLogConfig.Mode != AgentLatencyBreakdownLogSlow {
		t.Fatalf("expected slow mode fallback, got %q", config.LatencyBreakdownLogConfig.Mode)
	}
	if config.LatencyBreakdownLogConfig.Threshold != 10*time.Second {
		t.Fatalf("expected 10s threshold fallback, got %s", config.LatencyBreakdownLogConfig.Threshold)
	}
}
