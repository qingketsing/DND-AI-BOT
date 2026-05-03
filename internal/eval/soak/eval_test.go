package soak

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMarkdownReportPathReplacesExtension(t *testing.T) {
	path := markdownReportPath("reports/eval/soak.json")

	if path != "reports/eval/soak.md" {
		t.Fatalf("expected markdown path reports/eval/soak.md, got %q", path)
	}
}

func TestMarkdownReportPathAddsExtensionWhenMissing(t *testing.T) {
	path := markdownReportPath("reports/eval/soak")

	if path != "reports/eval/soak.md" {
		t.Fatalf("expected markdown path reports/eval/soak.md, got %q", path)
	}
}

func TestBuildCheckpointReporterWritesPartialReports(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "soak.json")
	reporter := BuildCheckpointReporter(outputPath)

	reporter(RoundRecord{Round: 1, Success: true}, BuildReport("session-1", []RoundRecord{{Round: 1, Success: true}}))

	assertFileExists(t, outputPath)
	assertFileExists(t, markdownReportPath(outputPath))
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %s to exist, got %v", path, err)
	}
}
