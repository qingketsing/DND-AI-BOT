# RAG Error Analysis

Date: 2026-05-04

Source reports:
- [reports/eval/rag_eval_report.json](/home/qingke/DND-AI-BOT/reports/eval/rag_eval_report.json)
- [reports/eval/rag_eval_report.md](/home/qingke/DND-AI-BOT/reports/eval/rag_eval_report.md)

## Scope

This analysis uses the current approved benchmark subset:

- total queries: 46
- rules: 21
- lore: 25

The goal is not to restate Recall@K, but to identify why `lexical` underperforms and what still limits `hybrid`.

## Summary

### Overall

| backend | MRR | Recall@1 | Recall@3 | Recall@5 | Recall@10 |
| --- | ---: | ---: | ---: | ---: | ---: |
| hybrid | 0.8826 | 0.6178 | 0.8098 | 0.9094 | 0.9783 |
| lexical | 0.3696 | 0.3116 | 0.3587 | 0.3587 | 0.3587 |

### By knowledge base

| KB | backend | MRR | Recall@1 | Recall@10 |
| --- | --- | ---: | ---: | ---: |
| rules | hybrid | 0.7429 | 0.4167 | 0.9524 |
| rules | lexical | 0.0476 | 0.0476 | 0.0476 |
| lore | hybrid | 1.0000 | 0.7867 | 1.0000 |
| lore | lexical | 0.6400 | 0.5333 | 0.6200 |

### By query type

| type | backend | Recall@1 | Recall@10 | MRR |
| --- | --- | ---: | ---: | ---: |
| exact_name | hybrid | 0.8500 | 1.0000 | 1.0000 |
| exact_name | lexical | 0.6000 | 0.6000 | 0.6000 |
| semantic | hybrid | 0.6377 | 1.0000 | 0.8667 |
| semantic | lexical | 0.2826 | 0.3478 | 0.3478 |
| alias | hybrid | 0.4048 | 0.8571 | 0.7381 |
| alias | lexical | 0.1429 | 0.1429 | 0.1429 |
| multi_chunk | hybrid | 0.4028 | 1.0000 | 0.9167 |
| multi_chunk | lexical | 0.1389 | 0.2500 | 0.3333 |

## Main findings

### 1. Lexical is not mainly a ranking problem. It is a recall failure.

The strongest signal is that `lexical` barely improves after top 3:

- Recall@3 = 0.3587
- Recall@5 = 0.3587
- Recall@10 = 0.3587

This means most failures are not "correct chunk exists but ranks low". The correct chunk is usually not in top 10 at all.

### 2. Lexical is effectively broken on the rules KB.

Observed breakdown:

- `lexical + rules`: 21 queries
  - empty result set: 20
  - hit: 1
  - non-empty miss: 0

- `lexical + lore`: 25 queries
  - empty result set: 8
  - hit: 16
  - non-empty miss: 1

Interpretation:

- On `rules`, the current lexical pipeline almost never returns anything useful.
- On `lore`, lexical still has partial value, especially for exact names and direct setting nouns.

This is not just "hybrid is better". It suggests the current lexical indexing or tokenization path is mismatched with Chinese natural-language rules queries.

### 3. Hybrid solves recall, but not always top-1 ranking.

Representative cases where `hybrid` hits but does not place the first relevant chunk at rank 1:

| query_id | query | rank of first hit |
| --- | --- | ---: |
| rules-001 | 创建角色时第一步要做什么 | 5 |
| rules-012 | 敏捷属性影响什么能力 | 5 |
| rules-016 | 魅力属性和哪些职业最相关 | 2 |
| rules-017 | 高等精灵适合法师吗 | 6 |
| rules-020 | 法师这个职业有什么特点 | 5 |
| rules-024 | 个性与背景章节是做什么的 | 3 |

Pattern:

- chapter-summary queries
- attribute-to-capability mapping queries
- "compatibility" or "适合/相关" phrasing

These are usually answered by a semantically correct chunk, but the top result is often a nearby context chunk, a sibling section, or a generic surrounding block.

### 4. Hybrid still misses one alias-heavy rules query.

Current `hybrid` rules miss:

- `rules-009`: `27 点购点法是什么`

Expected relevant chunk:

- `rules:phb:chapter:01-character-creation:0017`

Observed top 10 retrieval does not include the target chunk.

