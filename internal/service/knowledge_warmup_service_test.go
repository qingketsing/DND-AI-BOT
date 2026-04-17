package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"DND-AI-BOT/internal/game/state"
	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
	retrievalsearch "DND-AI-BOT/internal/retrieval/search"
)

func TestBuildWarmupAggregatesBaseAndCharacterWarmups(t *testing.T) {
	ruleSearcher := &fakeWarmupSearcher{
		results: map[string][]retrievalsearch.SearchResult{
			"DND 5e 角色创建 战斗 检定 施法 核心规则": {
				{Content: "规则摘要A"},
				{Content: "规则摘要B"},
			},
			"法师 核心职业特性": {
				{Content: "法师摘要"},
			},
			"精灵 种族特性": {
				{Content: "精灵摘要"},
			},
		},
	}
	loreSearcher := &fakeWarmupSearcher{
		results: map[string][]retrievalsearch.SearchResult{
			"the city 世界设定 城市 天空 滑门 裂隙": {
				{Content: "设定摘要A"},
			},
		},
	}
	repo := &fakeWarmupGameStateRepository{
		state: &state.GameState{
			SessionID: "session-1",
			Player: state.PlayerState{
				Class: "法师",
				Race:  "精灵",
			},
		},
	}

	service := NewKnowledgeWarmupService(ruleSearcher, loreSearcher, repo)
	bundle, err := service.BuildWarmup(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("expected BuildWarmup to succeed, got %v", err)
	}
	if !strings.Contains(bundle.BaseRulesSummary, "规则摘要A") {
		t.Fatalf("expected base rules summary, got %q", bundle.BaseRulesSummary)
	}
	if !strings.Contains(bundle.BaseLoreSummary, "设定摘要A") {
		t.Fatalf("expected base lore summary, got %q", bundle.BaseLoreSummary)
	}
	if !strings.Contains(bundle.CharacterRulesSummary, "法师摘要") || !strings.Contains(bundle.CharacterRulesSummary, "精灵摘要") {
		t.Fatalf("expected character rules summary, got %q", bundle.CharacterRulesSummary)
	}
}

func TestBuildCharacterRulesSummaryReturnsEmptyWhenCharacterMissing(t *testing.T) {
	service := NewKnowledgeWarmupService(&fakeWarmupSearcher{}, &fakeWarmupSearcher{}, &fakeWarmupGameStateRepository{
		err: repository.ErrGameStateNotFound,
	})
	bundle, err := service.BuildWarmup(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("expected BuildWarmup to succeed, got %v", err)
	}
	if bundle.CharacterRulesSummary != "" {
		t.Fatalf("expected empty character summary, got %q", bundle.CharacterRulesSummary)
	}
}

func TestBuildWarmupBestEffortReturnsWarningsInsteadOfError(t *testing.T) {
	service := NewKnowledgeWarmupService(
		&fakeWarmupSearcher{err: errors.New("rules unavailable")},
		&fakeWarmupSearcher{
			results: map[string][]retrievalsearch.SearchResult{
				"the city 世界设定 城市 天空 滑门 裂隙": {
					{Content: "the city 设定摘要"},
				},
			},
		},
		&fakeWarmupGameStateRepository{err: repository.ErrGameStateNotFound},
	)

	result := service.BuildWarmupBestEffort(context.Background(), "session-1")

	if result.Bundle.BaseRulesSummary != "" {
		t.Fatalf("expected failed rules warmup to be empty, got %q", result.Bundle.BaseRulesSummary)
	}
	if !strings.Contains(result.Bundle.BaseLoreSummary, "the city 设定摘要") {
		t.Fatalf("expected lore warmup to survive, got %q", result.Bundle.BaseLoreSummary)
	}
	if len(result.Warnings) == 0 || result.Warnings[0].Source != "base_rules" {
		t.Fatalf("expected base_rules warning, got %+v", result.Warnings)
	}
}

type fakeWarmupSearcher struct {
	results map[string][]retrievalsearch.SearchResult
	err     error
}

func (f *fakeWarmupSearcher) Search(ctx context.Context, request retrievalsearch.SearchRequest) ([]retrievalsearch.SearchResult, error) {
	_ = ctx
	if f.err != nil {
		return nil, f.err
	}
	return f.results[request.Query], nil
}

type fakeWarmupGameStateRepository struct {
	state *state.GameState
	err   error
}

func (f *fakeWarmupGameStateRepository) Save(ctx context.Context, gameState *state.GameState) error {
	_ = ctx
	f.state = gameState
	return nil
}

func (f *fakeWarmupGameStateRepository) LoadBySessionID(ctx context.Context, sessionID string) (*state.GameState, error) {
	_ = ctx
	_ = sessionID
	if f.err != nil {
		return nil, f.err
	}
	if f.state == nil {
		return nil, repository.ErrGameStateNotFound
	}
	return f.state, nil
}

var _ model.WarmupBundle = model.WarmupBundle{}
