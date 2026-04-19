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

	AgentRunsTotal     *prometheus.CounterVec
	AgentRunDuration   *prometheus.HistogramVec
	AgentFallbackTotal *prometheus.CounterVec

	ModelRequestsTotal   *prometheus.CounterVec
	ModelRequestDuration *prometheus.HistogramVec
	ModelErrorsTotal     *prometheus.CounterVec

	RAGSearchTotal    *prometheus.CounterVec
	RAGSearchDuration *prometheus.HistogramVec
	RAGDegradedTotal  *prometheus.CounterVec

	ToolCallsTotal    *prometheus.CounterVec
	ToolCallDuration  *prometheus.HistogramVec
	ToolErrorsTotal   *prometheus.CounterVec
	SessionMemoryRuns *prometheus.CounterVec
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
	}
	registry.MustRegister(
		metrics.HTTPRequestsTotal,
		metrics.HTTPRequestDuration,
		metrics.AgentRunsTotal,
		metrics.AgentRunDuration,
		metrics.AgentFallbackTotal,
		metrics.ModelRequestsTotal,
		metrics.ModelRequestDuration,
		metrics.ModelErrorsTotal,
		metrics.RAGSearchTotal,
		metrics.RAGSearchDuration,
		metrics.RAGDegradedTotal,
		metrics.ToolCallsTotal,
		metrics.ToolCallDuration,
		metrics.ToolErrorsTotal,
		metrics.SessionMemoryRuns,
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
