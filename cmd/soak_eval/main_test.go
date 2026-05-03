package main

import "testing"

func TestParseArgsUsesDefaultConfig(t *testing.T) {
	config, output, err := parseArgs(nil)
	if err != nil {
		t.Fatalf("expected parse args to succeed, got %v", err)
	}

	if config != "configs/eval/soak_the_city_50.json" {
		t.Fatalf("expected default config, got %q", config)
	}
	if output != "" {
		t.Fatalf("expected empty output, got %q", output)
	}
}

func TestParseArgsAcceptsConfigAndOutput(t *testing.T) {
	config, output, err := parseArgs([]string{"--config", "custom.json", "--output", "report.json"})
	if err != nil {
		t.Fatalf("expected parse args to succeed, got %v", err)
	}

	if config != "custom.json" {
		t.Fatalf("expected custom config, got %q", config)
	}
	if output != "report.json" {
		t.Fatalf("expected output report.json, got %q", output)
	}
}
