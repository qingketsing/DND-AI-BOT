package client

import (
	"time"

	"DND-AI-BOT/internal/agent/client/deepseek"
	"DND-AI-BOT/internal/agent/client/mock"
	"DND-AI-BOT/internal/agent/client/openai"
)

// ValidateConfig 校验模型适配器创建所需的最小配置。
func ValidateConfig(config Config) error {
	if config.Provider == "" {
		return ErrInvalidClientConfig
	}

	switch config.Provider {
	case ProviderMock:
		return nil
	case ProviderDeepSeek, ProviderOpenAI:
		if config.Model == "" || config.APIKey == "" {
			return ErrInvalidClientConfig
		}
		return nil
	default:
		return ErrUnsupportedProvider
	}
}

// NewModelAdapter 根据配置创建对应厂商的模型适配器。
func NewModelAdapter(config Config) (ModelAdapter, error) {
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}

	switch config.Provider {
	case ProviderMock:
		return mock.NewAdapter(nil), nil
	case ProviderDeepSeek:
		return deepseek.NewAdapter(config.Model, config.BaseURL, config.APIKey, timeoutFromSeconds(config.TimeoutSeconds))
	case ProviderOpenAI:
		return openai.NewAdapter(config.Model, config.BaseURL, config.APIKey, timeoutFromSeconds(config.TimeoutSeconds))
	default:
		return nil, ErrUnsupportedProvider
	}
}

func timeoutFromSeconds(seconds int) time.Duration {
	if seconds <= 0 {
		return 60 * time.Second
	}
	return time.Duration(seconds) * time.Second
}
