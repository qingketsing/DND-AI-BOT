package observability

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics 汇总应用核心链路的 Prometheus 指标。
type Metrics struct {
	registry *prometheus.Registry

	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	AgentRunsTotal          *prometheus.CounterVec
	AgentRunDuration        *prometheus.HistogramVec
	AgentFallbackTotal      *prometheus.CounterVec
	AgentPhaseDuration      *prometheus.HistogramVec
	AgentPromptSegmentChars *prometheus.HistogramVec

	ModelRequestsTotal   *prometheus.CounterVec
	ModelRequestDuration *prometheus.HistogramVec
	ModelErrorsTotal     *prometheus.CounterVec

	RAGSearchTotal    *prometheus.CounterVec
	RAGSearchDuration *prometheus.HistogramVec
	RAGDegradedTotal  *prometheus.CounterVec
	RAGPhaseDuration  *prometheus.HistogramVec

	ToolCallsTotal    *prometheus.CounterVec
	ToolCallDuration  *prometheus.HistogramVec
	ToolErrorsTotal   *prometheus.CounterVec
	SessionMemoryRuns *prometheus.CounterVec

	RuntimeModelCallDuration *prometheus.HistogramVec
	RuntimeToolStepDuration  *prometheus.HistogramVec
	RuntimeStepDuration      *prometheus.HistogramVec
	RuntimeModelCallsPerRun  *prometheus.HistogramVec
	RuntimeToolStepsPerRun   *prometheus.HistogramVec

	RateLimitChecksTotal  *prometheus.CounterVec
	RateLimitBlockedTotal *prometheus.CounterVec
	RateLimitErrorsTotal  *prometheus.CounterVec
}

