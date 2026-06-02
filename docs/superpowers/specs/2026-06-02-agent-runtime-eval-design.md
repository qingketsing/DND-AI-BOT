# Agent Runtime Evaluation Design

## Goal

Build a repeatable Agent Runtime evaluation pipeline for DND AI BOT so the project can measure whether the agent can reliably complete DND tasks, choose the right tools, control ReAct steps, and keep latency within target ranges.

The first release targets single-turn and short-context Agent Runtime evaluation. It does not replace long-session soak eval. Instead, it gives a faster, lower-cost benchmark for prompt, routing, tool, and ReAct changes.

## Current State

The project already has:

- RAG eval under `cmd/rag_eval` and `internal/eval/rag`
- soak eval under `cmd/soak_eval` and `internal/eval/soak`
- Agent Runtime with ReAct-style tool execution
- model routing between primary and fast model roles
- retrieval tools, game state tools, encounter tools, and high-level orchestration tools
- latency metrics and prompt segment metrics in the runtime path

What is still missing is a focused Agent Runtime eval that can answer:

- which request categories succeed or fail
- whether simple requests avoid unnecessary ReAct loops
- whether complex requests call the right tools
- how many steps each task takes
- which model role was selected
- whether latency improves after routing, high-level tools, or max-step changes
- whether failures come from prompt, model output format, tool selection, RAG, state, or timeout

## Scope

This spec covers:

- case dataset format for Agent Runtime eval
- single-case execution against local runtime wiring
- rule-based scoring
- optional LLM judge scoring
- step, tool, model route, and latency recording
- JSON, Markdown, and CSV report output
- failure taxonomy for debugging

This spec does not cover:

- 50-round long conversation soak eval
- browser or frontend behavior
- production traffic replay
- automatic prompt optimization
- full manual annotation UI
- load testing RabbitMQ or async recovery

## Evaluation Model

Agent eval should be treated as a task benchmark, not a pure text-quality benchmark.

Each case defines:

- initial session context
- user message
- expected task outcome
- expected tools
- forbidden tools
- expected answer points
- max step count
- max latency
- expected model role when model routing is enabled

The evaluator runs the case, captures execution traces, scores the result, and writes a report.

## Dataset Design

### Case Count

The first release should use 30 to 50 cases.

Recommended initial distribution:

| Category | Count | Purpose |
| --- | ---: | --- |
| `rules_qa` | 8 | rules retrieval and concise answer |
| `lore_qa` | 6 | lore retrieval and grounded setting answer |
| `character_creation` | 6 | character draft and state progression |
| `scene_progression` | 8 | DM narration and game state update |
| `encounter` | 6 | combat or encounter tool use |
| `ambiguous_followup` | 4 | context-dependent short messages |
| `negative_or_edge` | 4 | malformed, unsupported, or low-signal input |

This distribution keeps the benchmark small enough to run often while still covering the main product paths.

### Case File

Cases are stored in:

```text
configs/eval/agent_cases.jsonl
```

Each line has this shape:

```json
{
  "id": "agent-rules-001",
  "category": "rules_qa",
  "user_message": "1级角色的熟练加值是多少？",
  "session_seed": {
    "session_id": "eval-agent-rules-001",
    "user_id": "eval-user",
    "history": []
  },
  "expected_tools": ["rules_search"],
  "forbidden_tools": ["encounter_resolve", "game_state_update"],
  "expected_answer_points": ["1级", "+2", "熟练加值"],
  "max_steps": 2,
  "max_latency_ms": 10000,
  "expected_model_role": "fast"
}
```

Required fields:

- `id`
- `category`
- `user_message`
- `expected_answer_points`

Optional fields:

- `session_seed`
- `expected_tools`
- `forbidden_tools`
- `expected_state_changes`
- `expected_model_role`
- `max_steps`
- `max_latency_ms`
- `judge_rubric`
- `notes`

## Category Definitions

### `rules_qa`

Rules questions should verify:

- rules retrieval is used when needed
- no unnecessary combat or state mutation tool is called
- answer contains the expected rule points
- simple questions complete within low step and latency limits

Example:

```json
{
  "id": "agent-rules-002",
  "category": "rules_qa",
  "user_message": "山地矮人的属性加成是什么？",
  "expected_tools": ["rules_search"],
  "expected_answer_points": ["体质+2", "力量+2"],
  "max_steps": 2,
  "expected_model_role": "fast"
}
```

### `lore_qa`

Lore questions should verify:

