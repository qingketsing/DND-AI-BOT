# Session Memory Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 Agent 增加 session memory 长期记忆层，并先打通 memory 底座、读链路接入和事件触发更新。

**Architecture:** 新增 `SessionMemory` 模型、repository 和 service，沿用 PostgreSQL + Redis + composite repository 模式；在 `app.NewApp()` 的 agent runner 中读取 memory 并拼接到系统提示词；在 `GameStateService` 的关键事件成功路径中更新 memory。长度触发摘要接口一并预留，但首轮实现不接复杂 summarizer。

**Tech Stack:** Go, PostgreSQL, Redis, composite repository pattern, existing app runner, Go tests

---

### Task 1: Session Memory Domain and Repository Contracts

**Files:**
- Create: `internal/model/session_memory.go`
- Create: `internal/repository/session_memory.go`
- Test: `internal/model/session_memory_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestSessionMemoryStoresCoreFields(t *testing.T) {
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	memory := model.SessionMemory{
		SessionID:        "session-1",
		CharacterSummary: "青稞，精灵法师。",
		SceneSummary:     "位于 the city 的广场。",
		CurrentObjective: "寻找守卫队长格伦。",
		RecentKeyEvents:  []string{"创建角色", "查看布告栏"},
		UpdatedAt:        now,
	}

	if memory.SessionID != "session-1" || memory.CharacterSummary == "" || memory.SceneSummary == "" {
		t.Fatalf("expected session memory core fields to be preserved, got %+v", memory)
	}
	if len(memory.RecentKeyEvents) != 2 {
		t.Fatalf("expected recent events to be preserved, got %+v", memory.RecentKeyEvents)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/model -run TestSessionMemoryStoresCoreFields`
Expected: FAIL with `undefined: model.SessionMemory`

- [ ] **Step 3: Write minimal implementation**

```go
package model

import "time"

type SessionMemory struct {
	SessionID        string
	CharacterSummary string
	SceneSummary     string
	CurrentObjective string
	RecentKeyEvents  []string
	UpdatedAt        time.Time
}
```

And:

