# Hybrid RAG Index Build Pipeline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the remaining offline index build pipeline so existing rules and lore JSONL chunks can be embedded with the configured Qwen 8B model and loaded into PostgreSQL for hybrid retrieval.

**Architecture:** Keep runtime retrieval unchanged and finish the missing ingestion side. Add an index metadata table, a small metadata store, an `Indexer` that batches chunk embedding and upsert, and a dedicated `cmd/build_hybrid_index` entrypoint that rebuilds `rules`, `lore`, or both. The first release only supports offline full rebuild per knowledge base.

**Tech Stack:** Go, PostgreSQL, pgvector, existing chunk loaders, Qwen embedding API, current `HybridSearchStore`

---

## Scope of This Plan

Already implemented and **not** part of this plan:

- `knowledge_chunks` schema
- Qwen 8B embedder
- PostgreSQL hybrid query store
- `RRF` fusion
- `HybridSearcher`
- bootstrap/runtime wiring behind `SEARCH_BACKEND`

This plan only covers the remaining ingestion work:

- index metadata schema and store
- embedding text composition
- indexer orchestration
- CLI build command
- verification path for first real index build

## File Map

### Create

- `migrations/009_create_knowledge_index_metadata.sql`
- `internal/retrieval/search/indexer.go`
- `internal/retrieval/search/indexer_test.go`
- `internal/retrieval/search/index_metadata_store.go`
- `internal/retrieval/search/index_metadata_store_test.go`
- `cmd/build_hybrid_index/main.go`

### Modify

- `internal/retrieval/search/index.go`
- `internal/retrieval/search/postgres_store_impl.go`
- `internal/retrieval/search/postgres_store_test.go`
- `internal/bootstrap/search_runtime.go`
- `docs/superpowers/specs/2026-04-14-hybrid-rag-design.md`

### Keep Unchanged Contract

- `internal/retrieval/search/hybrid.go`
- `internal/agent/tools/retrieval_tools.go`
- `internal/service/knowledge_warmup_service.go`

---

### Task 1: Add Knowledge Index Metadata Schema

**Files:**
- Create: `migrations/009_create_knowledge_index_metadata.sql`

- [ ] **Step 1: Write the migration**

```sql
create table knowledge_index_metadata (
  knowledge_base text primary key,
  embedding_provider text not null,
  embedding_model text not null,
  embedding_dim integer not null,
  built_at timestamptz not null
);
```

- [ ] **Step 2: Run migration locally**

Run: `docker compose up -d postgres`
Expected: PostgreSQL is healthy and accepts migrations.

- [ ] **Step 3: Apply migrations through the existing app startup or migration path**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/app -run TestNewApp`
Expected: PASS and schema creation path does not fail.

- [ ] **Step 4: Commit**

```bash
git add migrations/009_create_knowledge_index_metadata.sql
git commit -m "update: hybrid rag index metadata schema finished"
```

---

### Task 2: Add Index Metadata Store

**Files:**
- Create: `internal/retrieval/search/index_metadata_store.go`
- Create: `internal/retrieval/search/index_metadata_store_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestPostgresIndexMetadataStoreUpsertAndLoad(t *testing.T) {
    db := postgres.NewFakeKnowledgePGDB(t, postgres.NewFakeKnowledgePGState())
    store := NewPostgresIndexMetadataStore(db)
    now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)

    err := store.UpsertIndexMetadata(context.Background(), IndexMetadata{
        KnowledgeBase:     KnowledgeBaseRules,
        EmbeddingProvider: "qwen",
        EmbeddingModel:    "Qwen/Qwen3-Embedding-8B",
        EmbeddingDim:      1024,
        BuiltAt:           now,
    })
    if err != nil {
        t.Fatalf("UpsertIndexMetadata() error = %v", err)
    }

    got, err := store.LoadIndexMetadata(context.Background(), KnowledgeBaseRules)
    if err != nil {
        t.Fatalf("LoadIndexMetadata() error = %v", err)
    }
    if got.EmbeddingModel != "Qwen/Qwen3-Embedding-8B" {
        t.Fatalf("EmbeddingModel = %q, want %q", got.EmbeddingModel, "Qwen/Qwen3-Embedding-8B")
    }
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/retrieval/search -run TestPostgresIndexMetadataStoreUpsertAndLoad`
Expected: FAIL because store does not exist yet.

- [ ] **Step 3: Implement the store**

```go
type IndexMetadata struct {
    KnowledgeBase     string
    EmbeddingProvider string
    EmbeddingModel    string
    EmbeddingDim      int
    BuiltAt           time.Time
}

type IndexMetadataStore interface {
    UpsertIndexMetadata(ctx context.Context, metadata IndexMetadata) error
    LoadIndexMetadata(ctx context.Context, knowledgeBase string) (*IndexMetadata, error)
}

