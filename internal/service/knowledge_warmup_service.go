package service

import (
	"context"
	"strings"

	"DND-AI-BOT/internal/game/state"
	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
	retrievalsearch "DND-AI-BOT/internal/retrieval/search"
)

type warmupSearcher interface {
	Search(ctx context.Context, request retrievalsearch.SearchRequest) ([]retrievalsearch.SearchResult, error)
}

type warmupGameStateRepository interface {
	LoadBySessionID(ctx context.Context, sessionID string) (*state.GameState, error)
}

// KnowledgeWarmupService 负责在 Agent 运行前构建轻量规则/设定预热摘要。
type KnowledgeWarmupService struct {
	ruleSearcher warmupSearcher
	loreSearcher warmupSearcher
	gameStates   warmupGameStateRepository
}

// NewKnowledgeWarmupService 创建知识预热服务。
func NewKnowledgeWarmupService(ruleSearcher warmupSearcher, loreSearcher warmupSearcher, gameStates warmupGameStateRepository) *KnowledgeWarmupService {
	return &KnowledgeWarmupService{
		ruleSearcher: ruleSearcher,
		loreSearcher: loreSearcher,
		gameStates:   gameStates,
	}
}

// BuildWarmup 聚合规则、设定和角色相关预热摘要。
func (s *KnowledgeWarmupService) BuildWarmup(ctx context.Context, sessionID string) (model.WarmupBundle, error) {
	rules, err := s.buildBaseRulesSummary(ctx)
	if err != nil {
		return model.WarmupBundle{}, err
	}
	lore, err := s.buildBaseLoreSummary(ctx)
	if err != nil {
		return model.WarmupBundle{}, err
	}
	character, err := s.buildCharacterRulesSummary(ctx, sessionID)
	if err != nil {
		return model.WarmupBundle{}, err
	}

	return model.WarmupBundle{
		BaseRulesSummary:      rules,
		BaseLoreSummary:       lore,
		CharacterRulesSummary: character,
	}, nil
}

func (s *KnowledgeWarmupService) buildBaseRulesSummary(ctx context.Context) (string, error) {
	results, err := s.ruleSearcher.Search(ctx, retrievalsearch.SearchRequest{
		Query: "DND 5e 角色创建 战斗 检定 施法 核心规则",
		TopK:  2,
	})
	if err != nil {
		return "", err
	}
	return summarizeWarmupResults(results, 400), nil
}

func (s *KnowledgeWarmupService) buildBaseLoreSummary(ctx context.Context) (string, error) {
	results, err := s.loreSearcher.Search(ctx, retrievalsearch.SearchRequest{
		Query: "the city 世界设定 城市 天空 滑门 裂隙",
		TopK:  2,
	})
	if err != nil {
		return "", err
	}
	return summarizeWarmupResults(results, 400), nil
}

func (s *KnowledgeWarmupService) buildCharacterRulesSummary(ctx context.Context, sessionID string) (string, error) {
	gameState, err := s.gameStates.LoadBySessionID(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		if err == repository.ErrGameStateNotFound {
			return "", nil
		}
		return "", err
	}

	parts := make([]string, 0, 2)
	if strings.TrimSpace(gameState.Player.Class) != "" {
		results, err := s.ruleSearcher.Search(ctx, retrievalsearch.SearchRequest{
			Query: gameState.Player.Class + " 核心职业特性",
			TopK:  1,
		})
		if err != nil {
			return "", err
		}
		parts = append(parts, summarizeWarmupResults(results, 180))
	}
	if strings.TrimSpace(gameState.Player.Race) != "" {
		results, err := s.ruleSearcher.Search(ctx, retrievalsearch.SearchRequest{
			Query: gameState.Player.Race + " 种族特性",
			TopK:  1,
		})
		if err != nil {
			return "", err
		}
		parts = append(parts, summarizeWarmupResults(results, 180))
	}

	return strings.TrimSpace(strings.Join(parts, "\n")), nil
}

func summarizeWarmupResults(results []retrievalsearch.SearchResult, maxChars int) string {
	if len(results) == 0 || maxChars <= 0 {
		return ""
	}

	parts := make([]string, 0, len(results))
	for _, result := range results {
		text := strings.TrimSpace(result.Content)
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	text := strings.Join(parts, "\n")
	runes := []rune(text)
	if len(runes) <= maxChars {
		return text
	}
	return string(runes[:maxChars])
}