// NewMetrics 创建并注册应用指标。
func NewMetrics(registry *prometheus.Registry) *Metrics {
	if registry == nil {
		registry = prometheus.NewRegistry()
	}
	metrics := &Metrics{
		registry: registry,
		HTTPRequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests.",
		}, []string{"method", "route", "status"}),
		HTTPRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route", "status"}),
		AgentRunsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "agent_runs_total",
			Help: "Total agent runs.",
		}, []string{"status"}),
		AgentRunDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "agent_run_duration_seconds",
			Help:    "Agent run duration.",
			Buckets: prometheus.DefBuckets,
		}, []string{"status"}),
		AgentFallbackTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "agent_fallback_total",
			Help: "Total agent fallback replies.",
		}, []string{"fallback_kind"}),
		AgentPhaseDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "agent_phase_duration_seconds",
			Help:    "Agent internal phase duration.",
			Buckets: prometheus.DefBuckets,
		}, []string{"phase", "status"}),
		AgentPromptSegmentChars: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "agent_prompt_segment_chars",
			Help:    "Agent prompt segment size in characters.",
			Buckets: []float64{128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536},
		}, []string{"segment"}),
		ModelRequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "model_requests_total",
			Help: "Total model requests.",
		}, []string{"provider", "model", "status"}),
		ModelRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "model_request_duration_seconds",
			Help:    "Model request duration.",
			Buckets: prometheus.DefBuckets,
		}, []string{"provider", "model", "status"}),
		ModelErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "model_errors_total",
			Help: "Total model request errors.",
		}, []string{"provider", "model"}),
		RAGSearchTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rag_search_total",
			Help: "Total RAG searches.",
		}, []string{"knowledge_base", "status"}),
		RAGSearchDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "rag_search_duration_seconds",
			Help:    "RAG search duration.",
			Buckets: prometheus.DefBuckets,
		}, []string{"knowledge_base", "status"}),
		RAGDegradedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rag_degraded_total",
			Help: "Total degraded RAG searches.",
		}, []string{"knowledge_base"}),
		RAGPhaseDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "rag_phase_duration_seconds",
			Help:    "RAG internal phase duration.",
			Buckets: prometheus.DefBuckets,
		}, []string{"knowledge_base", "phase", "status"}),
		ToolCallsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tool_calls_total",
			Help: "Total tool calls.",
		}, []string{"tool", "status"}),
		ToolCallDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "tool_call_duration_seconds",
			Help:    "Tool call duration.",
			Buckets: prometheus.DefBuckets,
		}, []string{"tool", "status"}),
		ToolErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tool_errors_total",
			Help: "Total tool call errors.",
		}, []string{"tool"}),
		SessionMemoryRuns: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "session_memory_refresh_total",
			Help: "Total session memory refresh attempts.",
		}, []string{"status"}),
		RuntimeModelCallDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "runtime_model_call_duration_seconds",
			Help:    "Runtime model call duration by output type.",
			Buckets: prometheus.DefBuckets,
		}, []string{"status", "output_type"}),
		RuntimeToolStepDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "runtime_tool_step_duration_seconds",
			Help:    "Runtime tool step duration.",
			Buckets: prometheus.DefBuckets,
		}, []string{"tool", "status"}),
		RuntimeStepDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "runtime_step_duration_seconds",
			Help:    "Runtime ReAct step duration.",
			Buckets: prometheus.DefBuckets,
		}, []string{"status", "output_type"}),
		RuntimeModelCallsPerRun: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "runtime_model_calls_per_run",
			Help:    "Number of model calls per runtime run.",
			Buckets: []float64{1, 2, 3, 4, 5, 8, 13, 21},
		}, []string{"status"}),
		RuntimeToolStepsPerRun: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "runtime_tool_steps_per_run",
			Help:    "Number of tool steps per runtime run.",
			Buckets: []float64{0, 1, 2, 3, 4, 5, 8, 13, 21},
		}, []string{"status"}),
		RateLimitChecksTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rate_limit_checks_total",
			Help: "Total rate limit checks.",
		}, []string{"endpoint", "scope", "status"}),
		RateLimitBlockedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rate_limit_blocked_total",
			Help: "Total blocked rate limit checks.",
		}, []string{"endpoint", "scope"}),
		RateLimitErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rate_limit_errors_total",
			Help: "Total rate limit backend errors.",
		}, []string{"endpoint", "scope"}),
	}
	registry.MustRegister(
		metrics.HTTPRequestsTotal,
		metrics.HTTPRequestDuration,
		metrics.AgentRunsTotal,
		metrics.AgentRunDuration,
		metrics.AgentFallbackTotal,
		metrics.AgentPhaseDuration,
		metrics.AgentPromptSegmentChars,
		metrics.ModelRequestsTotal,
		metrics.ModelRequestDuration,
		metrics.ModelErrorsTotal,
		metrics.RAGSearchTotal,
		metrics.RAGSearchDuration,
		metrics.RAGDegradedTotal,
		metrics.RAGPhaseDuration,
		metrics.ToolCallsTotal,
		metrics.ToolCallDuration,
		metrics.ToolErrorsTotal,
		metrics.SessionMemoryRuns,
		metrics.RuntimeModelCallDuration,
		metrics.RuntimeToolStepDuration,
		metrics.RuntimeStepDuration,
		metrics.RuntimeModelCallsPerRun,
		metrics.RuntimeToolStepsPerRun,
		metrics.RateLimitChecksTotal,
		metrics.RateLimitBlockedTotal,
		metrics.RateLimitErrorsTotal,
	)
	return metrics
}

// Registry 返回 Prometheus registry。
func (m *Metrics) Registry() *prometheus.Registry {
	if m == nil {
		return prometheus.NewRegistry()
	}
	return m.registry
}

// Handler 返回 /metrics HTTP handler。
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry(), promhttp.HandlerOpts{})
}

// ObserveDuration 记录耗时。
func ObserveDuration(histogram *prometheus.HistogramVec, labels prometheus.Labels, start time.Time) {
	if histogram == nil {
		return
	}
	histogram.With(labels).Observe(time.Since(start).Seconds())
}

// ObserveHistogram 记录任意数值型直方图。
func ObserveHistogram(histogram *prometheus.HistogramVec, labels prometheus.Labels, value float64) {
	if histogram == nil {
		return
	}
	histogram.With(labels).Observe(value)
}
