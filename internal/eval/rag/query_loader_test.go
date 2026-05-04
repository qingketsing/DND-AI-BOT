package rag

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadQueriesLoadsJSONL(t *testing.T) {
	path := writeTempFile(t, "queries.jsonl", ""+
		"{\"id\":\"rules-1\",\"knowledge_base\":\"rules\",\"query\":\"潜行规则\",\"query_type\":\"exact_name\"}\n"+
		"{\"id\":\"lore-1\",\"knowledge_base\":\"lore\",\"query\":\"the city 的滑门\",\"query_type\":\"semantic\"}\n",
	)

	queries, err := LoadQueries(path)
	if err != nil {
		t.Fatalf("expected queries to load, got %v", err)
	}
	if len(queries) != 2 {
		t.Fatalf("expected 2 queries, got %d", len(queries))
	}
	if queries[0].ID != "rules-1" || queries[1].KnowledgeBase != "lore" {
		t.Fatalf("unexpected queries: %+v", queries)
	}
}

func TestLoadQueriesRejectsMalformedJSONL(t *testing.T) {
	path := writeTempFile(t, "queries.jsonl", "{not-json}\n")

	if _, err := LoadQueries(path); err == nil {
		t.Fatal("expected malformed JSONL to fail")
	}
}

func TestLoadGoldsetFiltersApprovedEntries(t *testing.T) {
	path := writeTempFile(t, "goldset.jsonl", ""+
		"{\"query_id\":\"rules-1\",\"knowledge_base\":\"rules\",\"relevant_chunk_ids\":[\"rules-101\"],\"review_status\":\"approved\"}\n"+
		"{\"query_id\":\"rules-2\",\"knowledge_base\":\"rules\",\"relevant_chunk_ids\":[\"rules-202\"],\"review_status\":\"draft\"}\n",
	)

	entries, err := LoadGoldset(path)
	if err != nil {
		t.Fatalf("expected goldset to load, got %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected only approved entries, got %d", len(entries))
	}
	if entries[0].QueryID != "rules-1" {
		t.Fatalf("unexpected goldset entry: %+v", entries[0])
	}
}

func TestLoadGoldsetRejectsMissingRelevantChunkIDs(t *testing.T) {
	path := writeTempFile(t, "goldset.jsonl", ""+
		"{\"query_id\":\"rules-1\",\"knowledge_base\":\"rules\",\"review_status\":\"approved\"}\n",
	)

	if _, err := LoadGoldset(path); err == nil {
		t.Fatal("expected missing relevant chunk ids to fail")
	}
}

func writeTempFile(t *testing.T, name string, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}
