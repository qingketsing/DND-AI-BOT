package client

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"DND-AI-BOT/internal/agent/runtime"
	"DND-AI-BOT/internal/observability"
)

type observedModelAdapter struct {
	next    ModelAdapter
	config  Config
	metrics *observability.Metrics
}

// NewObservedModelAdapter 用指标包装模型适配器。
func NewObservedModelAdapter(next ModelAdapter, config Config, metrics *observability.Metrics) ModelAdapter {
	if next == nil || metrics == nil {
		return next
	}
	return &observedModelAdapter{
		next:    next,
		config:  config,
		metrics: metrics,
	}
}

func (a *observedModelAdapter) Run(ctx context.Context, input runtime.ModelInput) (runtime.ModelOutput, error) {
	startedAt := time.Now()
	output, err := a.next.Run(ctx, input)
	if err != nil {
		a.record("error", startedAt)
		a.metrics.ModelErrorsTotal.With(prometheus.Labels{
			"provider": string(a.config.Provider),
			"model":    a.config.Model,
		}).Inc()
		return runtime.ModelOutput{}, err
	}
	a.record("success", startedAt)
	return output, nil
}

func (a *observedModelAdapter) record(status string, startedAt time.Time) {
	labels := prometheus.Labels{
		"provider": string(a.config.Provider),
		"model":    a.config.Model,
		"status":   status,
	}
	a.metrics.ModelRequestsTotal.With(labels).Inc()
	observability.ObserveDuration(a.metrics.ModelRequestDuration, labels, startedAt)
}
