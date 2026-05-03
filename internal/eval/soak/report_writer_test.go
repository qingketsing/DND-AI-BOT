package soak

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteReportCreatesParentDirectoryAndWritesJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "report.json")
	report := sampleSoakReport()

	if err := WriteReport(path, report); err != nil {
		t.Fatalf("expected JSON report write to succeed, got %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected report file to exist, got %v", err)
	}
	if !strings.HasSuffix(string(raw), "\n") {
		t.Fatalf("expected JSON report to end with newline, got %q", string(raw))
	}

	var got SoakReport
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("expected valid JSON report, got %v", err)
	}
	if got.SessionID != report.SessionID {
		t.Fatalf("expected session id %q, got %q", report.SessionID, got.SessionID)
	}
	if got.FailureReasonCounts["lost_context"] != 1 {
		t.Fatalf("expected lost_context count 1, got %d", got.FailureReasonCounts["lost_context"])
	}
	if len(got.Records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(got.Records))
	}
}

func TestWriteMarkdownReportCreatesConciseSummary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "report.md")

	if err := WriteMarkdownReport(path, sampleSoakReport()); err != nil {
		t.Fatalf("expected Markdown report write to succeed, got %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected Markdown report file to exist, got %v", err)
	}
	body := string(raw)
	expectedSnippets := []string{
		"# Soak Eval Report",
		"- Session ID: city-session",
		"- Rounds: 2",
		"- Success: 1/2 (50.00%)",
		"- Average latency: 250 ms",
		"## Failure Reasons",
		"- lost_context: 1",
		"## Round Details",
		"| Round | Success | Score | Latency | Failure Reasons |",
		"| 2 | no | 0.40 | 400 ms | lost_context |",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(body, snippet) {
			t.Fatalf("expected Markdown report to contain %q, got:\n%s", snippet, body)
		}
	}
}

func sampleSoakReport() SoakReport {
	return SoakReport{
		SessionID:           "city-session",
		Rounds:              2,
		SuccessRounds:       1,
		SuccessRate:         0.5,
		AvgLatencyMS:        250,
		FailureReasonCounts: map[string]int{"lost_context": 1},
		Records: []RoundRecord{
			{
				Round:      1,
				UserInput:  "We enter the city.",
				AgentReply: "The gate opens.",
				LatencyMS:  100,
				HTTPStatus: 200,
				Success:    true,
				Score:      0.9,
			},
			{
				Round:          2,
				UserInput:      "Where is the gate?",
				AgentReply:     "A forest appears.",
				LatencyMS:      400,
				HTTPStatus:     200,
				Success:        false,
				Score:          0.4,
				FailureReasons: []string{"lost_context"},
				JudgeComment:   "Forgot the city gate.",
			},
		},
	}
}
