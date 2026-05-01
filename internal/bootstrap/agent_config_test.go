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

func TestLoadAgentConfigFromEnvForRolePrimaryUsesModelEnv(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "mock")
	t.Setenv("MODEL_NAME", "")
	t.Setenv("MODEL_API_KEY", "")
	t.Setenv("MODEL_BASE_URL", "")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "")

	config, err := LoadAgentConfigFromEnvForRole(client.ModelRolePrimary)
	if err != nil {
		t.Fatalf("expected primary config to load, got %v", err)
	}
	if config.Provider != client.ProviderMock {
		t.Fatalf("expected provider %q, got %q", client.ProviderMock, config.Provider)
	}
}

func TestLoadAgentConfigFromEnvForRoleSummarizerFallsBackToPrimary(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "deepseek")
	t.Setenv("MODEL_NAME", "deepseek-v4-flash")
	t.Setenv("MODEL_API_KEY", "primary-secret")
	t.Setenv("MODEL_BASE_URL", "https://primary.example.com")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "60")
	t.Setenv("SUMMARY_MODEL_PROVIDER", "")
	t.Setenv("SUMMARY_MODEL_NAME", "")
	t.Setenv("SUMMARY_MODEL_API_KEY", "")
	t.Setenv("SUMMARY_MODEL_BASE_URL", "")
	t.Setenv("SUMMARY_MODEL_TIMEOUT_SECONDS", "")

	config, err := LoadAgentConfigFromEnvForRole(client.ModelRoleSummarizer)
	if err != nil {
		t.Fatalf("expected summarizer fallback config to load, got %v", err)
	}
	if config.Provider != client.ProviderDeepSeek || config.Model != "deepseek-v4-flash" {
		t.Fatalf("expected summarizer to fallback to primary model config, got %+v", config)
	}
	if config.APIKey != "primary-secret" || config.BaseURL != "https://primary.example.com" || config.TimeoutSeconds != 60 {
		t.Fatalf("expected summarizer to fallback to primary credentials, got %+v", config)
	}
}

func TestLoadAgentConfigFromEnvForRoleSummarizerOverridesConfiguredFields(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "deepseek")
	t.Setenv("MODEL_NAME", "deepseek-v4-flash")
	t.Setenv("MODEL_API_KEY", "primary-secret")
	t.Setenv("MODEL_BASE_URL", "https://primary.example.com")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "60")
	t.Setenv("SUMMARY_MODEL_PROVIDER", "openai")
	t.Setenv("SUMMARY_MODEL_NAME", "gpt-5.4-mini")
	t.Setenv("SUMMARY_MODEL_API_KEY", "summary-secret")
	t.Setenv("SUMMARY_MODEL_BASE_URL", "https://summary.example.com")
	t.Setenv("SUMMARY_MODEL_TIMEOUT_SECONDS", "20")

	config, err := LoadAgentConfigFromEnvForRole(client.ModelRoleSummarizer)
	if err != nil {
		t.Fatalf("expected summarizer config to load, got %v", err)
	}
	if config.Provider != client.ProviderOpenAI {
		t.Fatalf("expected provider %q, got %q", client.ProviderOpenAI, config.Provider)
	}
	if config.Model != "gpt-5.4-mini" || config.APIKey != "summary-secret" || config.BaseURL != "https://summary.example.com" {
		t.Fatalf("expected summary-specific config, got %+v", config)
	}
	if config.TimeoutSeconds != 20 {
		t.Fatalf("expected summary timeout 20, got %d", config.TimeoutSeconds)
	}
}

func TestLoadAgentConfigFromEnvForRoleFastFallsBackToPrimary(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "deepseek")
	t.Setenv("MODEL_NAME", "deepseek-v4-flash")
	t.Setenv("MODEL_API_KEY", "primary-secret")
	t.Setenv("MODEL_BASE_URL", "https://primary.example.com")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "60")
	t.Setenv("FAST_MODEL_PROVIDER", "")
	t.Setenv("FAST_MODEL_NAME", "")
	t.Setenv("FAST_MODEL_API_KEY", "")
	t.Setenv("FAST_MODEL_BASE_URL", "")
	t.Setenv("FAST_MODEL_TIMEOUT_SECONDS", "")

	config, err := LoadAgentConfigFromEnvForRole(client.ModelRoleFast)
	if err != nil {
		t.Fatalf("expected fast fallback config to load, got %v", err)
	}
	if config.Provider != client.ProviderDeepSeek || config.Model != "deepseek-v4-flash" {
		t.Fatalf("expected fast model to fallback to primary model config, got %+v", config)
	}
	if config.APIKey != "primary-secret" || config.BaseURL != "https://primary.example.com" || config.TimeoutSeconds != 60 {
		t.Fatalf("expected fast model to fallback to primary credentials, got %+v", config)
	}
}

func TestLoadAgentConfigFromEnvForRoleFastOverridesConfiguredFields(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "deepseek")
	t.Setenv("MODEL_NAME", "deepseek-v4-flash")
	t.Setenv("MODEL_API_KEY", "primary-secret")
	t.Setenv("MODEL_BASE_URL", "https://primary.example.com")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "60")
	t.Setenv("FAST_MODEL_PROVIDER", "openai")
	t.Setenv("FAST_MODEL_NAME", "gpt-5.4-mini")
	t.Setenv("FAST_MODEL_API_KEY", "fast-secret")
	t.Setenv("FAST_MODEL_BASE_URL", "https://fast.example.com")
	t.Setenv("FAST_MODEL_TIMEOUT_SECONDS", "10")

	config, err := LoadAgentConfigFromEnvForRole(client.ModelRoleFast)
	if err != nil {
		t.Fatalf("expected fast config to load, got %v", err)
	}
	if config.Provider != client.ProviderOpenAI {
		t.Fatalf("expected provider %q, got %q", client.ProviderOpenAI, config.Provider)
	}
	if config.Model != "gpt-5.4-mini" || config.APIKey != "fast-secret" || config.BaseURL != "https://fast.example.com" {
		t.Fatalf("expected fast-specific config, got %+v", config)
	}
	if config.TimeoutSeconds != 10 {
		t.Fatalf("expected fast timeout 10, got %d", config.TimeoutSeconds)
	}
}
