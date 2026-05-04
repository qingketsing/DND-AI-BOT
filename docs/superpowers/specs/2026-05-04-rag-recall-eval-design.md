# RAG Recall Evaluation Design

## Goal

Build a repeatable retrieval evaluation pipeline for the current knowledge bases so the project can compare `lexical` and `hybrid` retrieval quality with stable `Recall@K` metrics.

The first release targets a fixed 50-query benchmark set spanning both `rules` and `lore`.

## Current State

The retrieval stack already supports two runtime backends:

- `lexical`
- `hybrid`

Runtime code already exposes a stable search abstraction through the retrieval package and bootstrap wiring. The project also already contains:

- offline chunk files under `data/chunks/...`
- lexical and hybrid search implementations under `internal/retrieval/search`
- a completed soak-eval pipeline under `cmd/soak_eval` and `internal/eval/soak`

What is still missing is a retrieval-specific benchmark with a stable goldset. Without that benchmark, the team cannot answer any of the following with evidence:

- whether `hybrid` actually outperforms `lexical`
- which query classes benefit from semantic retrieval
- whether later changes to chunking, embeddings, fusion, or query handling improve or degrade retrieval

## Scope

This spec covers:

- query dataset definition for a fixed 50-query benchmark
- LLM-assisted prelabeling of relevant chunk candidates
- human-reviewed goldset generation
- side-by-side lexical versus hybrid evaluation
- `Recall@1/3/5/10`
- `MRR`
- JSON, Markdown, and CSV report output

This spec does not cover:

- reranker evaluation
- answer-generation evaluation
- automated query rewriting experiments
- UI for annotation review
- external Python evaluation frameworks

## Product Requirements

### Evaluation Stability

The benchmark must be deterministic enough to compare runs over time.

That means:

- the final metric computation must not depend on live LLM judgment
- only approved goldset entries may participate in final metric runs
- lexical and hybrid must be evaluated against the same query set and same goldset

### Human Review Requirement

LLM may help reduce manual effort, but it must not be the final judge of relevance.

The required workflow is:

1. generate a draft relevance set with LLM assistance
2. review and correct that draft manually
3. promote reviewed entries into the approved goldset
4. run final evaluation from the approved goldset only

### Retrieval-Focused Metrics

This project needs retrieval metrics, not answer metrics.

The benchmark therefore measures:

- whether relevant chunks appear in the retrieved top-K
- where the first relevant chunk appears

The benchmark does not measure:

- whether the final agent answer is good
- whether the DM roleplay is good
- whether the retrieved text can be summarized well

## Dataset Design

### Query Count

The first release uses exactly 50 queries:

- `rules`: 25
- `lore`: 25

### Query Class Distribution

Each knowledge base should include a mixture of query types:

- `exact_name`
- `semantic`
- `alias`
- `multi_chunk`

Recommended distribution per knowledge base:

- `exact_name`: 8
- `semantic`: 8
- `alias`: 5
- `multi_chunk`: 4

This structure allows the project to compare:

- keyword-heavy retrieval behavior
- semantic paraphrase retrieval behavior
- alias handling quality
- cases where more than one chunk is relevant

### Query File Format

Two JSONL files define the benchmark queries:

- `configs/eval/rag_queries_rules.jsonl`
- `configs/eval/rag_queries_lore.jsonl`

Each line has this shape:

```json
{
  "id": "rules-stealth-001",
  "knowledge_base": "rules",
  "query": "潜行检定失败时会发生什么",
  "query_type": "semantic",
  "difficulty": "medium"
}
```

Required fields:

- `id`
- `knowledge_base`
- `query`
- `query_type`

Optional but recommended fields:

- `difficulty`
- `notes`

## Goldset Design

### Draft Goldset

LLM-assisted prelabeling writes a draft file:

- `configs/eval/rag_goldset_draft.jsonl`

Each line has this shape:

```json
{
  "query_id": "rules-stealth-001",
  "knowledge_base": "rules",
  "candidate_chunk_ids": ["rules-102", "rules-104", "rules-201"],
  "predicted_relevant_chunk_ids": ["rules-102", "rules-104"],
  "review_status": "draft",
  "notes": "LLM selected the main rule chunk and one supporting chunk"
}
```

### Approved Goldset

Human-reviewed entries are stored in:

- `configs/eval/rag_goldset.jsonl`

Each line has this shape:

```json
{
  "query_id": "rules-stealth-001",
  "knowledge_base": "rules",
  "relevant_chunk_ids": ["rules-102", "rules-104"],
  "review_status": "approved",
  "notes": "Reviewed manually"
}
```

Only entries with `review_status = "approved"` may be used in final evaluation.

### Versioning Constraint

The goldset is tied to chunk identities. If the chunking pipeline changes in a way that changes chunk IDs or major chunk boundaries, the goldset must be reviewed or rebuilt.

This is acceptable because the benchmark is intended to evaluate the actual current retrieval corpus, not an abstract external dataset.

## Prelabeling Design

### Why Prelabeling Exists

Manually finding relevant chunks from the full corpus for 50 queries is slow. LLM prelabeling reduces human review cost by narrowing each query to a small candidate pool.

### Candidate Pool Construction

For each query:

1. run `lexical` top 10
2. run `hybrid` top 10
3. merge and deduplicate results by chunk ID
4. keep the merged set as the candidate pool

This design intentionally avoids showing the full corpus to the LLM.

### LLM Input

The prelabeler sends the following to the evaluation model:

- the query text
- the knowledge base
- the candidate chunks
  - `chunk_id`
  - `title`
  - `content` truncated to a bounded size

