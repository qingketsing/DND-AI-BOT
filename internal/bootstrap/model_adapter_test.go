package bootstrap

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"DND-AI-BOT/internal/agent/client"
)

func TestBuildModelAdapterBuildsMockAdapter(t *testing.T) {
	adapter, err := BuildModelAdapter(client.Config{
		Provider: client.ProviderMock,
	})
	if err != nil {
		t.Fatalf("expected mock adapter build to succeed, got %v", err)
	}
	if adapter == nil {
		t.Fatal("expected mock adapter to be created")
	}
}

func TestBuildModelAdapterBuildsDeepSeekAdapter(t *testing.T) {
	adapter, err := BuildModelAdapter(client.Config{
		Provider: client.ProviderDeepSeek,
		Model:    "deepseek-chat",
		APIKey:   "secret",
	})
	if err != nil {
		t.Fatalf("expected deepseek adapter build to succeed, got %v", err)
	}
	if adapter == nil {
		t.Fatal("expected deepseek adapter to be created")
	}
}

func TestBuildModelAdapterBuildsOpenAIAdapter(t *testing.T) {
	adapter, err := BuildModelAdapter(client.Config{
		Provider: client.ProviderOpenAI,
		Model:    "gpt-5.4-mini",
		APIKey:   "secret",
	})
	if err != nil {
		t.Fatalf("expected openai adapter build to succeed, got %v", err)
	}
	if adapter == nil {
		t.Fatal("expected openai adapter to be created")
	}
}

func TestBuildModelAdapterFromEnvBuildsMockAdapter(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "mock")
	t.Setenv("MODEL_NAME", "")
	t.Setenv("MODEL_API_KEY", "")
	t.Setenv("MODEL_BASE_URL", "")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "")

	adapter, config, err := BuildModelAdapterFromEnv()
	if err != nil {
		t.Fatalf("expected model adapter to build from env, got %v", err)
	}
	if adapter == nil {
		t.Fatal("expected model adapter to be created from env")
	}
	if config.Provider != client.ProviderMock {
		t.Fatalf("expected provider %q, got %q", client.ProviderMock, config.Provider)
	}
}

func TestBuildModelAdapterFromEnvForRoleBuildsSummarizerAdapter(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "mock")
	t.Setenv("MODEL_NAME", "")
	t.Setenv("MODEL_API_KEY", "")
	t.Setenv("MODEL_BASE_URL", "")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "")
	t.Setenv("SUMMARY_MODEL_PROVIDER", "mock")
	t.Setenv("SUMMARY_MODEL_NAME", "")
	t.Setenv("SUMMARY_MODEL_API_KEY", "")
	t.Setenv("SUMMARY_MODEL_BASE_URL", "")
	t.Setenv("SUMMARY_MODEL_TIMEOUT_SECONDS", "30")

	adapter, config, err := BuildModelAdapterFromEnvForRole(client.ModelRoleSummarizer)
	if err != nil {
		t.Fatalf("expected summarizer adapter to build from env, got %v", err)
	}
	if adapter == nil {
		t.Fatal("expected summarizer adapter to be created from env")
	}
	if config.Provider != client.ProviderMock {
		t.Fatalf("expected provider %q, got %q", client.ProviderMock, config.Provider)
	}
	if config.TimeoutSeconds != 30 {
		t.Fatalf("expected summarizer timeout 30, got %d", config.TimeoutSeconds)
	}
}

func TestBuildModelAdapterFromEnvRejectsInvalidProvider(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "unknown")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "60")

	_, _, err := BuildModelAdapterFromEnv()
	if err != ErrInvalidModelProvider {
		t.Fatalf("expected ErrInvalidModelProvider, got %v", err)
	}
}

func TestBuildModelAdapterFromEnvRejectsInvalidTimeout(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "mock")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "bad")

	_, _, err := BuildModelAdapterFromEnv()
	if err != ErrInvalidModelTimeout {
		t.Fatalf("expected ErrInvalidModelTimeout, got %v", err)
	}
}

func TestLogModelAdapterReadyDoesNotLogAPIKey(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	logger := log.New(buffer, "", 0)

	LogModelAdapterReady(logger, client.Config{
		Provider:       client.ProviderDeepSeek,
		Model:          "deepseek-chat",
		APIKey:         "secret",
		BaseURL:        "https://api.deepseek.com",
		TimeoutSeconds: 30,
	})

	output := buffer.String()
	if !strings.Contains(output, "provider=deepseek") {
		t.Fatalf("expected provider to be logged, got %q", output)
	}
	if !strings.Contains(output, "model=deepseek-chat") {
		t.Fatalf("expected model to be logged, got %q", output)
	}
	if strings.Contains(output, "secret") {
		t.Fatalf("expected api key to be omitted from logs, got %q", output)
	}
}

func TestLogModelAdapterReadyForRoleLogsRole(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	logger := log.New(buffer, "", 0)

	LogModelAdapterReadyForRole(logger, client.ModelRoleSummarizer, client.Config{
		Provider:       client.ProviderMock,
		TimeoutSeconds: 30,
	})

	output := buffer.String()
	if !strings.Contains(output, "role=summarizer") {
		t.Fatalf("expected role to be logged, got %q", output)
	}
	if !strings.Contains(output, "provider=mock") {
		t.Fatalf("expected provider to be logged, got %q", output)
	}
}
