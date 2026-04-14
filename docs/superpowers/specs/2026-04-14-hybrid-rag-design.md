# Hybrid RAG Design

## Goal

Upgrade the current retrieval stack from JSONL-backed lexical search to a production-oriented hybrid RAG architecture that combines:

- PostgreSQL full-text search (`FTS`)
- pgvector semantic search
- `RRF` result fusion
- Qwen 8B embedding API for vector generation

The design must preserve the existing `Searcher` abstraction so the current agent tools and warmup layer can migrate without a large upstream rewrite.

## Current State

The current retrieval path is:

- chunk files stored as JSONL under `data/chunks/...`
- `LexicalSearcher` in [lexical.go](/home/qingke/DND-AI-BOT/internal/retrieval/search/lexical.go)
- `search_rules` and `search_lore` tools depend on the `Searcher` interface
- `KnowledgeWarmupService` also depends on the same `Searcher`

This works for exact term lookup, but it has three structural limits:

- poor semantic recall when the user does not use canonical rule or lore terms
- no persistent retrieval index inside the primary data store
- no controlled fusion between lexical matches and semantic matches

## Scope

This spec covers:

- database schema for indexed knowledge chunks
- embedding abstraction and first implementation against a fixed Qwen 8B embedding API
- hybrid search store and fusion strategy
- index build pipeline from existing JSONL chunks into PostgreSQL
- runtime integration behind the existing `Searcher` interface
- rollout and fallback strategy

This spec does not cover:

- rerankers
- query rewriting
- document authoring backoffice
- online chunk editing UI
- Elasticsearch / OpenSearch adoption
- multi-tenant retrieval isolation

## Product Requirements

### Retrieval Quality

The new retrieval stack must support both:

- precise lexical lookups for rule names, spell names, world terms, aliases, and titles
- semantic recall for natural-language paraphrases and incomplete user phrasing

### Compatibility

The new stack must remain compatible with:

- `search_rules`
- `search_lore`
- `KnowledgeWarmupService`
- any current caller that only depends on:

```go
type Searcher interface {
	Search(ctx context.Context, request SearchRequest) ([]SearchResult, error)
}
```

### Migration Safety

The system must support a safe rollout path:

- keep the current lexical implementation available
- allow backend selection through configuration
- support fallback to lexical search when hybrid dependencies are unavailable

### Cost and Operations

Embedding generation must be treated as an explicit dependency with cost and latency implications.

The design must distinguish:

- offline chunk embedding and index build
- online query embedding during retrieval

### Embedding Consistency

The first production release must use the same embedding provider, the same embedding model, and the same embedding dimension for:

- offline chunk indexing
- online query embedding

The system must not support mixing:

- one model for index build and another for online queries
- one vector dimension for stored chunks and another for query embedding

If the embedding model changes later, the index must be rebuilt before production traffic is switched.

## Architecture

The target architecture introduces a PostgreSQL-backed knowledge index with both full-text and vector search capabilities. Existing JSONL chunk files remain the source input for the first migration stage, but runtime retrieval moves to database-backed search.

The retrieval flow becomes:

1. a query arrives through the existing `Searcher` interface
2. the query is embedded through the configured embedding provider
3. PostgreSQL runs:
   - one `FTS` recall query
   - one pgvector similarity recall query
4. both candidate lists are fused with `RRF`
5. the fused list is returned as `[]SearchResult`

This keeps upstream code stable while replacing the retrieval core.

## Data Model

### Indexed Chunk

New runtime model:

```go
type IndexedChunk struct {
	ID            string
	KnowledgeBase string
	SourceID      string
	Title         string
	Aliases       []string
	Content       string
	Metadata      map[string]any
	Embedding     []float32
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
```

The model is intentionally narrower than `SearchChunk` and reflects what the database index must store.

### Database Table

New table:

```sql
create extension if not exists vector;

create table knowledge_chunks (
  id text primary key,
  knowledge_base text not null,
  source_id text not null,
  title text not null,
  aliases text[] not null default '{}',
  content text not null,
  metadata jsonb not null default '{}'::jsonb,
  content_tsv tsvector not null,
  embedding vector(1024),
  created_at timestamptz not null,
  updated_at timestamptz not null
);
```

Supporting indexes:

```sql
create index idx_knowledge_chunks_kb on knowledge_chunks (knowledge_base);
create index idx_knowledge_chunks_tsv on knowledge_chunks using gin (content_tsv);
create index idx_knowledge_chunks_embedding_ivfflat
on knowledge_chunks
using ivfflat (embedding vector_cosine_ops)
with (lists = 100);
```

