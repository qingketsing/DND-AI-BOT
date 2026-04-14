# Hybrid RAG Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a PostgreSQL-backed hybrid retrieval stack using FTS, pgvector, RRF fusion, and Qwen embedding API while preserving the existing `Searcher` interface.

**Architecture:** Keep the current retrieval callers unchanged and replace the retrieval core underneath `Searcher`. Add a PostgreSQL knowledge index, an embedding abstraction with a Qwen-backed provider, a hybrid search store, an RRF fusion layer, and an index build command. Rollout remains configuration-driven with lexical fallback.

**Tech Stack:** Go, PostgreSQL, pgvector, FTS/GIN, RRF, HTTP embedding API, existing bootstrap/runtime wiring

---

## File Map

### Create

- `migrations/008_enable_pgvector_and_create_knowledge_chunks.sql`
- `internal/retrieval/search/index.go`
- `internal/retrieval/search/embedding.go`
- `internal/retrieval/search/embedding_test.go`
- `internal/retrieval/search/qwen_embedder.go`
- `internal/retrieval/search/qwen_embedder_test.go`
- `internal/retrieval/search/postgres_store.go`
- `internal/retrieval/search/postgres_store_impl.go`
- `internal/retrieval/search/postgres_store_test.go`
- `internal/retrieval/search/fusion.go`
- `internal/retrieval/search/fusion_test.go`
- `internal/retrieval/search/hybrid.go`
- `internal/retrieval/search/hybrid_test.go`
- `internal/retrieval/search/indexer.go`
- `internal/retrieval/search/indexer_test.go`
- `scripts/rag/build_hybrid_index.go`

### Modify

- `internal/retrieval/search/defaults.go`
- `internal/retrieval/search/types.go`
- `internal/bootstrap/search_runtime.go`
- `internal/bootstrap/search_runtime_test.go`
- `internal/service/knowledge_warmup_service_test.go`

### Keep Unchanged Contract

- `internal/agent/tools/retrieval_tools.go`
- `internal/service/knowledge_warmup_service.go`

---

### Task 1: Add Knowledge Index Schema

**Files:**
- Create: `migrations/008_enable_pgvector_and_create_knowledge_chunks.sql`
- Test: `internal/repository/postgres/migration_smoke_test.go` or existing migration test harness if available

- [ ] Write the migration for `vector` extension and `knowledge_chunks`
- [ ] Add indexes for `knowledge_base`, `content_tsv`, and `embedding`
- [ ] Run migration in local development DB
- [ ] Verify schema exists and `pgvector` is enabled
- [ ] Commit:

```bash
git add migrations/008_enable_pgvector_and_create_knowledge_chunks.sql
git commit -m "update: hybrid rag schema finished"
```

---

### Task 2: Define Retrieval Index and Embedding Contracts

**Files:**
- Create: `internal/retrieval/search/index.go`
- Create: `internal/retrieval/search/embedding.go`
- Create: `internal/retrieval/search/embedding_test.go`

- [ ] Add `IndexedChunk`
- [ ] Add `Embedder`
- [ ] Add `EmbeddingConfig`
- [ ] Add `EmbedQuery(...)`
- [ ] Write tests for single-query embedding helper and empty result validation
- [ ] Run:

```bash
GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/retrieval/search -run 'EmbedQuery'
```

- [ ] Commit:

```bash
git add internal/retrieval/search/index.go internal/retrieval/search/embedding.go internal/retrieval/search/embedding_test.go
git commit -m "update: hybrid rag contracts finished"
```

---

### Task 3: Implement Qwen Embedding Provider

**Files:**
- Create: `internal/retrieval/search/qwen_embedder.go`
- Create: `internal/retrieval/search/qwen_embedder_test.go`

- [ ] Implement HTTP client for Qwen embedding API
- [ ] Support batching through `Embed(ctx, texts []string)`
- [ ] Validate vector dimension against config
- [ ] Add tests with fake HTTP server covering:
  - success
  - HTTP error
  - malformed response
  - dimension mismatch
- [ ] Run:

```bash
GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/retrieval/search -run 'QwenEmbedder'
```

- [ ] Commit:

```bash
git add internal/retrieval/search/qwen_embedder.go internal/retrieval/search/qwen_embedder_test.go
git commit -m "update: qwen embedding provider finished"
```

---

### Task 4: Implement PostgreSQL Hybrid Search Store

**Files:**
- Create: `internal/retrieval/search/postgres_store.go`
- Create: `internal/retrieval/search/postgres_store_impl.go`
- Create: `internal/retrieval/search/postgres_store_test.go`

- [ ] Add `HybridSearchStore`, `VectorSearchRequest`, and `ScoredCandidate`
- [ ] Implement `UpsertChunks(...)`
- [ ] Implement `SearchFTS(...)`
- [ ] Implement `SearchVector(...)`
- [ ] Add store tests for:
  - upsert round-trip
  - knowledge-base filter
  - FTS recall
  - vector recall ordering
- [ ] Run:

```bash
GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/retrieval/search -run 'Postgres'
```

- [ ] Commit:

