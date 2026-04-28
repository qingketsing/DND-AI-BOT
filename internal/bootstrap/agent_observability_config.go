package bootstrap

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultAgentLatencyBreakdownThreshold = 10 * time.Second

type AgentLatencyBreakdownLogMode string

const (
	AgentLatencyBreakdownLogOff  AgentLatencyBreakdownLogMode = "off"
	AgentLatencyBreakdownLogSlow AgentLatencyBreakdownLogMode = "slow"
	AgentLatencyBreakdownLogAll  AgentLatencyBreakdownLogMode = "all"
)

type AgentLatencyBreakdownLogConfig struct {
	Mode      AgentLatencyBreakdownLogMode
	Threshold time.Duration
}

type AgentObservabilityConfig struct {
	LatencyBreakdownLogConfig AgentLatencyBreakdownLogConfig
}

func LoadAgentObservabilityConfigFromEnv() AgentObservabilityConfig {
	return AgentObservabilityConfig{
		LatencyBreakdownLogConfig: AgentLatencyBreakdownLogConfig{
			Mode:      parseAgentLatencyBreakdownLogMode(os.Getenv("AGENT_LATENCY_BREAKDOWN_LOG_MODE")),
			Threshold: parseAgentLatencyBreakdownThreshold(os.Getenv("AGENT_LATENCY_BREAKDOWN_THRESHOLD_MS")),
		},
	}
}

func parseAgentLatencyBreakdownLogMode(value string) AgentLatencyBreakdownLogMode {
	switch AgentLatencyBreakdownLogMode(strings.TrimSpace(value)) {
	case AgentLatencyBreakdownLogOff:
		return AgentLatencyBreakdownLogOff
	case AgentLatencyBreakdownLogAll:
		return AgentLatencyBreakdownLogAll
	case AgentLatencyBreakdownLogSlow, "":
		return AgentLatencyBreakdownLogSlow
	default:
		return AgentLatencyBreakdownLogSlow
	}
}

func parseAgentLatencyBreakdownThreshold(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultAgentLatencyBreakdownThreshold
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return defaultAgentLatencyBreakdownThreshold
	}
	return time.Duration(parsed) * time.Millisecond
}
