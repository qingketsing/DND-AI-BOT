package tools

import (
	"context"

	agentcontext "DND-AI-BOT/internal/agent/context"
)

const defaultAgentContextLimit = 40

type contextToolProvider interface {
	BuildContext(ctx context.Context, sessionID string, limit int) (agentcontext.AgentContext, error)
}

// GetAgentContextResult 表示会话上下文工具的结构化返回结果。
type GetAgentContextResult struct {
	SessionID     string                    `json:"session_id"`
	Channel       string                    `json:"channel"`
	RecentRecords []any                     `json:"recent_records"`
	LastRecord    map[string]any            `json:"last_record,omitempty"`
	RawContext    agentcontext.AgentContext `json:"-"`
}

// GetAgentContextTool 用于读取当前会话的最小上下文。
type GetAgentContextTool struct {
	provider contextToolProvider
}

type getAgentContextArgs struct {
	Limit int `json:"limit"`
}

// NewGetAgentContextTool 创建会话上下文读取工具。
func NewGetAgentContextTool(provider contextToolProvider) *GetAgentContextTool {
	return &GetAgentContextTool{provider: provider}
}

// Spec 返回会话上下文工具的元信息描述。
func (t *GetAgentContextTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "get_agent_context",
		Description: "读取当前会话的最近消息、最后一条消息和接入渠道",
		InputSchema: objectSchema(map[string]any{
			"limit": map[string]any{
				"type":        "integer",
				"description": "最近消息条数限制",
			},
		}),
	}
}

// Call 解析输入并调用上下文提供器返回结构化上下文。
func (t *GetAgentContextTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	var args getAgentContextArgs
	if err := decodeToolInput(input.Raw, &args); err != nil {
		return CallOutput{}, err
	}
	if args.Limit <= 0 {
		args.Limit = defaultAgentContextLimit
	}

	result, err := t.provider.BuildContext(ctx, input.SessionID, args.Limit)
	if err != nil {
		return CallOutput{}, err
	}

	output := GetAgentContextResult{
		SessionID:  result.SessionID,
		Channel:    string(result.Channel),
		RawContext: result,
	}
	if result.LastRecord != nil {
		output.LastRecord = map[string]any{
			"id":       result.LastRecord.ID,
			"sequence": result.LastRecord.Sequence,
			"source":   string(result.LastRecord.Source),
			"user": map[string]any{
				"id":   result.LastRecord.User.ID,
				"name": result.LastRecord.User.Name,
			},
			"message": map[string]any{
				"content": result.LastRecord.Message.Content,
			},
		}
	}
	for _, record := range result.RecentRecords {
		output.RecentRecords = append(output.RecentRecords, map[string]any{
			"id":       record.ID,
			"sequence": record.Sequence,
			"source":   string(record.Source),
			"user": map[string]any{
				"id":   record.User.ID,
				"name": record.User.Name,
			},
			"message": map[string]any{
				"content": record.Message.Content,
			},
		})
	}

	return newToolOutput(t.Spec().Name, output), nil
}