```bash
git add internal/retrieval/search/postgres_store.go internal/retrieval/search/postgres_store_impl.go internal/retrieval/search/postgres_store_test.go
git commit -m "update: hybrid rag store finished"
```

---

### Task 5: Implement RRF Fusion

**Files:**
- Create: `internal/retrieval/search/fusion.go`
- Create: `internal/retrieval/search/fusion_test.go`

- [ ] Add `FusionStrategy`
- [ ] Implement `RRFFusion`
- [ ] Add tests for:
  - overlap between lexical and vector results
  - deduplication by chunk id
  - topK truncation
- [ ] Run:

```bash
GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/retrieval/search -run 'RRF|Fusion'
```

- [ ] Commit:

```bash
git add internal/retrieval/search/fusion.go internal/retrieval/search/fusion_test.go
git commit -m "update: hybrid rag fusion finished"
```

---

### Task 6: Implement HybridSearcher

**Files:**
- Create: `internal/retrieval/search/hybrid.go`
- Create: `internal/retrieval/search/hybrid_test.go`
- Modify: `internal/retrieval/search/types.go`

- [ ] Implement `HybridSearcher`
- [ ] Execute FTS and vector recall inside `Search(...)`
- [ ] Fuse results with `RRF`
- [ ] Add fallback behavior:
  - vector fails -> use FTS
  - FTS fails -> use vector
  - both fail -> return error
- [ ] Add tests with fake embedder and fake store
- [ ] Run:

```bash
GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/retrieval/search -run 'HybridSearcher'
```

- [ ] Commit:

```bash
git add internal/retrieval/search/hybrid.go internal/retrieval/search/hybrid_test.go internal/retrieval/search/types.go
git commit -m "update: hybrid rag searcher finished"
```

---

### Task 7: Add Index Build Pipeline

**Files:**
- Create: `internal/retrieval/search/indexer.go`
- Create: `internal/retrieval/search/indexer_test.go`
- Create: `scripts/rag/build_hybrid_index.go`

- [ ] Implement `buildEmbeddingText(...)`
- [ ] Implement `Indexer.BuildIndex(...)`
- [ ] Add command-line loader for:
  - rules
  - lore
  - all
- [ ] Add tests for embedding text composition and indexing orchestration
- [ ] Run:

```bash
GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/retrieval/search -run 'Indexer'
```

- [ ] Commit:

```bash
git add internal/retrieval/search/indexer.go internal/retrieval/search/indexer_test.go scripts/rag/build_hybrid_index.go
git commit -m "update: hybrid rag indexer finished"
```

---

### Task 8: Bootstrap and Configuration Wiring

**Files:**
- Modify: `internal/retrieval/search/defaults.go`
- Modify: `internal/bootstrap/search_runtime.go`
- Modify: `internal/bootstrap/search_runtime_test.go`

- [ ] Add configuration-driven backend selection:
  - `lexical`
  - `hybrid`
- [ ] Add embedding config loading from env
- [ ] Build lexical searchers when configured for lexical
- [ ] Build hybrid searchers when configured for hybrid
- [ ] Add optional fallback to lexical when hybrid init fails
- [ ] Update bootstrap tests
- [ ] Run:

```bash
GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/bootstrap ./internal/retrieval/search
```

- [ ] Commit:

```bash
git add internal/retrieval/search/defaults.go internal/bootstrap/search_runtime.go internal/bootstrap/search_runtime_test.go
git commit -m "update: hybrid rag bootstrap finished"
```

---

### Task 9: Warmup and Retrieval Compatibility Regression

**Files:**
- Modify: `internal/service/knowledge_warmup_service_test.go`
- Optionally modify: `internal/service/knowledge_warmup_service.go` only if result handling requires adaptation

- [ ] Confirm `KnowledgeWarmupService` still works against the same `Searcher` interface
- [ ] Add or update tests to run warmup against hybrid-style result shapes
- [ ] Run:

```bash
GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/service ./internal/agent/tools ./internal/bootstrap ./internal/app
```

- [ ] Commit:

```bash
git add internal/service/knowledge_warmup_service_test.go internal/service/knowledge_warmup_service.go
git commit -m "update: hybrid rag integration finished"
```

---

### Task 10: End-to-End Index and Retrieval Verification

**Files:**
- No new runtime code required unless defects are found

- [ ] Run hybrid index build against local data
- [ ] Verify representative queries:
  - exact rule term
  - paraphrased rule question
  - named lore term
  - paraphrased lore question
- [ ] Compare lexical-only vs hybrid results
- [ ] Record any retrieval regressions before rollout
- [ ] Commit any final fixes:

```bash
git commit -m "update: hybrid rag verification finished"
```

---

## Verification Commands

Primary package-level verification:

```bash
GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/retrieval/search ./internal/bootstrap ./internal/service ./internal/agent/tools ./internal/app
```

Migration and runtime verification:

```bash
docker compose up -d postgres redis
go run ./scripts/rag/build_hybrid_index.go --knowledge-base all
```

---

## Rollout Notes

- default first deployment can remain `SEARCH_BACKEND=lexical`
- build and validate the hybrid index before switching production traffic
- only flip to `hybrid` after query-quality verification succeeds
