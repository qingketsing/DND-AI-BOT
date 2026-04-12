# Knowledge Warmup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在每轮 Agent 运行前注入简短的规则与设定预热摘要，并在已有角色时追加职业/种族相关规则摘要。

**Architecture:** 新增 `KnowledgeWarmupService` 复用现有 `ruleSearcher`、`loreSearcher` 和 `gameStateRepository` 生成短摘要；新增 `prompt.ComposeSystemPrompt` 将 warmup 拼接到系统提示词；在 `app.NewApp()` 的 runner 包装层接入。预热只提供短摘要，不替代正式检索。

**Tech Stack:** Go, existing retrieval/search package, existing app runner, existing game state repository, Go tests

---

### Task 1: Warmup Prompt Composer

**Files:**
- Create: `internal/agent/prompt/warmup.go`
- Create: `internal/agent/prompt/warmup_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestComposeSystemPromptAppendsWarmupSections(t *testing.T) {
	base := "base prompt"
	warmup := service.WarmupBundle{
		BaseRulesSummary:      "rules summary",
		BaseLoreSummary:       "lore summary",
		CharacterRulesSummary: "character summary",
	}

	result := ComposeSystemPrompt(base, warmup)

	for _, expected := range []string{
		"base prompt",
		"已知基础上下文",
		"rules summary",
		"lore summary",
		"character summary",
	} {
		if !strings.Contains(result, expected) {
			t.Fatalf("expected %q in composed prompt, got %q", expected, result)
		}
	}
}

func TestComposeSystemPromptSkipsEmptySections(t *testing.T) {
	result := ComposeSystemPrompt("base prompt", service.WarmupBundle{})
	if strings.Contains(result, "已知基础上下文") {
		t.Fatalf("expected no warmup header when bundle is empty, got %q", result)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/agent/prompt`
Expected: FAIL with `undefined: ComposeSystemPrompt`

- [ ] **Step 3: Write minimal implementation**

```go
package prompt

import (
	"strings"

	"DND-AI-BOT/internal/service"
)

func ComposeSystemPrompt(basePrompt string, warmup service.WarmupBundle) string {
	sections := []string{strings.TrimSpace(basePrompt)}
	warmupLines := make([]string, 0, 3)
	if strings.TrimSpace(warmup.BaseRulesSummary) != "" {
		warmupLines = append(warmupLines, "[规则摘要]\n"+strings.TrimSpace(warmup.BaseRulesSummary))
	}
	if strings.TrimSpace(warmup.BaseLoreSummary) != "" {
		warmupLines = append(warmupLines, "[设定摘要]\n"+strings.TrimSpace(warmup.BaseLoreSummary))
	}
	if strings.TrimSpace(warmup.CharacterRulesSummary) != "" {
		warmupLines = append(warmupLines, "[角色相关规则]\n"+strings.TrimSpace(warmup.CharacterRulesSummary))
	}
	if len(warmupLines) == 0 {
		return strings.TrimSpace(basePrompt)
	}

	sections = append(sections, "已知基础上下文：\n"+strings.Join(warmupLines, "\n\n"))
	return strings.TrimSpace(strings.Join(sections, "\n\n"))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/agent/prompt`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/agent/prompt/warmup.go internal/agent/prompt/warmup_test.go
git commit -m "update: warmup prompt composer finished"
```

### Task 2: Knowledge Warmup Service

**Files:**
- Create: `internal/service/knowledge_warmup_service.go`
- Create: `internal/service/knowledge_warmup_service_test.go`

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/service`
Expected: FAIL with `undefined: NewKnowledgeWarmupService`

- [ ] **Step 3: Write minimal implementation**

