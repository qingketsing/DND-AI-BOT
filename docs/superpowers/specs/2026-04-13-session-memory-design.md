# Session Memory Design

## Goal

Build a production-oriented session memory layer for the DND AI BOT backend so the agent can preserve long-running session facts without relying only on the raw chat transcript or the most recent context window.

This design addresses three concrete product issues:

- the agent forgets established scene facts after enough turns
- the agent re-asks for information that was already collected earlier in the same session
- long sessions become increasingly expensive and unstable if the system depends only on an ever-growing raw message window

The target outcome is a three-layer context model:

1. recent raw messages for short-term detail
2. session memory for long-term summarized context
3. structured game state for durable factual state such as character, inventory, quests, and scene fields

## Scope

This spec covers:

- the `session memory` data model
- repository and service boundaries
- event-triggered memory updates
- sliding-window summary refresh
- how session memory is injected into the agent prompt path

This spec does not cover:

- complete roleplay workflow redesign
- onebot integration
- shared multi-user session membership
- replacing the current runtime with a third-party agent framework
- advanced reranking or retrieval architecture changes

## Product Rules

### Session Memory Purpose

Session memory stores long-lived conversation facts that should survive beyond the recent raw message window.

It is not a replacement for:

- raw session history
- structured game state
- RAG retrieval

Instead, it acts as the compact long-term context layer between them.

### What Session Memory Must Capture

For the first release, session memory must store:

- `CharacterSummary`
- `SceneSummary`
- `CurrentObjective`
- `RecentKeyEvents`

These fields are intentionally small and human-readable. They are designed to support agent reasoning, not full archival fidelity.

### Update Strategy

Session memory must be updated by a hybrid strategy:

1. **event-triggered updates**
   - immediately after important state changes
2. **length-triggered summary refresh**
   - when session history exceeds a configured threshold

Event-triggered updates are higher priority. They protect key facts from being lost before the transcript grows large enough to trigger summarization.

### Read Strategy

Before each agent run, the backend should provide the agent with:

1. knowledge warmup
2. session memory
3. recent raw messages
4. structured tools and game state access

Session memory is treated as the authoritative summary of established session context, but not as a replacement for direct tool lookups when exact state is required.

## Architecture

The design introduces a new `SessionMemory` domain object and supporting repository/service layer. The service is responsible for merging updates and keeping key event history bounded. Event-triggered write paths update the memory after durable state changes succeed. A separate refresh service handles length-triggered summarization from the session transcript.

At read time, `app.NewApp(...)` will load session memory before each agent run and append a compact memory block to the effective system prompt. This keeps the runtime contract stable and avoids invasive changes to the model adapter or runtime loop.

The resulting context model becomes:

- `knowledge warmup`: rules and lore baseline
- `session memory`: durable summary of current session context
- `recent raw history`: short-term conversational detail
- `game state + tools`: exact structured state and mutation path

## Data Model

### `SessionMemory`

File:

```text
internal/model/session_memory.go
```

Structure:

```go
type SessionMemory struct {
	SessionID        string
	CharacterSummary string
	SceneSummary     string
	CurrentObjective string
	RecentKeyEvents  []string
	UpdatedAt        time.Time
}
```

Rules:

- `SessionID` is unique per session
- `RecentKeyEvents` is append-only from the service perspective, but bounded to the most recent N items
- first release keeps at most 5 recent events
- memory fields may be empty for a new session

## Repository Design

### Interface

File:

```text
internal/repository/session_memory.go
```

```go
type SessionMemoryRepository interface {
	LoadBySessionID(ctx context.Context, sessionID string) (*model.SessionMemory, error)
	Save(ctx context.Context, memory *model.SessionMemory) error
}
```

Recommended error:

```go
var ErrSessionMemoryNotFound = errors.New("session memory not found")
```

### Storage Strategy

First release should follow the existing repository pattern:

