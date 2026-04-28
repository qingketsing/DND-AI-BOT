package bootstrap

import (
	"testing"
	"time"

	"DND-AI-BOT/internal/agent/runtime"
)

func TestLoadRuntimeObservabilityConfigDefaultsToSlowOneSecond(t *testing.T) {
	t.Setenv("RUNTIME_MODEL_CALL_LOG_MODE", "")
	t.Setenv("RUNTIME_MODEL_CALL_LOG_THRESHOLD_MS", "")

	config := LoadRuntimeObservabilityConfigFromEnv()

	if config.ModelCallLogConfig.Mode != runtime.RuntimeModelCallLogSlow {
		t.Fatalf("expected slow mode, got %q", config.ModelCallLogConfig.Mode)
	}
	if config.ModelCallLogConfig.Threshold != time.Second {
		t.Fatalf("expected one second threshold, got %s", config.ModelCallLogConfig.Threshold)
	}
}

func TestLoadRuntimeObservabilityConfigParsesAllModeAndThreshold(t *testing.T) {
	t.Setenv("RUNTIME_MODEL_CALL_LOG_MODE", "all")
	t.Setenv("RUNTIME_MODEL_CALL_LOG_THRESHOLD_MS", "250")

	config := LoadRuntimeObservabilityConfigFromEnv()

	if config.ModelCallLogConfig.Mode != runtime.RuntimeModelCallLogAll {
		t.Fatalf("expected all mode, got %q", config.ModelCallLogConfig.Mode)
	}
	if config.ModelCallLogConfig.Threshold != 250*time.Millisecond {
		t.Fatalf("expected 250ms threshold, got %s", config.ModelCallLogConfig.Threshold)
	}
}

func TestLoadRuntimeObservabilityConfigFallsBackOnInvalidValues(t *testing.T) {
	t.Setenv("RUNTIME_MODEL_CALL_LOG_MODE", "verbose")
	t.Setenv("RUNTIME_MODEL_CALL_LOG_THRESHOLD_MS", "-1")

	config := LoadRuntimeObservabilityConfigFromEnv()

	if config.ModelCallLogConfig.Mode != runtime.RuntimeModelCallLogSlow {
		t.Fatalf("expected slow mode fallback, got %q", config.ModelCallLogConfig.Mode)
	}
	if config.ModelCallLogConfig.Threshold != time.Second {
		t.Fatalf("expected one second threshold fallback, got %s", config.ModelCallLogConfig.Threshold)
	}
}
