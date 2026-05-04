package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	agentclient "DND-AI-BOT/internal/agent/client"
	agentruntime "DND-AI-BOT/internal/agent/runtime"
)

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

type Prelabeler struct {
	adapter agentruntime.ModelAdapter
}

func NewPrelabeler(adapter agentruntime.ModelAdapter) *Prelabeler {
	return &Prelabeler{adapter: adapter}
}

func (p *Prelabeler) Predraft(ctx context.Context, input PrelabelInput) (DraftGoldsetEntry, error) {
	if p == nil || p.adapter == nil {
		return buildDraftEntry(input.Query, input.Candidates, PrelabelResult{}), nil
	}
	output, err := p.adapter.Run(ctx, agentruntime.ModelInput{
		SessionID:    input.Query.ID,
		SystemPrompt: buildPrelabelSystemPrompt(),
		UserMessage:  buildPrelabelUserPrompt(input),
	})
	if err != nil {
		entry := buildDraftEntry(input.Query, input.Candidates, PrelabelResult{})
		entry.Notes = "prelabel failed: " + err.Error()
		return entry, nil
	}
	result, err := parsePrelabelResult(output.Reply)
	if err != nil {
		entry := buildDraftEntry(input.Query, input.Candidates, PrelabelResult{})
		entry.Notes = "prelabel failed: " + err.Error()
		return entry, nil
	}
	return buildDraftEntry(input.Query, input.Candidates, result), nil
}

func parsePrelabelResult(reply string) (PrelabelResult, error) {
	payload := strings.TrimSpace(reply)
	if start := strings.Index(payload, "{"); start >= 0 {
		if end := strings.LastIndex(payload, "}"); end >= start {
			payload = payload[start : end+1]
		}
	}
	var result PrelabelResult
	if err := json.Unmarshal([]byte(payload), &result); err != nil {
		return PrelabelResult{}, fmt.Errorf("parse prelabel JSON: %w", err)
	}
	return result, nil
}

func buildPrelabelSystemPrompt() string {
	return "你是检索评测预标注器。根据 query 和候选 chunks，仅返回 JSON，字段为 relevant_chunk_ids 和 reason。"
}

func buildPrelabelUserPrompt(input PrelabelInput) string {
	var builder strings.Builder
	builder.WriteString("query_id: " + input.Query.ID + "\n")
	builder.WriteString("knowledge_base: " + input.Query.KnowledgeBase + "\n")
	builder.WriteString("query: " + input.Query.Query + "\n")
	builder.WriteString("candidates:\n")
	for _, candidate := range input.Candidates {
		builder.WriteString("- chunk_id: " + candidate.ChunkID + "\n")
		builder.WriteString("  title: " + candidate.Title + "\n")
		builder.WriteString("  content: " + candidate.Content + "\n")
	}
	return builder.String()
}
