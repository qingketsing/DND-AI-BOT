package observability

import "github.com/prometheus/client_golang/prometheus"

// NewNoopMetrics 创建独立 registry 的指标集合，适合测试或未显式注入时使用。
func NewNoopMetrics() *Metrics {
	return NewMetrics(prometheus.NewRegistry())
}
