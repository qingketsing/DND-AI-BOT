package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"DND-AI-BOT/internal/eval/rag"
)

const (
	modePrelabel = "prelabel"
	modeEval     = "eval"
)

type cliOptions struct {
	Mode           string
	RulesQueryPath string
	LoreQueryPath  string
	GoldsetPath    string
	OutputPath     string
	CandidateTopK  int
	TopK           int
	KValues        []int
}

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		log.Fatalf("rag eval failed: %v", err)
	}
}

func run(ctx context.Context, args []string) error {
	options, err := parseArgs(args)
	if err != nil {
		return err
	}

	switch options.Mode {
	case modePrelabel:
		err = rag.RunPrelabel(ctx, rag.PrelabelRunConfig{
			RulesQueryPath: options.RulesQueryPath,
			LoreQueryPath:  options.LoreQueryPath,
			OutputPath:     options.OutputPath,
			CandidateTopK:  options.CandidateTopK,
			Model:          loadPrelabelModelConfigFromEnv(),
		})
		if err != nil {
			return err
		}
		fmt.Printf("rag prelabel completed: output=%s\n", options.OutputPath)
		return nil
	case modeEval:
		report, err := rag.RunEval(ctx, rag.EvalRunConfig{
			RulesQueryPath: options.RulesQueryPath,
			LoreQueryPath:  options.LoreQueryPath,
			GoldsetPath:    options.GoldsetPath,
			OutputPath:     options.OutputPath,
			KValues:        options.KValues,
			TopK:           options.TopK,
		})
		if err != nil {
			return err
		}
		fmt.Printf("rag eval completed: query_count=%d output=%s\n", report.QueryCount, options.OutputPath)
		return nil
	default:
		return fmt.Errorf("unsupported mode %q", options.Mode)
	}
}

func parseArgs(args []string) (cliOptions, error) {
	fs := flag.NewFlagSet("rag_eval", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	options := cliOptions{}
	var kValues string
	fs.StringVar(&options.Mode, "mode", modeEval, "rag eval mode: prelabel or eval")
	fs.StringVar(&options.RulesQueryPath, "rules-queries", filepath.Join("configs", "eval", "rag_queries_rules.jsonl"), "path to rules query JSONL")
	fs.StringVar(&options.LoreQueryPath, "lore-queries", filepath.Join("configs", "eval", "rag_queries_lore.jsonl"), "path to lore query JSONL")
	fs.StringVar(&options.GoldsetPath, "goldset", filepath.Join("configs", "eval", "rag_goldset.jsonl"), "path to approved goldset JSONL")
	fs.StringVar(&options.OutputPath, "output", "", "report or draft output path")
	fs.IntVar(&options.CandidateTopK, "candidate-topk", 10, "candidate pool size per backend for prelabel mode")
	fs.IntVar(&options.TopK, "topk", 10, "retrieval topK for final evaluation")
	fs.StringVar(&kValues, "k", "1,3,5,10", "comma-separated K values for Recall@K")
	if err := fs.Parse(args); err != nil {
		return cliOptions{}, err
	}

	options.Mode = strings.TrimSpace(options.Mode)
	if options.Mode != modePrelabel && options.Mode != modeEval {
		return cliOptions{}, fmt.Errorf("mode must be %q or %q", modePrelabel, modeEval)
	}
	if strings.TrimSpace(options.OutputPath) == "" {
		options.OutputPath = defaultOutputPath(options.Mode)
	}
	if options.Mode == modeEval && strings.TrimSpace(options.GoldsetPath) == "" {
		return cliOptions{}, fmt.Errorf("goldset path is required for eval mode")
	}

	parsedKValues, err := parseKValues(kValues)
	if err != nil {
		return cliOptions{}, err
	}
	options.KValues = parsedKValues
	return options, nil
}

func parseKValues(raw string) ([]int, error) {
	parts := strings.Split(raw, ",")
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, err := strconv.Atoi(part)
		if err != nil || value <= 0 {
			return nil, fmt.Errorf("invalid K value %q", part)
		}
		values = append(values, value)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("at least one K value is required")
	}
	return values, nil
}

func loadPrelabelModelConfigFromEnv() rag.ModelConfig {
	return rag.ModelConfig{
		Provider:       os.Getenv("RAG_EVAL_PRELABEL_PROVIDER"),
		Model:          os.Getenv("RAG_EVAL_PRELABEL_MODEL"),
		APIKey:         os.Getenv("RAG_EVAL_PRELABEL_API_KEY"),
		BaseURL:        os.Getenv("RAG_EVAL_PRELABEL_BASE_URL"),
		TimeoutSeconds: loadEnvInt("RAG_EVAL_PRELABEL_TIMEOUT_SECONDS", 60),
	}
}

func loadEnvInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func defaultOutputPath(mode string) string {
	switch mode {
	case modePrelabel:
		return filepath.Join("configs", "eval", "rag_goldset_draft.jsonl")
	default:
		return filepath.Join("reports", "eval", "rag_eval_report.json")
	}
}