- PostgreSQL as source of truth
- Redis as cache
- composite repository for read-through/write-through behavior

Expected implementation files:

- `internal/repository/postgres/session_memory_store.go`
- `internal/repository/postgres/session_memory_store_impl.go`
- `internal/repository/redis/session_memory_cache.go`
- `internal/repository/composite/session_memory_repository.go`

### Persistence Shape

PostgreSQL table design can remain small:

```sql
create table session_memories (
  session_id text primary key references sessions(id) on delete cascade,
  character_summary text not null default '',
  scene_summary text not null default '',
  current_objective text not null default '',
  recent_key_events jsonb not null default '[]'::jsonb,
  updated_at timestamptz not null
);
```

Rules:

- `on delete cascade` keeps memory aligned with session deletion
- `recent_key_events` stored as JSON array
- Redis value may mirror the model structure directly

## Service Design

### `SessionMemoryService`

File:

```text
internal/service/session_memory_service.go
```

```go
type UpdateSessionMemoryInput struct {
	SessionID        string
	CharacterSummary string
	SceneSummary     string
	CurrentObjective string
	AppendEvent      string
}

type MergeSummaryInput struct {
	SessionID        string
	CharacterSummary string
	SceneSummary     string
	CurrentObjective string
	RecentKeyEvents  []string
}

type SessionMemoryService struct {
	repository repository.SessionMemoryRepository
}

func NewSessionMemoryService(repository repository.SessionMemoryRepository) *SessionMemoryService

func (s *SessionMemoryService) GetBySessionID(ctx context.Context, sessionID string) (*model.SessionMemory, error)

func (s *SessionMemoryService) Update(ctx context.Context, input UpdateSessionMemoryInput, now time.Time) (*model.SessionMemory, error)

func (s *SessionMemoryService) MergeSummary(ctx context.Context, input MergeSummaryInput, now time.Time) (*model.SessionMemory, error)
```

Behavior rules:

- `GetBySessionID` returns an empty in-memory default if no row exists yet, or propagates a repository not-found error based on chosen repo convention
- `Update` only overwrites fields with non-empty values
- `AppendEvent` appends only when non-empty
- `RecentKeyEvents` is capped to the newest 5 entries
- `MergeSummary` is used by the summarizer flow and may overwrite all summary fields in one call

### Event-Triggered Update Policy

The following operations must update session memory after they succeed:

- `CreateCharacter(...)`
- `SetScene(...)`
- quest creation or update
- major objective changes

This integration should happen in service-layer success paths, not only in prompt logic.

Examples:

- after character creation:
  - update `CharacterSummary`
  - set initial `SceneSummary` if present
  - append an event noting character creation
- after scene update:
  - update `SceneSummary`
  - append an event noting scene change
- after quest acceptance:
  - update `CurrentObjective`
  - append an event describing the accepted task

## Summarization Design

### Summarizer Interface

File:

```text
internal/service/session_summarizer.go
```

```go
type SessionSummarizer interface {
	SummarizeMessages(ctx context.Context, sessionID string, messages []SummarizerMessage) (SummaryResult, error)
}

type SummarizerMessage struct {
	Source  string
	User    string
	Content string
}

type SummaryResult struct {
	CharacterSummary string
	SceneSummary     string
	CurrentObjective string
	RecentKeyEvents  []string
}
```

The summarizer abstraction exists so the first implementation can be simple while leaving room for later replacement.

First release options:

- LLM-backed summarizer using the current model provider
- deterministic rule-based summarizer for constrained cases

The interface should be introduced now even if the first implementation is intentionally basic.

### `SessionMemoryRefreshService`

File:

```text
internal/service/session_memory_refresh_service.go
```

