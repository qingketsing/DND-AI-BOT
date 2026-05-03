package soak

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	agentclient "DND-AI-BOT/internal/agent/client"
	agentruntime "DND-AI-BOT/internal/agent/runtime"
)

// PlayerInput is the prompt context for generating the next simulated player message.
type PlayerInput struct {
	SessionID     string         `json:"session_id"`
	Scenario      ScenarioConfig `json:"scenario"`
	Round         int            `json:"round"`
	Records       []RoundRecord  `json:"records"`
	PreviousTurns []RoundRecord  `json:"previous_turns"`
	GameObjective string         `json:"game_objective"`
}

// JudgeInput is the prompt context for evaluating one agent reply.
type JudgeInput struct {
	SessionID     string         `json:"session_id"`
	Scenario      ScenarioConfig `json:"scenario"`
	Round         int            `json:"round"`
	UserInput     string         `json:"user_input"`
	AgentReply    string         `json:"agent_reply"`
	Records       []RoundRecord  `json:"records"`
	PreviousTurns []RoundRecord  `json:"previous_turns"`
	HTTPStatus    int            `json:"http_status"`
	LatencyMS     int64          `json:"latency_ms"`
}

// PlayerSimulator generates realistic user messages through an LLM.
type PlayerSimulator struct {
	adapter agentruntime.ModelAdapter
}

// NewPlayerSimulator creates an LLM-backed player simulator.
func NewPlayerSimulator(adapter agentruntime.ModelAdapter) *PlayerSimulator {
	return &PlayerSimulator{adapter: adapter}
}

// NextInput returns the next simulated player input.
func (s *PlayerSimulator) NextInput(ctx context.Context, input PlayerInput) (string, error) {
	if s == nil || s.adapter == nil {
		return "", ErrInvalidRunner
	}
	output, err := s.adapter.Run(ctx, agentruntime.ModelInput{
		SessionID:    input.SessionID,
		SystemPrompt: buildPlayerSystemPrompt(),
		UserMessage:  buildPlayerUserPrompt(input),
	})
	if err != nil {
		return "", err
	}
	reply := strings.TrimSpace(output.Reply)
	if reply == "" {
		return "", fmt.Errorf("player simulator returned empty input")
	}
	return reply, nil
}

// Judge evaluates agent replies through an LLM.
type Judge struct {
	adapter agentruntime.ModelAdapter
}

// LLMJudge is kept as a compatibility alias for existing callers.
type LLMJudge = Judge

// NewJudge creates an LLM-backed judge.
func NewJudge(adapter agentruntime.ModelAdapter) *Judge {
	return &Judge{adapter: adapter}
}

// Evaluate returns a structured judgment for one round.
func (j *Judge) Evaluate(ctx context.Context, input JudgeInput) (JudgeResult, error) {
	if j == nil || j.adapter == nil {
		return JudgeResult{}, ErrInvalidRunner
	}
	output, err := j.adapter.Run(ctx, agentruntime.ModelInput{
		SessionID:    input.SessionID,
		SystemPrompt: buildJudgeSystemPrompt(),
		UserMessage:  buildJudgeUserPrompt(input),
	})
	if err != nil {
		return JudgeResult{}, err
	}
	return parseJudgeResult(output.Reply)
}

// BuildModelAdapter creates an evaluation model adapter from config.
func BuildModelAdapter(config ModelConfig) (agentruntime.ModelAdapter, error) {
	return agentclient.NewModelAdapter(agentclient.Config{
		Provider:       agentclient.Provider(strings.TrimSpace(config.Provider)),
		Model:          strings.TrimSpace(config.Model),
		APIKey:         strings.TrimSpace(config.APIKey),
		BaseURL:        strings.TrimSpace(config.BaseURL),
		TimeoutSeconds: config.TimeoutSeconds,
	})
}

func parseJudgeResult(reply string) (JudgeResult, error) {
	payload := strings.TrimSpace(reply)
	if start := strings.Index(payload, "{"); start >= 0 {
		if end := strings.LastIndex(payload, "}"); end >= start {
			payload = payload[start : end+1]
		}
	}
	var result JudgeResult
	if err := json.Unmarshal([]byte(payload), &result); err != nil {
		return JudgeResult{}, fmt.Errorf("parse judge JSON: %w", err)
	}
	if result.Score < 0 {
		result.Score = 0
	}
	if result.Score > 1 {
		result.Score = 1
	}
	return result, nil
}
