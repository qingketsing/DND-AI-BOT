package rag

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var ErrInvalidEvalFile = errors.New("invalid eval file")

func LoadQueries(path string) ([]Query, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var queries []Query
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var query Query
		if err := json.Unmarshal([]byte(line), &query); err != nil {
			return nil, err
		}
		if strings.TrimSpace(query.ID) == "" || strings.TrimSpace(query.KnowledgeBase) == "" || strings.TrimSpace(query.Query) == "" || strings.TrimSpace(query.QueryType) == "" {
			return nil, ErrInvalidEvalFile
		}
		queries = append(queries, query)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return queries, nil
}

func LoadGoldset(path string) ([]GoldsetEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []GoldsetEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry GoldsetEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, err
		}
		if strings.TrimSpace(entry.QueryID) == "" || strings.TrimSpace(entry.KnowledgeBase) == "" {
			return nil, ErrInvalidEvalFile
		}
		if strings.TrimSpace(entry.ReviewStatus) != "approved" {
			continue
		}
		if len(entry.RelevantChunkIDs) == 0 {
			return nil, ErrInvalidEvalFile
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func LoadQueriesFromPaths(paths ...string) ([]Query, error) {
	queries := make([]Query, 0)
	for _, path := range paths {
		loaded, err := LoadQueries(path)
		if err != nil {
			return nil, err
		}
		queries = append(queries, loaded...)
	}
	return queries, nil
}

func WriteDraftGoldset(path string, entries []DraftGoldsetEntry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			return err
		}
	}
	return nil
}

func SearchAndMergeCandidates(ctx context.Context, query Query, lexical SearcherSet, hybrid SearcherSet, topK int) ([]CandidateChunk, error) {
	lexicalSearcher, err := lexical.ForKnowledgeBase(query.KnowledgeBase)
	if err != nil {
		return nil, err
	}
	hybridSearcher, err := hybrid.ForKnowledgeBase(query.KnowledgeBase)
	if err != nil {
		return nil, err
	}
	lexicalResults, err := lexicalSearcher.Search(ctx, searchRequestForQuery(query, topK))
	if err != nil {
		return nil, err
	}
	hybridResults, err := hybridSearcher.Search(ctx, searchRequestForQuery(query, topK))
	if err != nil {
		return nil, err
	}
	return MergeCandidatePools(lexicalResults, hybridResults), nil
}
