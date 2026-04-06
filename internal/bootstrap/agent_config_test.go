package bootstrap

import (
	"testing"

	"DND-AI-BOT/internal/agent/client"
)

func TestLoadAgentConfigFromEnvLoadsMockConfig(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "mock")
	t.Setenv("MODEL_NAME", "")
	t.Setenv("MODEL_API_KEY", "")
	t.Setenv("MODEL_BASE_URL", "")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "")

	config, err := LoadAgentConfigFromEnv()
	if err != nil {
		t.Fatalf("expected mock config to load, got %v", err)
	}

	if config.Provider != client.ProviderMock {
		t.Fatalf("expected provider %q, got %q", client.ProviderMock, config.Provider)
	}
	if config.TimeoutSeconds != 60 {
		t.Fatalf("expected default timeout 60, got %d", config.TimeoutSeconds)
	}
}

func TestLoadAgentConfigFromEnvLoadsDeepSeekConfig(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "deepseek")
	t.Setenv("MODEL_NAME", "deepseek-chat")
	t.Setenv("MODEL_API_KEY", "secret")
	t.Setenv("MODEL_BASE_URL", "https://api.deepseek.com")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "45")

	config, err := LoadAgentConfigFromEnv()
	if err != nil {
		t.Fatalf("expected deepseek config to load, got %v", err)
	}

	if config.Provider != client.ProviderDeepSeek {
		t.Fatalf("expected provider %q, got %q", client.ProviderDeepSeek, config.Provider)
	}
	if config.Model != "deepseek-chat" {
		t.Fatalf("expected model %q, got %q", "deepseek-chat", config.Model)
	}
	if config.APIKey != "secret" {
		t.Fatalf("expected api key to be propagated")
	}
	if config.BaseURL != "https://api.deepseek.com" {
		t.Fatalf("expected base url to be propagated, got %q", config.BaseURL)
	}
	if config.TimeoutSeconds != 45 {
		t.Fatalf("expected timeout 45, got %d", config.TimeoutSeconds)
	}
}

func TestLoadAgentConfigFromEnvLoadsOpenAIConfig(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "openai")
	t.Setenv("MODEL_NAME", "gpt-5.4-mini")
	t.Setenv("MODEL_API_KEY", "secret")
	t.Setenv("MODEL_BASE_URL", "https://api.openai.com/v1")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "15")

	config, err := LoadAgentConfigFromEnv()
	if err != nil {
		t.Fatalf("expected openai config to load, got %v", err)
	}

	if config.Provider != client.ProviderOpenAI {
		t.Fatalf("expected provider %q, got %q", client.ProviderOpenAI, config.Provider)
	}
	if config.Model != "gpt-5.4-mini" {
		t.Fatalf("expected model %q, got %q", "gpt-5.4-mini", config.Model)
	}
	if config.TimeoutSeconds != 15 {
		t.Fatalf("expected timeout 15, got %d", config.TimeoutSeconds)
	}
}

func TestLoadAgentConfigFromEnvRejectsInvalidProvider(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "unknown")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "60")

	_, err := LoadAgentConfigFromEnv()
	if err != ErrInvalidModelProvider {
		t.Fatalf("expected ErrInvalidModelProvider, got %v", err)
	}
}

func TestLoadAgentConfigFromEnvRejectsInvalidTimeout(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "mock")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "bad")

	_, err := LoadAgentConfigFromEnv()
	if err != ErrInvalidModelTimeout {
		t.Fatalf("expected ErrInvalidModelTimeout, got %v", err)
	}
}

func TestLoadAgentConfigFromEnvUsesDefaultTimeout(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "mock")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "")

	config, err := LoadAgentConfigFromEnv()
	if err != nil {
		t.Fatalf("expected config load to succeed, got %v", err)
	}
	if config.TimeoutSeconds != 60 {
		t.Fatalf("expected default timeout 60, got %d", config.TimeoutSeconds)
	}
}