### LLM Output

The model returns structured JSON:

```json
{
  "relevant_chunk_ids": ["rules-102", "rules-104"],
  "reason": "102 contains the direct rule and 104 contains the consequence detail"
}
```

The output is advisory only. It becomes the starting point for human review.

### Failure Behavior

If the prelabel LLM fails or returns invalid JSON:

- that query should still produce a draft entry
- `predicted_relevant_chunk_ids` should be empty
- `notes` should record the prelabel failure

The draft generation command must not abort the entire dataset build for one failed query.

## Human Review Workflow

The first release uses a file-based review workflow, not a UI.

Workflow:

1. run the draft generator
2. open `configs/eval/rag_goldset_draft.jsonl`
3. review each entry manually
4. adjust the relevant chunk IDs if needed
5. mark the entry as `approved`
6. copy approved entries into `configs/eval/rag_goldset.jsonl`

The first release does not need an interactive review terminal UI. JSONL plus editor review is enough.

## Metric Design

### Recall@K

For one query:

- let `G` be the set of approved relevant chunk IDs
- let `R@K` be the set of retrieved chunk IDs in top K

Then:

```text
Recall@K = |G ∩ R@K| / |G|
```

The benchmark must compute:

- `Recall@1`
- `Recall@3`
- `Recall@5`
- `Recall@10`

### MRR

For one query:

- find the rank of the first relevant chunk in the returned list
- if none is present, the reciprocal rank is `0`

Then:

```text
MRR contribution = 1 / first_relevant_rank
```

### Aggregation

The benchmark should compute macro averages across:

- all queries
- `rules` only
- `lore` only
- each `query_type`

This allows the project to inspect both total performance and class-specific behavior.

## CLI Design

### Command

The benchmark should use one CLI:

- `cmd/rag_eval`

### Modes

The command must support at least two modes:

- `prelabel`
- `eval`

### Prelabel Mode

Example:

```sh
GOCACHE=/tmp/go-build go run ./cmd/rag_eval \
  --mode prelabel \
  --output configs/eval/rag_goldset_draft.jsonl
```

Behavior:

- load query files
- build lexical and hybrid searchers
- generate candidate pools
- call the prelabel model
- write draft JSONL

### Eval Mode

Example:

```sh
GOCACHE=/tmp/go-build go run ./cmd/rag_eval \
  --mode eval \
  --goldset configs/eval/rag_goldset.jsonl \
  --output reports/eval/rag_eval_report.json
```

Behavior:

- load approved goldset
- run lexical retrieval for every query
- run hybrid retrieval for every query
- compute metrics
- write JSON, Markdown, and CSV reports

## Runtime Architecture

### Why Go Is the Right Stack

This evaluation pipeline should be implemented in Go rather than a separate Python evaluation stack because:

- the retrieval system already lives in Go
- the project already has a Go evaluation pattern in `soak_eval`
- the metrics are simple and deterministic
- cross-language glue would add operational cost without adding real value

### Reuse Strategy

The evaluation should reuse:

- `internal/retrieval/search`
- `internal/bootstrap/search_runtime.go`
- existing model adapter patterns from `internal/eval/soak`

This avoids duplicating retrieval logic and ensures the benchmark measures the same searchers used by the runtime.

## File Structure

Suggested files:

- `cmd/rag_eval/main.go`
- `internal/eval/rag/types.go`
- `internal/eval/rag/query_loader.go`
- `internal/eval/rag/prelabeler.go`
- `internal/eval/rag/evaluator.go`
- `internal/eval/rag/metrics.go`
- `internal/eval/rag/report.go`
- `internal/eval/rag/llm.go`
- `docs/eval/rag-eval.md`

The first release does not need a separate interactive reviewer package.

## Reporting

### JSON Report

Primary machine-readable output:

- `reports/eval/rag_eval_report.json`

This report should include:

- query count
- backend metrics
- grouped metrics
- per-query records

### Markdown Report

Human-readable summary:

- `reports/eval/rag_eval_report.md`

It should include:

- overall lexical versus hybrid table
- grouped metrics by knowledge base
- grouped metrics by query type
- selected failure examples

### CSV Report

Flat record export:

- `reports/eval/rag_eval_records.csv`

Each row should represent one `query x backend` evaluation record.

## Operational Constraints

### Environment

Prelabel mode needs:

- database access for hybrid retrieval
- embedding configuration for hybrid search
- prelabel model credentials

Eval mode needs:

- database access for hybrid retrieval
- embedding configuration for hybrid search

The final metric run does not need the prelabel model if the approved goldset already exists.

### Cost

Only the draft prelabel workflow consumes extra LLM tokens. Final metric computation must be deterministic and model-free except for normal hybrid query embedding, which is already part of the retrieval backend itself.

## Success Criteria

This project is successful when:

- a fixed 50-query dataset exists for rules and lore
- the project can generate draft relevance sets with LLM assistance
- humans can review and approve a goldset
- the project can compute lexical versus hybrid `Recall@1/3/5/10`
- the project can compute `MRR`
- the project can emit stable JSON and Markdown reports
- future retrieval changes can be evaluated against the same benchmark

## Recommended Rollout

### Phase 1

- define query and goldset file formats
- implement loaders and metric calculators

### Phase 2

- implement draft prelabel generation
- generate the first draft dataset

### Phase 3

- manually review and approve the goldset
- implement lexical versus hybrid evaluation

### Phase 4

- publish evaluation reports
- use the benchmark as a regression tool for future retrieval changes

