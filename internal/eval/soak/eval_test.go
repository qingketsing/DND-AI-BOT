package soak

import "testing"

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
