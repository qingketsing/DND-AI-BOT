package search

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadChunksFromJSONL(t *testing.T) {
	t.Parallel()

	path := writeChunksFile(t, []SearchChunk{
		{
			ChunkID:       "rules:phb:class:wizard:0001",
			DocumentID:    "rules:phb:class:wizard",
			KnowledgeBase: KnowledgeBaseRules,
			SourceType:    "phb",
			DocType:       "class",
			Title:         "法师",
			Content:       "法师使用奥术施法。",
			Language:      "zh",
			SectionPath:   []string{"第 3 章：职业", "法师"},
			Tags:          []string{"class", "wizard"},
			Aliases:       []string{"法师", "Wizard"},
			Position:      1,
			ChunkStrategy: "whole_document",
		},
	})

	chunks, err := LoadChunksFromJSONL(path)
	if err != nil {
		t.Fatalf("LoadChunksFromJSONL() error = %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Title != "法师" {
		t.Fatalf("expected title 法师, got %s", chunks[0].Title)
	}
}

func TestLexicalSearcherPrefersTitleAndAliasMatches(t *testing.T) {
	t.Parallel()

	searcher := NewLexicalSearcher(KnowledgeBaseRules, []SearchChunk{
		{
			ChunkID:       "rules:phb:class:wizard:0001",
			DocumentID:    "rules:phb:class:wizard",
			KnowledgeBase: KnowledgeBaseRules,
			SourceType:    "phb",
			DocType:       "class",
			Title:         "法师",
			Content:       "法师使用奥术施法。",
			Language:      "zh",
			SectionPath:   []string{"第 3 章：职业", "法师"},
			Tags:          []string{"class", "wizard"},
			Aliases:       []string{"法师", "Wizard"},
			Position:      1,
			ChunkStrategy: "whole_document",
		},
		{
			ChunkID:       "rules:phb:chapter:10-spellcasting:0001",
			DocumentID:    "rules:phb:chapter:10-spellcasting",
			KnowledgeBase: KnowledgeBaseRules,
			SourceType:    "phb",
			DocType:       "chapter",
			Title:         "施法",
			Content:       "法师与牧师都会施法。",
			Language:      "zh",
			SectionPath:   []string{"第 10 章：施法"},
			Tags:          []string{"chapter", "spellcasting"},
			Aliases:       []string{"施法"},
			Position:      1,
			ChunkStrategy: "section_window",
		},
	})

	results, err := searcher.Search(context.Background(), SearchRequest{Query: "法师", TopK: 2})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ChunkID != "rules:phb:class:wizard:0001" {
		t.Fatalf("expected class chunk first, got %s", results[0].ChunkID)
	}
	if results[0].Score <= results[1].Score {
		t.Fatalf("expected first result score > second result score, got %f <= %f", results[0].Score, results[1].Score)
	}
}

func TestLexicalSearcherHonorsKnowledgeBaseFilter(t *testing.T) {
	t.Parallel()

	searcher := NewLexicalSearcher(KnowledgeBaseLore, []SearchChunk{
		{
			ChunkID:       "rules:phb:class:wizard:0001",
			DocumentID:    "rules:phb:class:wizard",
			KnowledgeBase: KnowledgeBaseRules,
			SourceType:    "phb",
			DocType:       "class",
			Title:         "法师",
			Content:       "法师使用奥术施法。",
			Language:      "zh",
			SectionPath:   []string{"第 3 章：职业", "法师"},
			Tags:          []string{"class", "wizard"},
			Aliases:       []string{"法师", "Wizard"},
			Position:      1,
			ChunkStrategy: "whole_document",
		},
		{
			ChunkID:       "lore:default-setting:frozen-sky:0001",
			DocumentID:    "lore:default-setting:frozen-sky",
			KnowledgeBase: KnowledgeBaseLore,
			SourceType:    "background_md",
			DocType:       "setting_section",
			Title:         "凝固的天空",
			Content:       "太阳静止地悬挂在城市上空。",
			Language:      "zh",
			SectionPath:   []string{"凝固的天空"},
			Tags:          []string{"setting_section", "凝固的天空"},
			Aliases:       []string{"凝固的天空"},
			Position:      1,
			ChunkStrategy: "paragraph_window",
		},
	})

	results, err := searcher.Search(context.Background(), SearchRequest{Query: "天空", TopK: 5})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 lore result, got %d", len(results))
	}
	if results[0].KnowledgeBase != KnowledgeBaseLore {
		t.Fatalf("expected lore result, got %s", results[0].KnowledgeBase)
	}
}

func TestNewDefaultSearchersLoadChunkFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	rulesPath := filepath.Join(root, "rules.jsonl")
	lorePath := filepath.Join(root, "lore.jsonl")
	writeChunksToPath(t, rulesPath, []SearchChunk{{
		ChunkID:       "rules:phb:class:wizard:0001",
		DocumentID:    "rules:phb:class:wizard",
		KnowledgeBase: KnowledgeBaseRules,
		SourceType:    "phb",
		DocType:       "class",
		Title:         "法师",
		Content:       "法师使用奥术施法。",
		Language:      "zh",
		Tags:          []string{"class"},
		Aliases:       []string{"法师"},
		Position:      1,
		ChunkStrategy: "whole_document",
	}})
	writeChunksToPath(t, lorePath, []SearchChunk{{
		ChunkID:       "lore:default-setting:dead-world:0001",
		DocumentID:    "lore:default-setting:dead-world",
		KnowledgeBase: KnowledgeBaseLore,
		SourceType:    "background_md",
		DocType:       "setting_section",
		Title:         "死寂的世界",
		Content:       "这里只有城市。",
		Language:      "zh",
		Tags:          []string{"setting_section"},
		Aliases:       []string{"死寂的世界"},
		Position:      1,
		ChunkStrategy: "paragraph_window",
	}})

	rulesSearcher, err := NewRuleSearcher(rulesPath)
	if err != nil {
		t.Fatalf("NewRuleSearcher() error = %v", err)
	}
	loreSearcher, err := NewLoreSearcher(lorePath)
	if err != nil {
		t.Fatalf("NewLoreSearcher() error = %v", err)
	}

	rulesResults, err := rulesSearcher.Search(context.Background(), SearchRequest{Query: "法师", TopK: 1})
	if err != nil {
		t.Fatalf("rules Search() error = %v", err)
	}
	loreResults, err := loreSearcher.Search(context.Background(), SearchRequest{Query: "城市", TopK: 1})
	if err != nil {
		t.Fatalf("lore Search() error = %v", err)
	}

	if len(rulesResults) != 1 || len(loreResults) != 1 {
		t.Fatalf("expected one result from each searcher, got %d and %d", len(rulesResults), len(loreResults))
	}
}

func writeChunksFile(t *testing.T, chunks []SearchChunk) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "chunks.jsonl")
	writeChunksToPath(t, path, chunks)
	return path
}

func writeChunksToPath(t *testing.T, path string, chunks []SearchChunk) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("os.Create() error = %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, chunk := range chunks {
		if err := encoder.Encode(chunk); err != nil {
			t.Fatalf("encoder.Encode() error = %v", err)
		}
	}
}