```go
package service

import (
	"context"
	"strings"

	"DND-AI-BOT/internal/game/state"
	"DND-AI-BOT/internal/repository"
	retrievalsearch "DND-AI-BOT/internal/retrieval/search"
)

type WarmupBundle struct {
	BaseRulesSummary      string
	BaseLoreSummary       string
	CharacterRulesSummary string
}

type warmupSearcher interface {
	Search(ctx context.Context, request retrievalsearch.SearchRequest) ([]retrievalsearch.SearchResult, error)
}

type warmupGameStateRepository interface {
	LoadBySessionID(ctx context.Context, sessionID string) (*state.GameState, error)
}

type KnowledgeWarmupService struct {
	ruleSearcher warmupSearcher
	loreSearcher warmupSearcher
	gameStates   warmupGameStateRepository
}

func NewKnowledgeWarmupService(ruleSearcher warmupSearcher, loreSearcher warmupSearcher, gameStates warmupGameStateRepository) *KnowledgeWarmupService {
	return &KnowledgeWarmupService{
		ruleSearcher: ruleSearcher,
		loreSearcher: loreSearcher,
		gameStates:   gameStates,
	}
}

func (s *KnowledgeWarmupService) BuildWarmup(ctx context.Context, sessionID string) (WarmupBundle, error) {
	rules, err := s.buildBaseRulesSummary(ctx)
	if err != nil {
		return WarmupBundle{}, err
	}
	lore, err := s.buildBaseLoreSummary(ctx)
	if err != nil {
		return WarmupBundle{}, err
	}
	character, err := s.buildCharacterRulesSummary(ctx, sessionID)
	if err != nil {
		return WarmupBundle{}, err
	}
	return WarmupBundle{
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
	var builder strings.Builder
	for _, result := range results {
		text := strings.TrimSpace(result.Content)
		if text == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(text)
		if builder.Len() >= maxChars {
			break
		}
	}
	text := builder.String()
	if len([]rune(text)) <= maxChars {
		return text
	}
	return string([]rune(text)[:maxChars])
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/service`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/knowledge_warmup_service.go internal/service/knowledge_warmup_service_test.go
git commit -m "update: knowledge warmup service finished"
```

### Task 3: Wire Warmup Into App Runner

**Files:**
- Modify: `internal/app/app.go`
- Test: `internal/app/app_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestNewAppBuildsKnowledgeWarmupService(t *testing.T) {
	deps := newTestRuntimeDependencies(t)
	application, err := NewApp(deps)
	if err != nil {
		t.Fatalf("expected NewApp to succeed, got %v", err)
	}
	if application.KnowledgeWarmupService == nil {
		t.Fatal("expected KnowledgeWarmupService to be initialized")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/app`
Expected: FAIL with missing `KnowledgeWarmupService`

- [ ] **Step 3: Write minimal implementation**

```go
type App struct {
	Handler                http.Handler
	AgentService           *service.AgentService
	AuthService            *service.AuthService
	SessionService         *service.SessionService
	GameStateService       *service.GameStateService
	EncounterService       *service.EncounterService
	KnowledgeWarmupService *service.KnowledgeWarmupService
}
```

在 `NewApp(...)` 中新增：

```go
knowledgeWarmupService := service.NewKnowledgeWarmupService(
	searchRuntime.RuleSearcher,
	searchRuntime.LoreSearcher,
	gameStateRepository,
)
```

在 runner 中替换：

```go
systemPrompt := input.SystemPrompt
warmup, err := knowledgeWarmupService.BuildWarmup(ctx, input.SessionID)
if err != nil {
	return service.AgentReplyResult{}, err
}
systemPrompt = prompt.ComposeSystemPrompt(systemPrompt, warmup)
```

然后传给 runtime：

```go
SystemPrompt: systemPrompt,
```

最后挂到返回值：

```go
KnowledgeWarmupService: knowledgeWarmupService,
```

- [ ] **Step 4: Run test to verify it passes**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/app ./internal/service ./internal/agent/prompt`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/app.go internal/app/app_test.go
git commit -m "update: knowledge warmup wiring finished"
```

