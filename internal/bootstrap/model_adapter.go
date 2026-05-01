package bootstrap

import (
	"log"

	"DND-AI-BOT/internal/agent/client"
)

// BuildModelAdapter 根据已解析的模型配置创建统一模型适配器。
func BuildModelAdapter(config client.Config) (client.ModelAdapter, error) {
	if err := client.ValidateConfig(config); err != nil {
		return nil, err
	}

	return client.NewModelAdapter(config)
}

// BuildModelAdapterFromEnv 从环境变量加载模型配置并创建统一模型适配器。
func BuildModelAdapterFromEnv() (client.ModelAdapter, client.Config, error) {
	return BuildModelAdapterFromEnvForRole(client.ModelRolePrimary)
}

// BuildModelAdapterFromEnvForRole 从环境变量加载指定角色的模型配置并创建适配器。
func BuildModelAdapterFromEnvForRole(role client.ModelRole) (client.ModelAdapter, client.Config, error) {
	config, err := LoadAgentConfigFromEnvForRole(role)
	if err != nil {
		return nil, client.Config{}, err
	}

	adapter, err := BuildModelAdapter(config)
	if err != nil {
		return nil, client.Config{}, err
	}

	return adapter, config, nil
}

// LogModelAdapterReady 输出安全的模型适配器初始化摘要，不记录敏感信息。
func LogModelAdapterReady(logger *log.Logger, config client.Config) {
	LogModelAdapterReadyForRole(logger, "", config)
}

// LogModelAdapterReadyForRole 输出带角色的模型适配器初始化摘要，不记录敏感信息。
func LogModelAdapterReadyForRole(logger *log.Logger, role client.ModelRole, config client.Config) {
	if logger == nil {
		return
	}

	if role == "" {
		logger.Printf(
			"model adapter ready: provider=%s model=%s timeout_seconds=%d custom_base_url=%t",
			config.Provider,
			config.Model,
			config.TimeoutSeconds,
			hasCustomBaseURL(config),
		)
		return
	}

	logger.Printf(
		"model adapter ready: role=%s provider=%s model=%s timeout_seconds=%d custom_base_url=%t",
		role,
		config.Provider,
		config.Model,
		config.TimeoutSeconds,
		hasCustomBaseURL(config),
	)
}

func hasCustomBaseURL(config client.Config) bool {
	return config.BaseURL != ""
}
