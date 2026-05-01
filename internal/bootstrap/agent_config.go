package bootstrap

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"DND-AI-BOT/internal/agent/client"
)

const defaultModelTimeoutSeconds = 60

var (
	// ErrMissingModelProvider 表示未配置模型厂商类型。
	ErrMissingModelProvider = errors.New("missing MODEL_PROVIDER")
	// ErrInvalidModelProvider 表示模型厂商类型不在支持列表中。
	ErrInvalidModelProvider = errors.New("invalid MODEL_PROVIDER")
	// ErrInvalidModelTimeout 表示模型超时时间不是合法正整数。
	ErrInvalidModelTimeout = errors.New("invalid MODEL_TIMEOUT_SECONDS")
)

// LoadAgentConfigFromEnv 从环境变量读取模型配置并转换为统一 client.Config。
func LoadAgentConfigFromEnv() (client.Config, error) {
	return LoadAgentConfigFromEnvForRole(client.ModelRolePrimary)
}

// LoadAgentConfigFromEnvForRole 从环境变量读取指定模型角色的配置。
func LoadAgentConfigFromEnvForRole(role client.ModelRole) (client.Config, error) {
	if role == client.ModelRoleFast {
		return loadRoleConfigFromEnv("FAST_MODEL", "MODEL")
	}
	if role == client.ModelRoleSummarizer {
		return loadRoleConfigFromEnv("SUMMARY_MODEL", "MODEL")
	}
	return loadRoleConfigFromEnv("MODEL", "")
}

func loadRoleConfigFromEnv(prefix string, fallbackPrefix string) (client.Config, error) {
	provider, err := parseProvider(envWithFallback(prefix+"_PROVIDER", fallbackPrefix+"_PROVIDER"))
	if err != nil {
		return client.Config{}, err
	}

	timeoutSeconds, err := parseTimeoutSeconds(envWithFallback(prefix+"_TIMEOUT_SECONDS", fallbackPrefix+"_TIMEOUT_SECONDS"))
	if err != nil {
		return client.Config{}, err
	}

	config := normalizeModelConfig(client.Config{
		Provider:       provider,
		Model:          envWithFallback(prefix+"_NAME", fallbackPrefix+"_NAME"),
		APIKey:         envWithFallback(prefix+"_API_KEY", fallbackPrefix+"_API_KEY"),
		BaseURL:        envWithFallback(prefix+"_BASE_URL", fallbackPrefix+"_BASE_URL"),
		TimeoutSeconds: timeoutSeconds,
	})

	if err := validateAgentConfig(config); err != nil {
		return client.Config{}, err
	}

	return config, nil
}

func envWithFallback(name string, fallbackName string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value != "" || strings.TrimSpace(fallbackName) == "" {
		return value
	}
	return strings.TrimSpace(os.Getenv(fallbackName))
}

// parseProvider 将环境变量字符串转换为统一的模型厂商枚举。
func parseProvider(value string) (client.Provider, error) {
	switch strings.TrimSpace(value) {
	case "":
		return "", ErrMissingModelProvider
	case string(client.ProviderMock):
		return client.ProviderMock, nil
	case string(client.ProviderDeepSeek):
		return client.ProviderDeepSeek, nil
	case string(client.ProviderOpenAI):
		return client.ProviderOpenAI, nil
	default:
		return "", ErrInvalidModelProvider
	}
}

// parseTimeoutSeconds 将超时秒数字符串转换为整数，空值时返回默认值。
func parseTimeoutSeconds(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultModelTimeoutSeconds, nil
	}

	timeoutSeconds, err := strconv.Atoi(value)
	if err != nil || timeoutSeconds <= 0 {
		return 0, ErrInvalidModelTimeout
	}

	return timeoutSeconds, nil
}

// normalizeModelConfig 统一清理模型配置中的空白字符并补齐默认值。
func normalizeModelConfig(config client.Config) client.Config {
	config.Model = strings.TrimSpace(config.Model)
	config.APIKey = strings.TrimSpace(config.APIKey)
	config.BaseURL = strings.TrimSpace(config.BaseURL)
	if config.TimeoutSeconds <= 0 {
		config.TimeoutSeconds = defaultModelTimeoutSeconds
	}

	return config
}

// validateAgentConfig 复用 client 层的配置校验逻辑，确保模型配置完整可用。
func validateAgentConfig(config client.Config) error {
	return client.ValidateConfig(config)
}