Interpretation:

- This is likely an alias / terminology normalization problem.
- The query uses a numeric colloquial form (`27 点购点法`), while the source chunk is likely phrased as a formal rule name or descriptive paragraph.

## Representative error cases

### Lexical miss, hybrid hit

These are the most useful examples for future debugging.

1. `rules-001` `创建角色时第一步要做什么`
   - lexical: no result
   - hybrid: hit at rank 5
   - issue class: chapter-step semantic query

2. `rules-012` `敏捷属性影响什么能力`
   - lexical: no result
   - hybrid: hit at rank 5
   - issue class: attribute semantics spread across multiple rules chunks

3. `rules-017` `高等精灵适合法师吗`
   - lexical: no result
   - hybrid: hit at rank 6
   - issue class: compatibility phrasing does not lexically match source wording

4. `lore-019` `滑门是做什么的`
   - lexical: retrieved unrelated `无光社会:0003`
   - hybrid: exact hit at rank 1 via `gates-to-the-city:0001`
   - issue class: alias / translated term mismatch

5. `lore-023` `大厅和走廊有什么共同特征`
   - lexical: no result
   - hybrid: both relevant chunks at top ranks
   - issue class: summary query over multiple setting fragments

## Likely root causes

### A. Chinese lexical retrieval is under-tokenized or weakly matched for rules queries

The `rules` failures are too extreme to explain by ordinary ranking noise. The current lexical backend likely has one or more of these problems:

- poor Chinese token segmentation for long PHB chunks
- low overlap between user phrasing and chunk wording
- weak title / heading matching
- no synonym or alias normalization

### B. Query wording is often semantic, not literal

Examples:

- `适合法师吗`
- `主要讲什么`
- `有什么区别`
- `和哪些职业最相关`

These questions ask for interpretation or summarization. Dense retrieval handles them much better than literal keyword matching.

### C. Numeric and colloquial aliases are not normalized

Example:

- `27 点购点法`

This likely needs a rewrite or synonym map to hit the formal chunk language reliably.

### D. Hybrid ranking still lacks a last-mile rerank signal

Hybrid already finds the right neighborhood. The remaining issue is that the best chunk is not always rank 1. This points to:

- insufficient title weighting
- no section-heading boost
- no lightweight rerank step

## Recommended next steps

### Priority 1. Inspect lexical indexing for the rules KB

Do not start with prompt tricks. First verify the retrieval substrate.

Check:

- how `rules` chunks are tokenized for FTS
- whether titles/headings are indexed separately
- whether Chinese queries are being normalized before lexical search
- whether the rules KB and lore KB share the same lexical strategy even though their query distributions differ

Expected outcome:

- confirm whether the rules lexical path is fundamentally broken or just poorly tuned

### Priority 2. Add a small query rewrite / alias normalization layer before retrieval

Especially for:

- numeric aliases: `27 点购点法`
- compatibility phrasing: `适合法师吗`
- chapter-summary phrasing: `主要讲什么`, `有什么区别`

This does not need to be a full LLM rewrite first. A deterministic rewrite table for high-frequency benchmark patterns is enough to validate the gain.

### Priority 3. Add title / heading boost or lightweight rerank for hybrid top 10

Target:

- improve `hybrid` top-1 precision
- keep current strong recall

The best first step is not a heavy reranker model. Start with:

- title boost
- heading boost
- query-type-aware weight tuning for `rules`

### Priority 4. Separate retrieval tuning for `rules` and `lore`

Current evidence says they behave very differently:

- `lore` still gives lexical some value
- `rules` does not

Treating them as the same retrieval problem is leaving performance on the table.

## Actionable follow-up tasks

1. Build a `lexical miss / hybrid hit` export command for top failed queries.
2. Add benchmark-time query rewrite hooks for a controlled A/B test.
3. Re-run Recall@K after:
   - lexical tuning
   - alias normalization
   - hybrid weighting adjustment
4. Keep the same goldset and compare only one retrieval variable at a time.

## Bottom line

The current benchmark already supports a strong engineering conclusion:

- `hybrid` is the correct default backend for this project.
- `lexical` still has limited value on `lore exact-name` queries.
- the next bottleneck is no longer recall; it is:
  - lexical rules indexing quality
  - alias normalization
  - hybrid top-1 ranking quality
