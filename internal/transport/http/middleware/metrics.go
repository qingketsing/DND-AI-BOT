package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"DND-AI-BOT/internal/observability"
)

// NewMetricsMiddleware 记录 HTTP 请求计数和耗时。
func NewMetricsMiddleware(metrics *observability.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(recorder, r)
			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}
			if metrics != nil {
				labels := prometheus.Labels{
					"method": r.Method,
					"route":  routePattern(r.URL.Path),
					"status": strconv.Itoa(status),
				}
				metrics.HTTPRequestsTotal.With(labels).Inc()
				observability.ObserveDuration(metrics.HTTPRequestDuration, labels, start)
			}
		})
	}
}

func routePattern(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return "/"
	}
	if parts[0] == "sessions" && len(parts) >= 2 {
		parts[1] = "{session_id}"
	}
	return "/" + strings.Join(parts, "/")
}
