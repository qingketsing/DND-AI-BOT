package soak

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// WriteReport writes a JSON soak report, creating parent directories as needed.
func WriteReport(path string, report SoakReport) error {
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return writeReportFile(path, body)
}

// WriteMarkdownReport writes a concise Markdown soak report summary.
func WriteMarkdownReport(path string, report SoakReport) error {
	var builder strings.Builder
	fmt.Fprintln(&builder, "# Soak Eval Report")
	fmt.Fprintln(&builder)
	fmt.Fprintf(&builder, "- Session ID: %s\n", report.SessionID)
	fmt.Fprintf(&builder, "- Rounds: %d\n", report.Rounds)
	fmt.Fprintf(&builder, "- Success: %d/%d (%.2f%%)\n", report.SuccessRounds, report.Rounds, report.SuccessRate*100)
	fmt.Fprintf(&builder, "- Average latency: %d ms\n", report.AvgLatencyMS)
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, "## Failure Reasons")
	if len(report.FailureReasonCounts) == 0 {
		fmt.Fprintln(&builder, "- none")
	} else {
		for _, reason := range sortedFailureReasons(report.FailureReasonCounts) {
			fmt.Fprintf(&builder, "- %s: %d\n", reason, report.FailureReasonCounts[reason])
		}
	}
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, "## Round Details")
	fmt.Fprintln(&builder, "| Round | Success | Score | Latency | Failure Reasons |")
	fmt.Fprintln(&builder, "| --- | --- | --- | --- | --- |")
	for _, record := range report.Records {
		fmt.Fprintf(
			&builder,
			"| %d | %s | %.2f | %d ms | %s |\n",
			record.Round,
			markdownBool(record.Success),
			record.Score,
			record.LatencyMS,
			markdownFailureReasons(record.FailureReasons),
		)
	}

	return writeReportFile(path, []byte(builder.String()))
}

func writeReportFile(path string, body []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func sortedFailureReasons(counts map[string]int) []string {
	reasons := make([]string, 0, len(counts))
	for reason := range counts {
		reasons = append(reasons, reason)
	}
	sort.Strings(reasons)
	return reasons
}

func markdownBool(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func markdownFailureReasons(reasons []string) string {
	if len(reasons) == 0 {
		return "none"
	}
	escaped := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		escaped = append(escaped, strings.ReplaceAll(reason, "|", "\\|"))
	}
	return strings.Join(escaped, ", ")
}
