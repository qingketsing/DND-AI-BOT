package app

import (
	"log/slog"
	"time"
	"unicode/utf8"

	"github.com/prometheus/client_golang/prometheus"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/observability"
)

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

func DefaultAgentLatencyBreakdownLogConfig() AgentLatencyBreakdownLogConfig {
	return AgentLatencyBreakdownLogConfig{
		Mode:      AgentLatencyBreakdownLogSlow,
		Threshold: 10 * time.Second,
	}
}

func NormalizeAgentLatencyBreakdownLogConfig(config AgentLatencyBreakdownLogConfig) AgentLatencyBreakdownLogConfig {
	if config.Mode != AgentLatencyBreakdownLogOff && config.Mode != AgentLatencyBreakdownLogSlow && config.Mode != AgentLatencyBreakdownLogAll {
		config.Mode = AgentLatencyBreakdownLogSlow
	}
	if config.Threshold < 0 {
		config.Threshold = 10 * time.Second
	}
	return config
}

type agentLatencyBreakdown struct {
	WarmupBuild            time.Duration
	SystemPromptCompose    time.Duration
	PreloadedContextBuild  time.Duration
	RuntimeTotal           time.Duration
	Total                  time.Duration
	BaseSystemPromptChars  int
	WarmupBundleChars      int
	PreloadedContextChars  int
	FinalSystemPromptChars int
	UserMessageChars       int
}

func recordAgentPhase(metrics *observability.Metrics, phase string, status string, startedAt time.Time) {
	if metrics == nil {
		return
	}
	observability.ObserveDuration(metrics.AgentPhaseDuration, prometheus.Labels{
		"phase":  phase,
		"status": status,
	}, startedAt)
}

func recordAgentPromptSegmentChars(metrics *observability.Metrics, segment string, text string) {
	if metrics == nil {
		return
	}
	observability.ObserveHistogram(metrics.AgentPromptSegmentChars, prometheus.Labels{
		"segment": segment,
	}, float64(countChars(text)))
}

func warmupBundleText(bundle model.WarmupBundle) string {
	return bundle.BaseRulesSummary + "\n" + bundle.BaseLoreSummary + "\n" + bundle.CharacterRulesSummary
}

func countChars(text string) int {
	return utf8.RuneCountInString(text)
}

func logAgentLatencyBreakdown(logger *slog.Logger, sessionID string, breakdown agentLatencyBreakdown, config AgentLatencyBreakdownLogConfig) {
	if logger == nil || !shouldLogAgentLatencyBreakdown(breakdown.Total, config) {
		return
	}
	logger.Warn(
		"agent latency breakdown",
		"session_id", sessionID,
		"total_ms", breakdown.Total.Milliseconds(),
		"warmup_build_ms", breakdown.WarmupBuild.Milliseconds(),
		"system_prompt_compose_ms", breakdown.SystemPromptCompose.Milliseconds(),
		"preloaded_context_build_ms", breakdown.PreloadedContextBuild.Milliseconds(),
		"runtime_total_ms", breakdown.RuntimeTotal.Milliseconds(),
		"base_system_prompt_chars", breakdown.BaseSystemPromptChars,
		"warmup_bundle_chars", breakdown.WarmupBundleChars,
		"preloaded_context_chars", breakdown.PreloadedContextChars,
		"final_system_prompt_chars", breakdown.FinalSystemPromptChars,
		"user_message_chars", breakdown.UserMessageChars,
	)
}

func shouldLogAgentLatencyBreakdown(total time.Duration, config AgentLatencyBreakdownLogConfig) bool {
	config = NormalizeAgentLatencyBreakdownLogConfig(config)
	switch config.Mode {
	case AgentLatencyBreakdownLogOff:
		return false
	case AgentLatencyBreakdownLogAll:
		return true
	case AgentLatencyBreakdownLogSlow:
		return total >= config.Threshold
	default:
		return total >= config.Threshold
	}
}