Notes:

- `embedding vector(1024)` assumes the first release uses a Qwen 8B embedding model configured to output `1024` dimensions
- the schema dimension must match the configured embedding output dimension exactly
- the first release does not support mixing multiple embedding dimensions inside one index
- `content_tsv` should be built from weighted title, aliases, and content

## Embedding Design

### Why an Embedding Layer Exists

Hybrid retrieval needs semantic search. Semantic search requires turning text into vectors. PostgreSQL stores vectors and searches them, but it does not generate them.

### Abstraction

Embedding should be introduced as a small reusable interface:

```go
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}
```

Helper for single-query embedding:

```go
func EmbedQuery(ctx context.Context, embedder Embedder, text string) ([]float32, error)
```

### First Provider

First implementation should call a Qwen 8B embedding API.

This is not a chat model dependency. It is a dedicated embedding provider used to generate vectors for:

- chunk indexing
- query-time semantic recall

Provider config:

```go
type EmbeddingConfig struct {
	Provider  string
	Model     string
	Dim       int
	BatchSize int
	Timeout   time.Duration
	BaseURL   string
	APIKey    string
}
```

Initial deployment assumptions:

- provider: `qwen`
- model: configured externally, but fixed to one 8B embedding model for both indexing and online queries
- dimension: `1024`
- API key: configured through environment variables

The abstraction must not hard-code Qwen outside the first provider implementation, but the first release intentionally standardizes on one Qwen 8B model to avoid vector-space mismatch between offline indexing and online query retrieval.

## Retrieval Components

### Hybrid Search Store

The PostgreSQL query layer should be isolated behind a store interface:

```go
type HybridSearchStore interface {
	UpsertChunks(ctx context.Context, chunks []IndexedChunk) error
	SearchFTS(ctx context.Context, request SearchRequest) ([]ScoredCandidate, error)
	SearchVector(ctx context.Context, request VectorSearchRequest) ([]ScoredCandidate, error)
}
```

Supporting request and candidate types:

```go
type VectorSearchRequest struct {
	KnowledgeBase string
	Query         string
	QueryVector   []float32
	TopK          int
}

type ScoredCandidate struct {
	ID            string
	KnowledgeBase string
	Title         string
	Content       string
	Metadata      map[string]any
	FTSScore      float64
	VectorScore   float64
}
```

### Fusion

The first release should use reciprocal rank fusion:

```go
type FusionStrategy interface {
	Fuse(fts []ScoredCandidate, vector []ScoredCandidate, topK int) []SearchResult
}

type RRFFusion struct {
	K float64
}

func NewRRFFusion(k float64) *RRFFusion
func (f *RRFFusion) Fuse(fts []ScoredCandidate, vector []ScoredCandidate, topK int) []SearchResult
```

Why RRF:

- lexical scores and vector scores do not naturally share one scale
- RRF is stable, simple, and cheap to reason about
- it avoids premature tuning complexity

### Hybrid Searcher

The new runtime searcher keeps the existing interface:

```go
type HybridSearcher struct {
	store    HybridSearchStore
	embedder Embedder
	fusion   FusionStrategy
}

func NewHybridSearcher(
	store HybridSearchStore,
	embedder Embedder,
	fusion FusionStrategy,
) *HybridSearcher

func (s *HybridSearcher) Search(ctx context.Context, request SearchRequest) ([]SearchResult, error)
```

Search flow:

1. normalize and validate request
2. embed the query
3. execute `FTS` recall
4. execute vector recall
5. fuse results
6. return `[]SearchResult`

## Index Build Pipeline

The first release should keep existing chunk generation intact and add a separate index-build step.

### Input

- current JSONL chunks under `data/chunks/rules/chunks.jsonl`
- current JSONL chunks under `data/chunks/lore/chunks.jsonl`

### New Indexer

```go
type Indexer struct {
	store    HybridSearchStore
	embedder Embedder
}

func NewIndexer(store HybridSearchStore, embedder Embedder) *Indexer
func (i *Indexer) BuildIndex(ctx context.Context, knowledgeBase string, chunks []SearchChunk) error
```

### Embedding Text Policy

Chunk embedding text should not use raw content alone.

Recommended shape:

```text
title: <title>
aliases: <comma-joined aliases>
tags: <comma-joined tags>
content: <content>
```

