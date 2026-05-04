# RAG Recall Evaluation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go-based retrieval benchmark that can prelabel query relevance candidates with an LLM, load an approved goldset, and compare lexical versus hybrid retrieval with `Recall@K` and `MRR`.

**Architecture:** Reuse the existing retrieval/search code and the soak-eval style model adapter pattern. Split the work into focused units: dataset and goldset models, metric calculation, report writers, prelabel workflow, searcher/runtime wiring, and the `cmd/rag_eval` CLI.

**Tech Stack:** Go, JSONL, existing retrieval/search package, PostgreSQL/pgvector runtime search stack, existing model adapter patterns

---

### Task 1: Dataset models, loaders, and metrics

**Files:**
- Create: `internal/eval/rag/types.go`
- Create: `internal/eval/rag/query_loader.go`
- Create: `internal/eval/rag/query_loader_test.go`
- Create: `internal/eval/rag/metrics.go`
- Create: `internal/eval/rag/metrics_test.go`

- [ ] **Step 1: Write failing loader tests**

Cover:
- loading queries from JSONL
- loading approved goldset entries from JSONL
- rejecting malformed JSONL
- filtering non-approved entries for final eval

- [ ] **Step 2: Run loader tests and verify they fail**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/eval/rag -run 'TestLoadQueries|TestLoadGoldset'
```

- [ ] **Step 3: Implement dataset and goldset types plus JSONL loaders**

Add:
- `Query`
- `DraftGoldsetEntry`
- `GoldsetEntry`
- `LoadQueries(...)`
- `LoadGoldset(...)`

- [ ] **Step 4: Re-run loader tests**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/eval/rag -run 'TestLoadQueries|TestLoadGoldset'
```

- [ ] **Step 5: Write failing metric tests**

Cover:
- `Recall@K`
- `MRR`
- zero-hit behavior
- grouped aggregation

- [ ] **Step 6: Run metric tests and verify they fail**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/eval/rag -run 'TestRecallAtK|TestMRR|TestAggregateMetrics'
```

- [ ] **Step 7: Implement metric calculation**

Add:
- per-query metric helpers
- aggregate report builders for overall and grouped metrics

- [ ] **Step 8: Re-run focused rag tests**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/eval/rag -run 'TestLoadQueries|TestLoadGoldset|TestRecallAtK|TestMRR|TestAggregateMetrics'
```

### Task 2: Report writers and prelabel model integration

**Files:**
- Create: `internal/eval/rag/report.go`
- Create: `internal/eval/rag/report_test.go`
- Create: `internal/eval/rag/llm.go`
- Create: `internal/eval/rag/llm_test.go`
- Create: `internal/eval/rag/prelabeler.go`
- Create: `internal/eval/rag/prelabeler_test.go`

- [ ] **Step 1: Write failing report tests**

Cover:
- JSON report output
- Markdown summary output
- CSV row output

- [ ] **Step 2: Run report tests and verify they fail**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/eval/rag -run 'TestWriteJSONReport|TestWriteMarkdownReport|TestWriteCSVReport'
```

- [ ] **Step 3: Implement report writers**

Add report serialization helpers for:
- JSON
- Markdown
- CSV

- [ ] **Step 4: Re-run report tests**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/eval/rag -run 'TestWriteJSONReport|TestWriteMarkdownReport|TestWriteCSVReport'
```

- [ ] **Step 5: Write failing LLM/prelabel tests**

Cover:
- prelabel model adapter creation
- parsing prelabel JSON
- prelabel failure fallback creating draft entries with empty relevant IDs

- [ ] **Step 6: Run LLM/prelabel tests and verify they fail**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/eval/rag -run 'TestBuildModelAdapter|TestParsePrelabelResult|TestPrelabeler'
```

- [ ] **Step 7: Implement model adapter wrapper and prelabeler**

Add:
- `ModelConfig`
- `BuildModelAdapter(...)`
- `Prelabeler`
- prelabel prompt builders
- prelabel JSON parser

- [ ] **Step 8: Re-run focused rag tests**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/eval/rag -run 'TestBuildModelAdapter|TestParsePrelabelResult|TestPrelabeler|TestWriteJSONReport|TestWriteMarkdownReport|TestWriteCSVReport'
```

### Task 3: Search runtime wiring and evaluation runner

**Files:**
- Create: `internal/eval/rag/searchers.go`
- Create: `internal/eval/rag/searchers_test.go`
- Create: `internal/eval/rag/evaluator.go`
- Create: `internal/eval/rag/evaluator_test.go`

- [ ] **Step 1: Write failing searcher wiring tests**

Cover:
- lexical searcher set creation
- hybrid searcher set creation with runtime dependencies
- candidate pool merge and dedupe

- [ ] **Step 2: Run wiring tests and verify they fail**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/eval/rag -run 'TestBuildLexicalSearchers|TestMergeCandidatePools'
```

- [ ] **Step 3: Implement searcher construction helpers**

Add:
- lexical searcher builder
- hybrid searcher builder
- merged candidate helper for prelabel mode

- [ ] **Step 4: Re-run wiring tests**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/eval/rag -run 'TestBuildLexicalSearchers|TestMergeCandidatePools'
```

- [ ] **Step 5: Write failing evaluator tests**

Cover:
- side-by-side lexical/hybrid execution
- per-query record creation
- grouped metric output

- [ ] **Step 6: Run evaluator tests and verify they fail**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/eval/rag -run 'TestEvaluator|TestBuildEvalReport'
```

- [ ] **Step 7: Implement evaluator**

Add:
- lexical/hybrid comparison runner
- record builders
- report builder integration

- [ ] **Step 8: Re-run evaluator tests**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/eval/rag -run 'TestEvaluator|TestBuildEvalReport|TestBuildLexicalSearchers|TestMergeCandidatePools'
```

### Task 4: CLI, sample data, and docs

**Files:**
- Create: `cmd/rag_eval/main.go`
- Create: `cmd/rag_eval/main_test.go`
- Create: `docs/eval/rag-eval.md`
- Create: `configs/eval/rag_queries_rules.jsonl`
- Create: `configs/eval/rag_queries_lore.jsonl`

- [ ] **Step 1: Write failing CLI tests**

Cover:
- mode parsing
- missing required flags for `eval`
- default output path behavior

- [ ] **Step 2: Run CLI tests and verify they fail**

Run:

```bash
GOCACHE=/tmp/go-build go test ./cmd/rag_eval -run 'TestParseFlags|TestEvalRequiresGoldset'
```

- [ ] **Step 3: Implement CLI**

Support:
- `--mode prelabel`
- `--mode eval`
- query path flags
- goldset path flag
- output path flag

- [ ] **Step 4: Add sample query files and docs**

Seed the repository with starter benchmark files and usage documentation.

- [ ] **Step 5: Re-run CLI and package tests**

Run:

```bash
GOCACHE=/tmp/go-build go test ./cmd/rag_eval ./internal/eval/rag
```

- [ ] **Step 6: Run broader verification**

Run:

```bash
GOCACHE=/tmp/go-build go test ./cmd/rag_eval ./internal/eval/rag ./internal/retrieval/search ./internal/bootstrap
git diff --check
```

- [ ] **Step 7: Commit**

```bash
git add cmd/rag_eval internal/eval/rag docs/eval/rag-eval.md configs/eval/rag_queries_rules.jsonl configs/eval/rag_queries_lore.jsonl docs/superpowers/plans/2026-05-04-rag-recall-eval.md
git commit -m "update: Add RAG recall evaluation pipeline"
```
