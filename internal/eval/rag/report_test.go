package rag

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteJSONReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	report := EvalReport{
		QueryCount: 2,
		Metrics: MetricsReport{
			Overall: map[string]MetricSummary{
				"lexical": {QueryCount: 2, RecallAtK: map[int]float64{1: 0.5}, MRR: 0.75},
			},
		},
	}

	if err := WriteJSONReport(path, report); err != nil {
		t.Fatalf("expected JSON report to be written, got %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var decoded EvalReport
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	if decoded.QueryCount != 2 {
		t.Fatalf("expected query count 2, got %d", decoded.QueryCount)
	}
}

func TestWriteMarkdownReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.md")
	report := EvalReport{
		QueryCount: 1,
		Metrics: MetricsReport{
			Overall: map[string]MetricSummary{
				"hybrid": {QueryCount: 1, RecallAtK: map[int]float64{1: 1, 3: 1}, MRR: 1},
			},
		},
	}

	if err := WriteMarkdownReport(path, report); err != nil {
		t.Fatalf("expected markdown report to be written, got %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read markdown report: %v", err)
	}
	content := string(raw)
	for _, expected := range []string{"# RAG Eval Report", "Query Count", "hybrid"} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected markdown report to contain %q, got %q", expected, content)
		}
	}
}

func TestWriteCSVReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.csv")
	report := EvalReport{
		QueryCount: 1,
		Metrics: MetricsReport{
			Records: []QueryEvalRecord{
				{
					QueryID:           "rules-1",
					KnowledgeBase:     "rules",
					QueryType:         "semantic",
					Backend:           "lexical",
					RetrievedChunkIDs: []string{"a", "b"},
					RelevantChunkIDs:  []string{"b"},
					RecallAtK:         map[int]float64{1: 0, 3: 1},
					MRR:               0.5,
					FirstRelevantRank: 2,
				},
			},
		},
	}

	if err := WriteCSVReport(path, report); err != nil {
		t.Fatalf("expected csv report to be written, got %v", err)
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open csv report: %v", err)
	}
	defer file.Close()

	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 csv rows, got %d", len(rows))
	}
	if rows[1][0] != "rules-1" || rows[1][3] != "lexical" {
		t.Fatalf("unexpected csv row: %+v", rows[1])
	}
}