- lore retrieval is used
- answer is grounded in setting chunks
- rules tools are not overused for setting-only queries

### `character_creation`

Character creation cases should verify:

- the agent extracts race, class, background, or ability preferences
- character draft state is created or updated
- missing required fields are asked clearly
- rules retrieval is used only when needed

### `scene_progression`

Scene progression cases should verify:

- the agent responds as DM
- narration is consistent with session state
- relevant game state is updated
- the reply offers meaningful next actions

### `encounter`

Encounter cases should verify:

- combat or encounter tools are selected
- attack, damage, initiative, or status changes are handled consistently
- state updates are persisted
- random or rules-dependent results are explained

### `ambiguous_followup`

Follow-up cases should verify:

- short messages such as "继续", "我看看周围", or "我攻击它" resolve against context
- the agent does not treat the message as an isolated new task

### `negative_or_edge`

Edge cases should verify:

- empty or low-signal input is handled safely
- unsupported requests do not trigger unrelated tools
- max-step and format fallback behavior remains stable

## Execution Design

### Runtime Path

The evaluator should run the same local Agent Runtime wiring used by the application where practical.

The first implementation should prefer an in-process runtime over HTTP:

1. load eval cases
2. create or restore an in-memory or test-backed session
3. run `AgentService` or the runtime runner directly
4. capture reply, steps, tool calls, model route, errors, and latency
5. score the result
6. write reports

This keeps the first eval deterministic and avoids coupling Agent eval to auth, RabbitMQ, or async message polling.

### Trace Capture

Each case record should capture:

- `case_id`
- `category`
- `success`
- `score`
- `reply`
- `steps`
- `tool_calls`
- `model_role`
- `latency_ms`
- `error`
- `failure_reasons`
- `expected_answer_points_hit`
- `expected_tools_hit`
- `forbidden_tools_called`

The runtime already exposes step data. If model role is not directly exposed in the first implementation, the evaluator can record it as `unknown` and mark model-route scoring as skipped for that case. The long-term target is to expose route decision metadata explicitly.

## Scoring Design

The first release uses rule-based scoring as the primary source of truth.

### Required Rule Checks

For each case:

- `reply_present`: assistant reply is non-empty
- `answer_points`: reply contains expected answer points
- `expected_tools`: all required tools were called
- `forbidden_tools`: no forbidden tools were called
- `max_steps`: step count is within case limit
- `max_latency`: latency is within case limit
- `model_role`: actual route matches expected route when available

### Suggested Score

Each case receives a normalized score from 0 to 1.

Default weights:

| Check | Weight |
| --- | ---: |
| reply present | 0.15 |
| expected answer points | 0.30 |
| expected tools | 0.20 |
| forbidden tools | 0.10 |
| max steps | 0.10 |
| max latency | 0.10 |
| model route | 0.05 |

A case is considered successful when:

```text
score >= 0.80
and no critical failure occurred
```

Critical failures:

- no assistant reply
- runtime error
- tool protocol or JSON format failure
- forbidden state-mutating tool called
- max step reached without final answer

## Optional LLM Judge

LLM judge is optional and should not replace rule scoring in the first release.

It can be used for:

- task completion quality
- DND DM style
- hallucination check
- groundedness against tool outputs
- context consistency

Judge config should be loaded from environment:

```sh
export AGENT_EVAL_JUDGE_PROVIDER="<provider>"
export AGENT_EVAL_JUDGE_MODEL="<model>"
export AGENT_EVAL_JUDGE_API_KEY="<api-key>"
export AGENT_EVAL_JUDGE_BASE_URL="<base-url>"
export AGENT_EVAL_JUDGE_TIMEOUT_SECONDS=60
```

If judge config is missing, eval still runs with rule scoring only.

If judge output is malformed, the case should record `judge_error` and continue.

## Metrics

The report should include:

- `case_count`
- `success_rate`
- `average_score`
- `avg_latency_ms`
- `p95_latency_ms`
- `avg_steps`
- `p95_steps`
- `tool_accuracy`
- `forbidden_tool_rate`
- `max_step_exceeded_rate`
- `format_error_rate`
- `loop_rate`
- `model_route_accuracy`

Grouped summaries:

- overall
- by `category`
- by `model_role`
- by failure reason

## Failure Taxonomy

Each failed case should have one or more failure reasons:

- `missing_reply`
- `runtime_error`
- `format_error`
- `wrong_tool`
- `missing_tool`
- `forbidden_tool`
- `missing_answer_point`
- `max_step_exceeded`
- `latency_exceeded`
- `loop_detected`
- `wrong_model_route`
- `rag_grounding_issue`
- `state_update_issue`
- `judge_error`

