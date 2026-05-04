package main

import "testing"

func TestParseFlagsUsesDefaults(t *testing.T) {
	options, err := parseArgs(nil)
	if err != nil {
		t.Fatalf("expected parse args to succeed, got %v", err)
	}
	if options.Mode != modeEval {
		t.Fatalf("expected default mode eval, got %q", options.Mode)
	}
	if options.RulesQueryPath != "configs/eval/rag_queries_rules.jsonl" {
		t.Fatalf("unexpected default rules path: %q", options.RulesQueryPath)
	}
}

func TestParseFlagsAcceptsPrelabelMode(t *testing.T) {
	options, err := parseArgs([]string{"--mode", "prelabel", "--output", "draft.jsonl"})
	if err != nil {
		t.Fatalf("expected parse args to succeed, got %v", err)
	}
	if options.Mode != modePrelabel {
		t.Fatalf("expected prelabel mode, got %q", options.Mode)
	}
	if options.OutputPath != "draft.jsonl" {
		t.Fatalf("unexpected output path: %q", options.OutputPath)
	}
}

func TestEvalRequiresGoldset(t *testing.T) {
	_, err := parseArgs([]string{"--mode", "eval", "--goldset", ""})
	if err == nil {
		t.Fatal("expected missing goldset to fail")
	}
}
