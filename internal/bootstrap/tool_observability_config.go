package bootstrap

import (
	"os"
	"strconv"
	"strings"
	"time"

	"DND-AI-BOT/internal/agent/tools"
)

const defaultToolCallLogThreshold = time.Second

type ToolObservabilityConfig struct {
	CallLogConfig tools.ToolCallLogConfig
}

func LoadToolObservabilityConfigFromEnv() ToolObservabilityConfig {
	return ToolObservabilityConfig{
		CallLogConfig: tools.ToolCallLogConfig{
			Mode:      parseToolCallLogMode(os.Getenv("TOOL_CALL_LOG_MODE")),
			Threshold: parseToolCallLogThreshold(os.Getenv("TOOL_CALL_LOG_THRESHOLD_MS")),
		},
	}
}

func parseToolCallLogMode(value string) tools.ToolCallLogMode {
	switch tools.ToolCallLogMode(strings.TrimSpace(value)) {
	case tools.ToolCallLogOff:
		return tools.ToolCallLogOff
	case tools.ToolCallLogAll:
		return tools.ToolCallLogAll
	case tools.ToolCallLogSlow, "":
		return tools.ToolCallLogSlow
	default:
		return tools.ToolCallLogSlow
	}
}

func parseToolCallLogThreshold(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultToolCallLogThreshold
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return defaultToolCallLogThreshold
	}
	return time.Duration(parsed) * time.Millisecond
}
