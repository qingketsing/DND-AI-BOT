# RAG Eval

RAG eval compares `lexical` and `hybrid` retrieval on a fixed query set and reports `Recall@K` plus `MRR`.

The first release uses:

- `configs/eval/rag_queries_rules.jsonl`
- `configs/eval/rag_queries_lore.jsonl`

These files are a starter benchmark set. They are not the final goldset. The intended workflow is:

1. run `prelabel` to generate `rag_goldset_draft.jsonl`
2. review the draft manually
3. write approved entries into `configs/eval/rag_goldset.jsonl`
4. run `eval`

## Environment

### Shared retrieval environment

The evaluator reuses the same retrieval stack as the main service, so `hybrid` mode requires:

```sh
export SEARCH_BACKEND=hybrid
export POSTGRES_DSN="<dsn>"
export EMBEDDING_PROVIDER="qwen"
export EMBEDDING_MODEL="Qwen3-Embedding-8B"
export EMBEDDING_API_KEY="<api-key>"
export EMBEDDING_BASE_URL="https://ai.opencumt.org/v1"
export EMBEDDING_DIM=1024
export EMBEDDING_BATCH_SIZE=16
export EMBEDDING_TIMEOUT_SECONDS=120
```

### Prelabel model environment

`prelabel` mode also needs a chat model for candidate relevance prediction:

```sh
export RAG_EVAL_PRELABEL_PROVIDER="deepseek"
export RAG_EVAL_PRELABEL_MODEL="deepseek-v4-flash"
export RAG_EVAL_PRELABEL_API_KEY="<api-key>"
export RAG_EVAL_PRELABEL_BASE_URL="https://api.deepseek.com"
export RAG_EVAL_PRELABEL_TIMEOUT_SECONDS=60
```

## Run prelabel

```sh
GOCACHE=/tmp/go-build go run ./cmd/rag_eval \
  --mode prelabel \
  --output configs/eval/rag_goldset_draft.jsonl
```

This command:

- loads rules and lore queries
- recalls lexical and hybrid candidates for each query
- merges and deduplicates candidate chunks
- asks the prelabel model to predict relevant chunk IDs
- writes JSONL draft entries

## Run final eval

After manual review produces `configs/eval/rag_goldset.jsonl`:

```sh
GOCACHE=/tmp/go-build go run ./cmd/rag_eval \
  --mode eval \
  --goldset configs/eval/rag_goldset.jsonl \
  --output reports/eval/rag_eval_report.json
```

This command writes:

```text
reports/eval/rag_eval_report.json
reports/eval/rag_eval_report.md
reports/eval/rag_eval_report.csv
```

## Metrics

The evaluator currently reports:

- `Recall@1`
- `Recall@3`
- `Recall@5`
- `Recall@10`
- `MRR`

Grouped summaries are produced for:

- all queries
- `rules` vs `lore`
- `query_type`

## Local verification

```sh
GOCACHE=/tmp/go-build go test ./cmd/rag_eval ./internal/eval/rag
```