type PostgresIndexMetadataStore struct {
    db *sql.DB
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/retrieval/search -run TestPostgresIndexMetadataStoreUpsertAndLoad`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/retrieval/search/index_metadata_store.go internal/retrieval/search/index_metadata_store_test.go
git commit -m "update: hybrid rag index metadata store finished"
```

---

### Task 3: Add Embedding Text Builder and Indexer Contracts

**Files:**
- Modify: `internal/retrieval/search/index.go`
- Create: `internal/retrieval/search/indexer.go`
- Create: `internal/retrieval/search/indexer_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestBuildEmbeddingTextIncludesTitleAliasesTagsAndContent(t *testing.T) {
    chunk := SearchChunk{
        Title:   "法师准备法术",
        Aliases: []string{"prepared spells", "法术准备"},
        Tags:    []string{"wizard", "spellcasting"},
        Content: "法师在长休后准备法术。",
    }

    got := buildEmbeddingText(chunk)

    for _, want := range []string{
        "title: 法师准备法术",
        "aliases: prepared spells, 法术准备",
        "tags: wizard, spellcasting",
        "content: 法师在长休后准备法术。",
    } {
        if !strings.Contains(got, want) {
            t.Fatalf("embedding text missing %q: %s", want, got)
        }
    }
}
```

```go
func TestBuildIndexedChunksRejectsVectorCountMismatch(t *testing.T) {
    _, err := buildIndexedChunks([]SearchChunk{{ChunkID: "c1"}}, nil, time.Now())
    if err == nil {
        t.Fatal("expected error, got nil")
    }
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/retrieval/search -run 'BuildEmbeddingText|BuildIndexedChunks'`
Expected: FAIL because the helpers do not exist yet.

- [ ] **Step 3: Implement the indexer contracts**

```go
type ChunkSource interface {
    Load(ctx context.Context) ([]SearchChunk, error)
}

type Indexer struct {
    store         HybridSearchStore
    metadataStore IndexMetadataStore
    embedder      Embedder
    config        EmbeddingConfig
    now           func() time.Time
}

func NewIndexer(
    store HybridSearchStore,
    metadataStore IndexMetadataStore,
    embedder Embedder,
    config EmbeddingConfig,
) *Indexer
```

- [ ] **Step 4: Implement the helpers**

```go
func buildEmbeddingText(chunk SearchChunk) string
func buildIndexedChunks(chunks []SearchChunk, vectors [][]float32, now time.Time) ([]IndexedChunk, error)
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/retrieval/search -run 'BuildEmbeddingText|BuildIndexedChunks'`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/retrieval/search/index.go internal/retrieval/search/indexer.go internal/retrieval/search/indexer_test.go
git commit -m "update: hybrid rag indexer contracts finished"
```

---

### Task 4: Implement Indexer Orchestration

**Files:**
- Create: `internal/retrieval/search/indexer.go`
- Create: `internal/retrieval/search/indexer_test.go`

- [ ] **Step 1: Write the failing orchestration test**

```go
func TestIndexerBuildIndexEmbedsUpsertsAndWritesMetadata(t *testing.T) {
    store := &fakeHybridSearchStore{}
    metadata := &fakeIndexMetadataStore{}
    embedder := &fakeEmbedder{vectors: [][]float32{{0.1, 0.2}}}

    indexer := NewIndexer(store, metadata, embedder, EmbeddingConfig{
        Provider: "qwen",
        Model:    "Qwen/Qwen3-Embedding-8B",
        Dim:      2,
    })
    indexer.now = func() time.Time { return time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC) }

    err := indexer.BuildIndex(context.Background(), KnowledgeBaseRules, []SearchChunk{{
        ChunkID:       "chunk-1",
        KnowledgeBase: KnowledgeBaseRules,
        DocumentID:    "doc-1",
        Title:         "法师准备法术",
        Content:       "法师在长休后准备法术。",
    }})
    if err != nil {
        t.Fatalf("BuildIndex() error = %v", err)
    }
    if len(store.upserted) != 1 {
        t.Fatalf("upserted chunks = %d, want 1", len(store.upserted))
    }
    if metadata.last.EmbeddingModel != "Qwen/Qwen3-Embedding-8B" {
        t.Fatalf("metadata model = %q", metadata.last.EmbeddingModel)
    }
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/retrieval/search -run TestIndexerBuildIndexEmbedsUpsertsAndWritesMetadata`
Expected: FAIL because `BuildIndex` is not complete yet.

- [ ] **Step 3: Implement `BuildIndex` and `BuildIndexFromSource`**

```go
func (i *Indexer) BuildIndex(ctx context.Context, knowledgeBase string, chunks []SearchChunk) error
func (i *Indexer) BuildIndexFromSource(ctx context.Context, knowledgeBase string, source ChunkSource) error
```

Behavior:

- validate `knowledgeBase`
- build embedding texts in original chunk order
- call `Embed(...)` in batches using `config.BatchSize`
- convert vectors into `IndexedChunk`
- call `UpsertChunks(...)`
- write `IndexMetadata`

- [ ] **Step 4: Run the test to verify it passes**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/retrieval/search -run TestIndexerBuildIndexEmbedsUpsertsAndWritesMetadata`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/retrieval/search/indexer.go internal/retrieval/search/indexer_test.go
git commit -m "update: hybrid rag indexer orchestration finished"
```

---

### Task 5: Add the CLI Entry Point

**Files:**
- Create: `cmd/build_hybrid_index/main.go`

- [ ] **Step 1: Write a small smoke test or command parsing test if feasible**

```go
func TestParseKnowledgeBaseArg(t *testing.T) {
    got, err := parseKnowledgeBaseArg([]string{"--knowledge-base", "rules"})
    if err != nil {
        t.Fatalf("parseKnowledgeBaseArg() error = %v", err)
    }
    if got != search.KnowledgeBaseRules {
        t.Fatalf("got %q", got)
    }
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./cmd/build_hybrid_index -run TestParseKnowledgeBaseArg`
Expected: FAIL because command does not exist yet.

- [ ] **Step 3: Implement the command**

```go
func main()
func run(ctx context.Context, args []string) error
func parseKnowledgeBaseArg(args []string) (string, error)
```

Behavior:

- load env/config through existing bootstrap helpers where possible
- require hybrid embedding config
- open PostgreSQL using existing DSN/env pattern
- construct:
  - `PostgresHybridSearchStore`
  - `PostgresIndexMetadataStore`
  - `QwenEmbedder`
  - `Indexer`
- load chunk source for:
  - `rules`
  - `lore`
  - `all`
- print concise progress logs

- [ ] **Step 4: Run the test to verify it passes**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./cmd/build_hybrid_index`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/build_hybrid_index/main.go
git commit -m "update: hybrid rag build command finished"
```

---

### Task 6: Verify Against a Real Local Database

**Files:**
- No new files required unless defects are found

- [ ] **Step 1: Start local services**

Run: `docker compose up -d postgres redis`
Expected: both containers are healthy.

- [ ] **Step 2: Build the rules index**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go run ./cmd/build_hybrid_index --knowledge-base rules`
Expected: successful batch embedding and upsert logs, no fatal error.

- [ ] **Step 3: Build the lore index**

Run: `GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go run ./cmd/build_hybrid_index --knowledge-base lore`
Expected: successful batch embedding and upsert logs, no fatal error.

- [ ] **Step 4: Spot-check metadata**

Run: `docker compose exec postgres psql -U dnd -d dndbot -c "select knowledge_base, embedding_provider, embedding_model, embedding_dim from knowledge_index_metadata order by knowledge_base;"`
Expected: `rules` and `lore` rows present with the configured Qwen 8B model and dimension.

- [ ] **Step 5: Commit any fixes**

```bash
git commit -m "update: hybrid rag index pipeline verification finished"
```

---

### Task 7: Switch to Hybrid Only After Data Exists

**Files:**
- Modify: `.env`

- [ ] **Step 1: Confirm `knowledge_chunks` contains both knowledge bases**

Run: `docker compose exec postgres psql -U dnd -d dndbot -c "select knowledge_base, count(*) from knowledge_chunks group by knowledge_base order by knowledge_base;"`
Expected: both `rules` and `lore` counts are non-zero.

- [ ] **Step 2: Change search backend**

```env
SEARCH_BACKEND=hybrid
```

- [ ] **Step 3: Restart the application**

Run: `docker compose up -d --build app`
Expected: app starts without hybrid initialization errors.

- [ ] **Step 4: Commit**

```bash
git add .env
git commit -m "update: enable hybrid search backend"
```

---

## Verification Commands

Primary package-level verification:

```bash
GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/retrieval/search ./internal/bootstrap ./internal/app ./cmd/build_hybrid_index
```

Real build verification:

```bash
docker compose up -d postgres redis
GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go run ./cmd/build_hybrid_index --knowledge-base all
```

---

## Rollout Notes

- keep `SEARCH_BACKEND=lexical` until `knowledge_chunks` is populated for both `rules` and `lore`
- the first release uses one fixed embedding space:
  - provider: `qwen`
  - model: `Qwen/Qwen3-Embedding-8B`
  - dimension: `1024`
- if provider, model, or dimension changes later, rebuild the index before switching production traffic
