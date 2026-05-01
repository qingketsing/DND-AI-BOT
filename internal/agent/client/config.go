package client

// Provider 表示当前选择的模型厂商类型。
type Provider string

const (
	// ProviderMock 表示使用本地脚本型 mock 模型。
	ProviderMock Provider = "mock"
	// ProviderDeepSeek 表示使用 DeepSeek 模型。
	ProviderDeepSeek Provider = "deepseek"
	// ProviderOpenAI 表示使用 OpenAI 模型。
	ProviderOpenAI Provider = "openai"
)

// Config 定义创建模型适配器时的最小配置。
type Config struct {
	Provider       Provider
	Model          string
	APIKey         string
	BaseURL        string
	TimeoutSeconds int
}

// ModelRole 表示同一应用内不同用途的模型配置角色。
type ModelRole string

const (
	// ModelRolePrimary 用于主 Agent 对话和工具决策。
	ModelRolePrimary ModelRole = "primary"
	// ModelRoleFast 用于轻量状态查询、短回复和后续意图路由。
	ModelRoleFast ModelRole = "fast"
	// ModelRoleSummarizer 用于会话摘要等后台压缩任务。
	ModelRoleSummarizer ModelRole = "summarizer"
)