This taxonomy is more important than the aggregate success rate because it tells the team what to fix next.

## Loop Detection

The evaluator should mark `loop_detected` when either condition is true:

- the same tool is called 3 or more times with materially similar arguments
- the runtime reaches `max_steps` without producing a final answer

The first implementation may use exact tool name and JSON argument equality. Later versions can add normalized argument comparison.

## Reports

The command writes:

```text
reports/eval/agent_eval_report.json
reports/eval/agent_eval_report.md
reports/eval/agent_eval_report.csv
```

Markdown report should include:

- overall metrics
- per-category metrics
- failure reason counts
- slowest cases
- failed cases table

CSV report should include one row per case for quick spreadsheet analysis.

## CLI Design

Command:

```sh
GOCACHE=/tmp/go-build go run ./cmd/agent_eval \
  --cases configs/eval/agent_cases.jsonl \
  --output reports/eval/agent_eval_report.json
```

Useful flags:

- `--cases`
- `--output`
- `--category`
- `--case-id`
- `--max-cases`
- `--judge`
- `--timeout`

Examples:

```sh
GOCACHE=/tmp/go-build go run ./cmd/agent_eval \
  --category rules_qa \
  --output reports/eval/agent_eval_rules.json
```

```sh
GOCACHE=/tmp/go-build go run ./cmd/agent_eval \
  --case-id agent-rules-001 \
  --output reports/eval/agent_eval_single.json
```

## Implementation Components

### `internal/eval/agent/types.go`

Defines:

- `Case`
- `SessionSeed`
- `ExpectedStateChange`
- `CaseResult`
- `Report`
- `MetricsSummary`

### `internal/eval/agent/loader.go`

Loads JSONL cases and validates required fields.

Validation rules:

- duplicate IDs are rejected
- empty `user_message` is rejected unless category is `negative_or_edge`
- `max_steps` defaults to runtime default if missing
- `max_latency_ms` defaults by category if missing

### `internal/eval/agent/runner.go`

Runs one case through the local runtime.

Responsibilities:

- prepare session seed
- call the agent runner
- capture steps and latency
- return raw result

### `internal/eval/agent/scorer.go`

Applies rule scoring and failure taxonomy.

### `internal/eval/agent/report.go`

Writes JSON, Markdown, and CSV reports.

### `cmd/agent_eval`

CLI entry point.

Responsibilities:

- parse flags
- load cases
- build runtime dependencies
- run selected cases
- write reports

## Runtime Dependency Strategy

The first implementation should support two runtime modes:

### `mock` mode

Uses mock model and fake tools where possible.

Purpose:

- fast local verification
- scorer and report tests
- CI safety

Limitation:

- does not measure real model quality

### `live` mode

Uses the same model, retrieval, and tool wiring as the app.

Purpose:

- real Agent eval
- latency and route measurement
- prompt and tool quality validation

Required environment is the same as app startup:

- model provider config
- search backend config
- PostgreSQL when hybrid RAG is enabled
- embedding config when hybrid RAG is enabled

Redis, RabbitMQ, and auth are not required for first-release in-process eval unless a case explicitly depends on persistent stores.

## Testing Strategy

Unit tests:

- case loader validates JSONL
- scorer handles pass, partial pass, and critical failures
- loop detector catches repeated calls
- report writer creates JSON/Markdown/CSV

Integration tests:

- mock runtime runs a small case set
- CLI writes reports

Manual verification:

```sh
GOCACHE=/tmp/go-build go test ./internal/eval/agent ./cmd/agent_eval
```

Live eval example:

```sh
GOCACHE=/tmp/go-build go run ./cmd/agent_eval \
  --cases configs/eval/agent_cases.jsonl \
  --output reports/eval/agent_eval_report.json
```

## Success Criteria

The first Agent eval release is complete when:

- at least 30 cases are defined
- `cmd/agent_eval` can run selected cases
- rule-based score is produced for every case
- report JSON, Markdown, and CSV are written
- failures include actionable reason codes
- eval can be run without LLM judge
- focused tests pass for `internal/eval/agent` and `cmd/agent_eval`

## Future Work

Future extensions:

- compare baseline and optimized runtime in the same report
- add answer groundedness using retrieved chunks and tool outputs
- add model route confusion matrix
- add state diff checks for character and game state cases
- add prompt version metadata
- connect to soak eval for long-session regression tracking
- support production trace replay after privacy filtering