```go
package repository

import (
	"context"
	"errors"

	"DND-AI-BOT/internal/model"
)

var ErrSessionMemoryNotFound = errors.New("session memory not found")

type SessionMemoryRepository interface {
	LoadBySessionID(ctx context.Context, sessionID string) (*model.SessionMemory, error)
	Save(ctx context.Context, memory *model.SessionMemory) error
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/model -run TestSessionMemoryStoresCoreFields`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/model/session_memory.go internal/model/session_memory_test.go internal/repository/session_memory.go
git commit -m "update: session memory contracts finished"
```

### Task 2: PostgreSQL, Redis, and Composite Session Memory Repository

**Files:**
- Create: `internal/repository/postgres/session_memory_store.go`
- Create: `internal/repository/postgres/session_memory_store_impl.go`
- Create: `internal/repository/postgres/session_memory_store_test.go`
- Create: `internal/repository/redis/session_memory_cache.go`
- Create: `internal/repository/redis/session_memory_cache_test.go`
- Create: `internal/repository/composite/session_memory_repository.go`
- Create: `internal/repository/composite/session_memory_repository_test.go`
- Create: `migrations/007_create_session_memories.sql`

- [ ] **Step 1: Write the failing repository tests**

```go
func TestPGSessionMemoryStoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := NewPGSessionMemoryStore(testDB(t))
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)

	memory := &model.SessionMemory{
		SessionID:        createSessionFixture(t),
		CharacterSummary: "青稞，精灵法师。",
		SceneSummary:     "the city 广场",
		CurrentObjective: "接取下水道任务",
		RecentKeyEvents:  []string{"创建角色"},
		UpdatedAt:        now,
	}

	if err := store.Save(ctx, memory); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}
	got, err := store.LoadBySessionID(ctx, memory.SessionID)
	if err != nil {
		t.Fatalf("expected load to succeed, got %v", err)
	}
	if got.CharacterSummary != memory.CharacterSummary || got.SceneSummary != memory.SceneSummary {
		t.Fatalf("unexpected memory round trip: %+v", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/repository/postgres ./internal/repository/redis ./internal/repository/composite -run SessionMemory`
Expected: FAIL with undefined store/cache/repository symbols or missing migration table

- [ ] **Step 3: Write minimal repository implementations**

Implement:
- PostgreSQL store with `LoadBySessionID` and `Save`
- Redis cache with `GetBySessionID`, `Set`, `DeleteBySessionID`, `SetNotFound`
- composite repository with read-through/write-through semantics aligned to existing game state/session patterns
- SQL migration:

```sql
create table if not exists session_memories (
  session_id text primary key references sessions(id) on delete cascade,
  character_summary text not null default '',
  scene_summary text not null default '',
  current_objective text not null default '',
  recent_key_events jsonb not null default '[]'::jsonb,
  updated_at timestamptz not null
);
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/repository/postgres ./internal/repository/redis ./internal/repository/composite -run SessionMemory`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add migrations/007_create_session_memories.sql internal/repository/postgres/session_memory_store.go internal/repository/postgres/session_memory_store_impl.go internal/repository/postgres/session_memory_store_test.go internal/repository/redis/session_memory_cache.go internal/repository/redis/session_memory_cache_test.go internal/repository/composite/session_memory_repository.go internal/repository/composite/session_memory_repository_test.go
git commit -m "update: session memory persistence finished"
```

### Task 3: SessionMemoryService

**Files:**
- Create: `internal/service/session_memory_service.go`
- Create: `internal/service/session_memory_service_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestSessionMemoryServiceUpdateAppendsEventAndCapsHistory(t *testing.T) {
	repo := &fakeSessionMemoryRepository{
		memory: &model.SessionMemory{
			SessionID:       "session-1",
			RecentKeyEvents: []string{"e1", "e2", "e3", "e4", "e5"},
		},
	}
	service := NewSessionMemoryService(repo)
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)

	got, err := service.Update(context.Background(), UpdateSessionMemoryInput{
		SessionID:        "session-1",
		SceneSummary:     "the city 广场",
		CurrentObjective: "寻找格伦",
		AppendEvent:      "查看布告栏",
	}, now)
	if err != nil {
		t.Fatalf("expected update to succeed, got %v", err)
	}
	if got.SceneSummary != "the city 广场" || got.CurrentObjective != "寻找格伦" {
		t.Fatalf("expected fields to update, got %+v", got)
	}
	if len(got.RecentKeyEvents) != 5 || got.RecentKeyEvents[4] != "查看布告栏" {
		t.Fatalf("expected capped event history, got %+v", got.RecentKeyEvents)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/service -run SessionMemoryService`
Expected: FAIL with `undefined: NewSessionMemoryService`

- [ ] **Step 3: Write minimal implementation**

Implement:
- `UpdateSessionMemoryInput`
- `MergeSummaryInput`
- `SessionMemoryService`
- `GetBySessionID`
- `Update`
- `MergeSummary`

Rules:
- only overwrite non-empty fields
- append event only when non-empty
- keep newest 5 events
- save through repository every time

- [ ] **Step 4: Run test to verify it passes**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/service -run SessionMemoryService`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/session_memory_service.go internal/service/session_memory_service_test.go
git commit -m "update: session memory service finished"
```

### Task 4: Session Memory Prompt and App Read Path

**Files:**
- Create: `internal/agent/prompt/session_memory.go`
- Create: `internal/agent/prompt/session_memory_test.go`
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestComposeSessionMemoryPromptRendersNonEmptySections(t *testing.T) {
	memory := &model.SessionMemory{
		CharacterSummary: "青稞，精灵法师。",
		SceneSummary:     "位于 the city 广场。",
		CurrentObjective: "寻找格伦。",
		RecentKeyEvents:  []string{"创建角色", "查看布告栏"},
	}

	result := ComposeSessionMemoryPrompt(memory)

	for _, expected := range []string{"当前会话长期记忆", "青稞，精灵法师。", "位于 the city 广场。", "寻找格伦。", "查看布告栏"} {
		if !strings.Contains(result, expected) {
			t.Fatalf("expected %q in session memory prompt, got %q", expected, result)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/agent/prompt ./internal/app -run SessionMemory`
Expected: FAIL with undefined prompt or missing app wiring

- [ ] **Step 3: Write minimal implementation**

Implement `ComposeSessionMemoryPrompt(...)` so it:
- returns empty string for nil/empty memory
- renders only non-empty sections
- limits output to the stored summary fields

Modify `app.NewApp(...)` to:
- build session memory repository/service
- read memory in the agent runner before runtime execution
- append the memory prompt block after warmup composition and before runtime call
- expose `SessionMemoryService` on `App`

- [ ] **Step 4: Run tests to verify they pass**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/agent/prompt ./internal/app -run SessionMemory`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/agent/prompt/session_memory.go internal/agent/prompt/session_memory_test.go internal/app/app.go internal/app/app_test.go
git commit -m "update: session memory prompt finished"
```

### Task 5: Event-Triggered Memory Updates

**Files:**
- Modify: `internal/service/game_state_service.go`
- Modify: `internal/service/game_state_service_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestCreateCharacterUpdatesSessionMemory(t *testing.T) {
	gameStates := &fakeGameStateRepository{}
	memories := &fakeSessionMemoryRepository{}
	service := NewGameStateService(gameStates, NewSessionMemoryService(memories))
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)

	_, err := service.CreateCharacter(context.Background(), CreateCharacterInput{
		SessionID: "session-1",
		Name:      "青稞",
		Race:      "精灵",
		Class:     "法师",
		Level:     1,
		Scene:     "the city 广场",
	}, now)
	if err != nil {
		t.Fatalf("expected create character to succeed, got %v", err)
	}

	got, ok := memories.saved["session-1"]
	if !ok {
		t.Fatal("expected session memory to be updated")
	}
	if !strings.Contains(got.CharacterSummary, "青稞") || !strings.Contains(got.SceneSummary, "the city 广场") {
		t.Fatalf("unexpected memory update %+v", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/service -run 'CreateCharacterUpdatesSessionMemory|SetSceneUpdatesSessionMemory'`
Expected: FAIL because `GameStateService` has no memory dependency or update path

- [ ] **Step 3: Write minimal implementation**

Modify `GameStateService` to optionally accept a `SessionMemoryService`:

```go
func NewGameStateService(repository repository.GameStateRepository, memoryService ...*SessionMemoryService) *GameStateService
```

Behavior:
- preserve current callers by using variadic optional dependency
- after `CreateCharacter(...)` succeeds, update character/scene summary and append event
- after `SetScene(...)` succeeds, update scene summary and append event

- [ ] **Step 4: Run tests to verify they pass**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/service -run 'CreateCharacterUpdatesSessionMemory|SetSceneUpdatesSessionMemory'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/game_state_service.go internal/service/game_state_service_test.go
git commit -m "update: session memory events finished"
```

### Task 6: Sliding-Window Refresh Skeleton

**Files:**
- Create: `internal/service/session_summarizer.go`
- Create: `internal/service/session_memory_refresh_service.go`
- Create: `internal/service/session_memory_refresh_service_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestSessionMemoryRefreshServiceSkipsBelowThreshold(t *testing.T) {
	sessionRepo := &fakeSessionRepository{
		session: &model.Session{
			ID: "session-1",
			History: make([]model.HistoryRecord, 10),
		},
	}
	memoryService := NewSessionMemoryService(&fakeSessionMemoryRepository{})
	refresh := NewSessionMemoryRefreshService(sessionRepo, memoryService, &fakeSessionSummarizer{}, 30, 40)

	if err := refresh.RefreshIfNeeded(context.Background(), "session-1", time.Now().UTC()); err != nil {
		t.Fatalf("expected no error below threshold, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/service -run SessionMemoryRefreshService`
Expected: FAIL with undefined refresh service

- [ ] **Step 3: Write minimal implementation**

Implement:
- `SessionSummarizer`
- `SummarizerMessage`
- `SummaryResult`
- `SessionMemoryRefreshService`
- `RefreshIfNeeded(...)`

First release rules:
- return early below threshold
- build summarizer message list from old history
- call summarizer
- pass result to `MergeSummary(...)`

- [ ] **Step 4: Run tests to verify they pass**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/service -run SessionMemoryRefreshService`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/session_summarizer.go internal/service/session_memory_refresh_service.go internal/service/session_memory_refresh_service_test.go
git commit -m "update: session memory refresh finished"
```