```go
type SessionMemoryRefreshService struct {
	sessions         repository.SessionRepository
	memoryService    *SessionMemoryService
	summarizer       SessionSummarizer
	messageWindow    int
	summaryThreshold int
}

func NewSessionMemoryRefreshService(
	sessions repository.SessionRepository,
	memoryService *SessionMemoryService,
	summarizer SessionSummarizer,
	messageWindow int,
	summaryThreshold int,
) *SessionMemoryRefreshService

func (s *SessionMemoryRefreshService) RefreshIfNeeded(ctx context.Context, sessionID string, now time.Time) error
```

Behavior:

1. load the session
2. if history length is below `summaryThreshold`, return immediately
3. keep the newest `messageWindow` records untouched
4. summarize older records
5. merge summary into session memory

Recommended first-release defaults:

- `messageWindow = 30`
- `summaryThreshold = 40`

This first release should use message-count thresholds instead of estimated token counts. Token-aware tuning can come later.

## Agent Read Path

### Prompt Injection

Session memory should be injected before each agent run as a compact memory block.

New helper:

File:

```text
internal/agent/prompt/session_memory.go
```

```go
func ComposeSessionMemoryPrompt(memory *model.SessionMemory) string
```

Example rendered block:

```text
当前会话长期记忆：
- 角色：...
- 场景：...
- 当前目标：...
- 最近关键事件：
  1. ...
  2. ...
```

Rules:

- skip empty sections
- keep output concise
- do not duplicate raw chat transcript

### App Integration

File:

```text
internal/app/app.go
```

`NewApp(...)` should:

- construct the session memory repository
- construct `SessionMemoryService`
- load session memory before agent execution
- append the memory prompt block to the effective system prompt

This mirrors the current warmup integration pattern and avoids runtime API churn.

## Integration Plan

### Phase 1: Memory Foundation

Implement:

- `SessionMemory` model
- repository interfaces and concrete repos
- `SessionMemoryService`

Success criteria:

- session memory can be loaded and saved independently of agent runs

### Phase 2: Read-Path Integration

Implement:

- session memory prompt composition
- `app.go` integration into the agent runner path

Success criteria:

- each agent run can see the current session memory block

### Phase 3: Event-Triggered Updates

Integrate updates into:

- character creation
- scene update
- quest or objective updates

Success criteria:

- key facts enter memory immediately after important state changes

### Phase 4: Length-Triggered Summarization

Implement:

- summarizer interface
- refresh service
- threshold-based refresh flow

Success criteria:

- old session history can be summarized while preserving recent raw context

## Testing Strategy

### Repository Tests

Cover:

- save/load round trip
- not-found behavior
- Redis cache read-through behavior
- session deletion cascading behavior in PostgreSQL

### Service Tests

Cover:

- update merges only non-empty fields
- append event behavior
- event cap at 5 items
- merge summary overwrite semantics

### Prompt Tests

Cover:

- memory prompt renders non-empty sections
- empty sections are skipped
- formatting remains concise and deterministic

### Integration Tests

Cover:

- agent runner receives session memory in the effective prompt
- character creation updates memory
- scene update updates memory
- refresh service summarizes only when threshold exceeded

## Risks and Controls

### Risk: Summary Drift

If summaries are generated too freely, the memory may diverge from actual state.

Control:

- keep structured state in `game state`
- treat session memory as narrative summary, not canonical inventory/state storage
- update memory from service success paths for high-value facts

### Risk: Prompt Bloat

If memory grows unchecked, it creates the same cost problem as raw history.

Control:

- keep only the newest 5 events
- keep summary fields concise
- avoid copying large transcript sections into memory

### Risk: Duplicate Responsibility with Game State

If the same information is persisted in multiple places without discipline, contradictions can increase.

Control:

- game state remains canonical for structured state
- session memory remains canonical for long-lived session summary
- prompt policy must prefer exact tools when precision is required

## Recommended Next Step

After this spec is approved, implementation should begin with:

1. `SessionMemory` model and repository
2. `SessionMemoryService`
3. read-path integration in `app.go`

Sliding-window summarization should come after the memory foundation is already readable by the agent.
