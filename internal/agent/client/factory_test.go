package client

import (
	"testing"

	"DND-AI-BOT/internal/agent/runtime"
)

func TestNewModelAdapterReturnsMockAdapter(t *testing.T) {
	adapter, err := NewModelAdapter(Config{
		Provider: ProviderMock,
	})
	if err != nil {
		t.Fatalf("expected mock adapter creation to succeed, got %v", err)
	}
	if adapter == nil {
		t.Fatal("expected adapter to be created")
	}

	if _, ok := adapter.(ModelAdapter); !ok {
		t.Fatal("expected adapter to implement ModelAdapter")
	}

	_, err = adapter.Run(t.Context(), runtime.ModelInput{
		SessionID:   "session-1",
		UserMessage: "hello",
	})
	if err == nil {
		t.Fatal("expected empty mock adapter to report no more outputs")
	}
}

func TestNewModelAdapterRejectsUnsupportedProvider(t *testing.T) {
	_, err := NewModelAdapter(Config{
		Provider: Provider("unknown"),
	})
	if err != ErrUnsupportedProvider {
		t.Fatalf("expected ErrUnsupportedProvider, got %v", err)
	}
}

func TestValidateConfigRejectsInvalidConfig(t *testing.T) {
	if err := ValidateConfig(Config{}); err != ErrInvalidClientConfig {
		t.Fatalf("expected ErrInvalidClientConfig for empty provider, got %v", err)
	}

	if err := ValidateConfig(Config{Provider: ProviderDeepSeek}); err != ErrInvalidClientConfig {
		t.Fatalf("expected ErrInvalidClientConfig for missing deepseek model, got %v", err)
	}

	if err := ValidateConfig(Config{Provider: ProviderMock}); err != nil {
		t.Fatalf("expected mock config to be valid, got %v", err)
	}

	if err := ValidateConfig(Config{
		Provider: ProviderDeepSeek,
		Model:    "deepseek-chat",
		APIKey:   "secret",
	}); err != nil {
		t.Fatalf("expected deepseek config to be valid, got %v", err)
	}

	if err := ValidateConfig(Config{
		Provider: ProviderOpenAI,
		Model:    "gpt-5.4-mini",
		APIKey:   "secret",
	}); err != nil {
		t.Fatalf("expected openai config to be valid, got %v", err)
	}
}

func TestNewModelAdapterReturnsDeepSeekAdapter(t *testing.T) {
	adapter, err := NewModelAdapter(Config{
		Provider: ProviderDeepSeek,
		Model:    "deepseek-chat",
		APIKey:   "secret",
	})
	if err != nil {
		t.Fatalf("expected deepseek adapter creation to succeed, got %v", err)
	}
	if adapter == nil {
		t.Fatal("expected deepseek adapter to be created")
	}
}

func TestNewModelAdapterReturnsOpenAIAdapter(t *testing.T) {
	adapter, err := NewModelAdapter(Config{
		Provider: ProviderOpenAI,
		Model:    "gpt-5.4-mini",
		APIKey:   "secret",
	})
	if err != nil {
		t.Fatalf("expected openai adapter creation to succeed, got %v", err)
	}
	if adapter == nil {
		t.Fatal("expected openai adapter to be created")
	}
}
