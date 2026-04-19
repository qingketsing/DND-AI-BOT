package client

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"DND-AI-BOT/internal/agent/runtime"
	"DND-AI-BOT/internal/observability"
)

func TestObservedModelAdapterRecordsSuccessMetrics(t *testing.T) {
	metrics := observability.NewMetrics(prometheus.NewRegistry())
	adapter := NewObservedModelAdapter(stubModelAdapter{
		output: runtime.ModelOutput{Reply: "ok"},
	}, Config{
		Provider: ProviderDeepSeek,
		Model:    "deepseek-chat",
	}, metrics)

	output, err := adapter.Run(context.Background(), runtime.ModelInput{SessionID: "session-1"})
	if err != nil {
		t.Fatalf("expected model run to succeed, got %v", err)
	}
	if output.Reply != "ok" {
		t.Fatalf("expected model reply, got %q", output.Reply)
	}

	body := scrapeClientMetrics(t, metrics)
	if !strings.Contains(body, `model_requests_total{model="deepseek-chat",provider="deepseek",status="success"} 1`) {
		t.Fatalf("expected successful model metric, got %s", body)
	}
}

func TestObservedModelAdapterRecordsErrorMetrics(t *testing.T) {
	metrics := observability.NewMetrics(prometheus.NewRegistry())
	adapter := NewObservedModelAdapter(stubModelAdapter{
		err: errors.New("model timeout"),
	}, Config{
		Provider: ProviderDeepSeek,
		Model:    "deepseek-chat",
	}, metrics)

	_, err := adapter.Run(context.Background(), runtime.ModelInput{SessionID: "session-1"})
	if err == nil {
		t.Fatal("expected model run to fail")
	}

	body := scrapeClientMetrics(t, metrics)
	if !strings.Contains(body, `model_requests_total{model="deepseek-chat",provider="deepseek",status="error"} 1`) ||
		!strings.Contains(body, `model_errors_total{model="deepseek-chat",provider="deepseek"} 1`) {
		t.Fatalf("expected model error metrics, got %s", body)
	}
}

func scrapeClientMetrics(t *testing.T, metrics *observability.Metrics) string {
	t.Helper()
	recorder := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(recorder, httptest.NewRequest("GET", "/metrics", nil))
	return recorder.Body.String()
}

type stubModelAdapter struct {
	output runtime.ModelOutput
	err    error
}

func (s stubModelAdapter) Run(ctx context.Context, input runtime.ModelInput) (runtime.ModelOutput, error) {
	_ = ctx
	_ = input
	if s.err != nil {
		return runtime.ModelOutput{}, s.err
	}
	return s.output, nil
}
