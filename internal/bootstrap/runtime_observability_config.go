package bootstrap

import (
	"os"
	"strconv"
	"strings"
	"time"

	"DND-AI-BOT/internal/agent/runtime"
)

const defaultRuntimeModelCallLogThreshold = time.Second

type RuntimeObservabilityConfig struct {
	ModelCallLogConfig runtime.RuntimeModelCallLogConfig
}

func LoadRuntimeObservabilityConfigFromEnv() RuntimeObservabilityConfig {
	return RuntimeObservabilityConfig{
		ModelCallLogConfig: runtime.RuntimeModelCallLogConfig{
			Mode:      parseRuntimeModelCallLogMode(os.Getenv("RUNTIME_MODEL_CALL_LOG_MODE")),
			Threshold: parseRuntimeModelCallLogThreshold(os.Getenv("RUNTIME_MODEL_CALL_LOG_THRESHOLD_MS")),
		},
	}
}

func parseRuntimeModelCallLogMode(value string) runtime.RuntimeModelCallLogMode {
	switch runtime.RuntimeModelCallLogMode(strings.TrimSpace(value)) {
	case runtime.RuntimeModelCallLogOff:
		return runtime.RuntimeModelCallLogOff
	case runtime.RuntimeModelCallLogAll:
		return runtime.RuntimeModelCallLogAll
	case runtime.RuntimeModelCallLogSlow, "":
		return runtime.RuntimeModelCallLogSlow
	default:
		return runtime.RuntimeModelCallLogSlow
	}
}

func parseRuntimeModelCallLogThreshold(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultRuntimeModelCallLogThreshold
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return defaultRuntimeModelCallLogThreshold
	}
	return time.Duration(parsed) * time.Millisecond
}
