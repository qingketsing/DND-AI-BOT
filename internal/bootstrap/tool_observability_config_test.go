package bootstrap

import (
	"testing"
	"time"

	"DND-AI-BOT/internal/agent/tools"
)

func TestLoadToolObservabilityConfigDefaultsToSlowOneSecond(t *testing.T) {
	t.Setenv("TOOL_CALL_LOG_MODE", "")
	t.Setenv("TOOL_CALL_LOG_THRESHOLD_MS", "")

	config := LoadToolObservabilityConfigFromEnv()

	if config.CallLogConfig.Mode != tools.ToolCallLogSlow {
		t.Fatalf("expected slow mode, got %q", config.CallLogConfig.Mode)
	}
	if config.CallLogConfig.Threshold != time.Second {
		t.Fatalf("expected one second threshold, got %s", config.CallLogConfig.Threshold)
	}
}

func TestLoadToolObservabilityConfigParsesAllModeAndThreshold(t *testing.T) {
	t.Setenv("TOOL_CALL_LOG_MODE", "all")
	t.Setenv("TOOL_CALL_LOG_THRESHOLD_MS", "250")

	config := LoadToolObservabilityConfigFromEnv()

	if config.CallLogConfig.Mode != tools.ToolCallLogAll {
		t.Fatalf("expected all mode, got %q", config.CallLogConfig.Mode)
	}
	if config.CallLogConfig.Threshold != 250*time.Millisecond {
		t.Fatalf("expected 250ms threshold, got %s", config.CallLogConfig.Threshold)
	}
}

func TestLoadToolObservabilityConfigFallsBackOnInvalidValues(t *testing.T) {
	t.Setenv("TOOL_CALL_LOG_MODE", "verbose")
	t.Setenv("TOOL_CALL_LOG_THRESHOLD_MS", "-1")

	config := LoadToolObservabilityConfigFromEnv()

	if config.CallLogConfig.Mode != tools.ToolCallLogSlow {
		t.Fatalf("expected slow mode fallback, got %q", config.CallLogConfig.Mode)
	}
	if config.CallLogConfig.Threshold != time.Second {
		t.Fatalf("expected one second threshold fallback, got %s", config.CallLogConfig.Threshold)
	}
}