This improves semantic retrieval for title-driven and alias-driven queries.

### Build Command

First release should provide a script or command such as:

```bash
go run ./scripts/rag/build_hybrid_index.go --knowledge-base rules
go run ./scripts/rag/build_hybrid_index.go --knowledge-base lore
```

or a combined command:

```bash
go run ./scripts/rag/build_hybrid_index.go --knowledge-base all
```

## Runtime Integration

### Search Runtime Bootstrap

`BuildSearchRuntime()` must be extended to support backend selection:

- `lexical`
- `hybrid`

Configuration example:

```text
SEARCH_BACKEND=hybrid
```

### Default Construction

Suggested behavior:

- if `SEARCH_BACKEND=lexical`, build current JSONL lexical searchers
- if `SEARCH_BACKEND=hybrid`, build PostgreSQL-backed `HybridSearcher`
- if hybrid initialization fails and fallback is enabled, log and fall back to lexical

### Compatibility

No caller changes should be required in:

- `internal/agent/tools/retrieval_tools.go`
- `internal/service/knowledge_warmup_service.go`
- `internal/bootstrap/agent_runtime.go`

## Error Handling and Fallback

### Query-Time Failure

If query embedding fails:

- return lexical fallback if enabled
- otherwise return a retrieval error

If vector search fails but `FTS` succeeds:

- return `FTS` results only

If `FTS` fails but vector search succeeds:

- return vector results only

If fusion input is empty:

- return an empty result set, not a panic

### Indexing Failure

If chunk embedding fails during index build:

- fail the batch
- emit actionable logs
- do not partially mark the batch as successful without explicit retry behavior

## Configuration

Recommended environment variables:

```text
SEARCH_BACKEND=hybrid
SEARCH_FALLBACK_ENABLED=true
HYBRID_FTS_TOPK=20
HYBRID_VECTOR_TOPK=20
HYBRID_FINAL_TOPK=8
HYBRID_RRF_K=60

EMBEDDING_PROVIDER=qwen
EMBEDDING_MODEL=<configured-8b-embedding-model>
EMBEDDING_API_KEY=<required>
EMBEDDING_BASE_URL=<required>
EMBEDDING_DIM=1024
EMBEDDING_BATCH_SIZE=32
EMBEDDING_TIMEOUT_SECONDS=30
```

`EMBEDDING_API_KEY` must be treated as a required secret in hybrid mode. If it is missing and lexical fallback is disabled, hybrid initialization must fail fast at startup.

## Testing Strategy

### Unit Tests

Cover:

- embedding helper behavior
- `RRF` fusion behavior
- `HybridSearcher` request flow and fallback behavior
- `buildEmbeddingText(...)` rules

### Store Tests

Cover:

- `FTS` retrieval returns expected rule/lore chunks
- vector retrieval returns expected nearest chunks
- `knowledge_base` filtering works
- upsert behavior is idempotent

### Bootstrap Tests

Cover:

- lexical backend still builds
- hybrid backend builds when config is present
- fallback behavior works when hybrid setup fails

## Rollout Plan

1. ship schema and index build command
2. populate PostgreSQL hybrid index from existing chunks
3. add hybrid backend behind config flag
4. run side-by-side retrieval evaluation
5. switch warmup and tool retrieval to hybrid in non-production
6. enable in production with lexical fallback still available

## Risks

### Embedding Cost

Query-time embedding introduces per-request cost and latency. This is acceptable for the first production architecture, but must be monitored.

### Dimension Lock-In

The database vector dimension must match the configured embedding output dimension. This should be explicit in migration and config.

### Model Lock-In Per Index Build

Each built knowledge index is intentionally bound to a single embedding model. If the project later migrates from one Qwen embedding model size to another, the chunk embeddings must be rebuilt before that new model is used for query embedding in production.

### Recall Drift

Hybrid retrieval will change retrieval behavior. This is expected. Evaluation must compare:

- precision for exact rules terms
- semantic recall for paraphrased queries
- lore retrieval stability for named locations and setting terms

## Success Criteria

The hybrid retrieval project is successful when:

- `search_rules` and `search_lore` continue to work through the same `Searcher` interface
- the system can retrieve exact rule/lore terms through `FTS`
- the system can retrieve semantically related chunks through vector recall
- fused results outperform the current lexical-only implementation on representative queries
- the backend can fall back safely if embedding or hybrid retrieval fails
