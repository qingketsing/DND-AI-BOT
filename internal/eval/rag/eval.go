package rag

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"

	"DND-AI-BOT/internal/bootstrap"
)

type PrelabelRunConfig struct {
	RulesQueryPath string
	LoreQueryPath  string
	OutputPath     string
	CandidateTopK  int
	Model          ModelConfig
}

type EvalRunConfig struct {
	RulesQueryPath string
	LoreQueryPath  string
	GoldsetPath    string
	OutputPath     string
	KValues        []int
	TopK           int
}

func RunPrelabel(ctx context.Context, config PrelabelRunConfig) error {
	queries, err := LoadQueriesFromPaths(config.RulesQueryPath, config.LoreQueryPath)
	if err != nil {
		return err
	}
	db, err := bootstrap.OpenPostgresFromEnv()
	if err != nil {
		return err
	}
	defer db.Close()

	lexical, hybrid, err := buildEvalSearcherSets(db)
	if err != nil {
		return err
	}
	adapter, err := BuildModelAdapter(config.Model)
	if err != nil {
		return err
	}
	prelabeler := NewPrelabeler(adapter)

	entries := make([]DraftGoldsetEntry, 0, len(queries))
	for _, query := range queries {
		candidates, err := SearchAndMergeCandidates(ctx, query, lexical, hybrid, config.CandidateTopK)
		if err != nil {
			return err
		}
		entry, err := prelabeler.Predraft(ctx, PrelabelInput{
			Query:      query,
			Candidates: candidates,
		})
		if err != nil {
			return err
		}
		entries = append(entries, entry)
	}
	return WriteDraftGoldset(config.OutputPath, entries)
}

func RunEval(ctx context.Context, config EvalRunConfig) (EvalReport, error) {
	queries, err := LoadQueriesFromPaths(config.RulesQueryPath, config.LoreQueryPath)
	if err != nil {
		return EvalReport{}, err
	}
	goldset, err := LoadGoldset(config.GoldsetPath)
	if err != nil {
		return EvalReport{}, err
	}

	db, err := bootstrap.OpenPostgresFromEnv()
	if err != nil {
		return EvalReport{}, err
	}
	defer db.Close()

	lexical, hybrid, err := buildEvalSearcherSets(db)
	if err != nil {
		return EvalReport{}, err
	}
	report, err := NewEvaluator(lexical, hybrid, config.KValues, config.TopK).Evaluate(ctx, queries, goldset)
	if err != nil {
		return EvalReport{}, err
	}
	if strings.TrimSpace(config.OutputPath) != "" {
		if err := WriteJSONReport(config.OutputPath, report); err != nil {
			return EvalReport{}, err
		}
		if err := WriteMarkdownReport(markdownReportPath(config.OutputPath), report); err != nil {
			return EvalReport{}, err
		}
		if err := WriteCSVReport(csvReportPath(config.OutputPath), report); err != nil {
			return EvalReport{}, err
		}
	}
	return report, nil
}

func buildEvalSearcherSets(db *sql.DB) (SearcherSet, SearcherSet, error) {
	lexical, err := BuildLexicalSearchers()
	if err != nil {
		return SearcherSet{}, SearcherSet{}, err
	}
	hybrid, err := BuildHybridSearchers(db)
	if err != nil {
		return SearcherSet{}, SearcherSet{}, err
	}
	return lexical, hybrid, nil
}

func markdownReportPath(jsonPath string) string {
	extension := filepath.Ext(jsonPath)
	if extension == "" {
		return jsonPath + ".md"
	}
	return strings.TrimSuffix(jsonPath, extension) + ".md"
}

func csvReportPath(jsonPath string) string {
	extension := filepath.Ext(jsonPath)
	if extension == "" {
		return jsonPath + ".csv"
	}
	return strings.TrimSuffix(jsonPath, extension) + ".csv"
}
