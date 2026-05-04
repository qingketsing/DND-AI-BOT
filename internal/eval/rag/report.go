package rag

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func WriteJSONReport(path string, report EvalReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func WriteMarkdownReport(path string, report EvalReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var builder strings.Builder
	builder.WriteString("# RAG Eval Report\n\n")
	builder.WriteString("## Summary\n\n")
	builder.WriteString("- Query Count: " + strconv.Itoa(report.QueryCount) + "\n\n")
	builder.WriteString("## Overall\n\n")
	builder.WriteString("| Backend | Query Count | MRR |")
	ks := sortedKs(report.Metrics.Overall)
	for _, k := range ks {
		builder.WriteString(" Recall@" + strconv.Itoa(k) + " |")
	}
	builder.WriteString("\n| --- | --- | --- |")
	for range ks {
		builder.WriteString(" --- |")
	}
	builder.WriteString("\n")
	backendNames := sortedBackendNames(report.Metrics.Overall)
	for _, backend := range backendNames {
		summary := report.Metrics.Overall[backend]
		builder.WriteString("| " + backend + " | " + strconv.Itoa(summary.QueryCount) + " | " + formatFloat(summary.MRR) + " |")
		for _, k := range ks {
			builder.WriteString(" " + formatFloat(summary.RecallAtK[k]) + " |")
		}
		builder.WriteString("\n")
	}
	return os.WriteFile(path, []byte(builder.String()), 0o644)
}

func WriteCSVReport(path string, report EvalReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{
		"query_id", "knowledge_base", "query_type", "backend",
		"retrieved_chunk_ids", "relevant_chunk_ids", "first_relevant_rank", "mrr",
	}
	if err := writer.Write(header); err != nil {
		return err
	}
	for _, record := range report.Metrics.Records {
		row := []string{
			record.QueryID,
			record.KnowledgeBase,
			record.QueryType,
			record.Backend,
			strings.Join(record.RetrievedChunkIDs, "|"),
			strings.Join(record.RelevantChunkIDs, "|"),
			strconv.Itoa(record.FirstRelevantRank),
			formatFloat(record.MRR),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	return writer.Error()
}

func sortedKs(overall map[string]MetricSummary) []int {
	set := map[int]struct{}{}
	for _, summary := range overall {
		for k := range summary.RecallAtK {
			set[k] = struct{}{}
		}
	}
	ks := make([]int, 0, len(set))
	for k := range set {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	return ks
}

func sortedBackendNames(overall map[string]MetricSummary) []string {
	names := make([]string, 0, len(overall))
	for name := range overall {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
