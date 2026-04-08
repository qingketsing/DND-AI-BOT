package search

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"strings"
)

const (
	defaultTopK            = 5
	titleMatchWeight       = 10.0
	aliasMatchWeight       = 8.0
	tagMatchWeight         = 6.0
	contentMatchWeight     = 2.0
	tokenTitleWeight       = 3.0
	tokenAliasWeight       = 2.5
	tokenTagWeight         = 2.0
	tokenContentWeight     = 0.5
)

var tokenizerPattern = regexp.MustCompile(`[\p{L}\p{N}]+`)

type LexicalSearcher struct {
	knowledgeBase string
	chunks        []SearchChunk
}

func NewLexicalSearcher(knowledgeBase string, chunks []SearchChunk) *LexicalSearcher {
	return &LexicalSearcher{
		knowledgeBase: knowledgeBase,
		chunks:        append([]SearchChunk(nil), chunks...),
	}
}

func NewLexicalSearcherFromJSONL(path string, knowledgeBase string) (*LexicalSearcher, error) {
	chunks, err := LoadChunksFromJSONL(path)
	if err != nil {
		return nil, err
	}
	return NewLexicalSearcher(knowledgeBase, chunks), nil
}

func NewRuleSearcher(path string) (Searcher, error) {
	return NewLexicalSearcherFromJSONL(path, KnowledgeBaseRules)
}

func NewLoreSearcher(path string) (Searcher, error) {
	return NewLexicalSearcherFromJSONL(path, KnowledgeBaseLore)
}

func LoadChunksFromJSONL(path string) ([]SearchChunk, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var chunks []SearchChunk
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var chunk SearchChunk
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return chunks, nil
}

func (s *LexicalSearcher) Search(ctx context.Context, request SearchRequest) ([]SearchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	request = normalizeSearchRequest(request)
	if err := validateSearchRequest(request); err != nil {
		return nil, err
	}

	query := normalizeText(request.Query)
	queryTokens := tokenizeQuery(request.Query)

	results := make([]SearchResult, 0, request.TopK)
	for _, chunk := range s.chunks {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if s.knowledgeBase != "" && chunk.KnowledgeBase != s.knowledgeBase {
			continue
		}
		score := ScoreChunk(query, queryTokens, chunk)
		if score <= 0 {
			continue
		}
		results = append(results, SearchResult{
			ChunkID:       chunk.ChunkID,
			DocumentID:    chunk.DocumentID,
			KnowledgeBase: chunk.KnowledgeBase,
			SourceType:    chunk.SourceType,
			DocType:       chunk.DocType,
			Title:         chunk.Title,
			Content:       chunk.Content,
			SectionPath:   append([]string(nil), chunk.SectionPath...),
			Tags:          append([]string(nil), chunk.Tags...),
			Aliases:       append([]string(nil), chunk.Aliases...),
			Position:      chunk.Position,
			ChunkStrategy: chunk.ChunkStrategy,
			Score:         score,
		})
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			if results[i].Title == results[j].Title {
				return results[i].ChunkID < results[j].ChunkID
			}
			return results[i].Title < results[j].Title
		}
		return results[i].Score > results[j].Score
	})

	if len(results) > request.TopK {
		results = results[:request.TopK]
	}
	return results, nil
}

func ScoreChunk(query string, queryTokens []string, chunk SearchChunk) float64 {
	score := 0.0

	score += scorePhraseMatch(query, chunk.Title, titleMatchWeight)
	score += scoreSlicePhraseMatch(query, chunk.Aliases, aliasMatchWeight)
	score += scoreSlicePhraseMatch(query, chunk.Tags, tagMatchWeight)
	score += scorePhraseMatch(query, chunk.Content, contentMatchWeight)

	score += scoreTokenMatches(queryTokens, chunk.Title, tokenTitleWeight)
	score += scoreSliceTokenMatches(queryTokens, chunk.Aliases, tokenAliasWeight)
	score += scoreSliceTokenMatches(queryTokens, chunk.Tags, tokenTagWeight)
	score += scoreTokenMatches(queryTokens, chunk.Content, tokenContentWeight)

	return score
}

func scorePhraseMatch(query string, text string, weight float64) float64 {
	if query == "" {
		return 0
	}
	if strings.Contains(normalizeText(text), query) {
		return weight
	}
	return 0
}

func scoreSlicePhraseMatch(query string, values []string, weight float64) float64 {
	score := 0.0
	for _, value := range values {
		score += scorePhraseMatch(query, value, weight)
	}
	return score
}

func scoreTokenMatches(tokens []string, text string, weight float64) float64 {
	if len(tokens) == 0 {
		return 0
	}
	normalized := normalizeText(text)
	score := 0.0
	for _, token := range tokens {
		if strings.Contains(normalized, token) {
			score += weight
		}
	}
	return score
}

func scoreSliceTokenMatches(tokens []string, values []string, weight float64) float64 {
	score := 0.0
	for _, value := range values {
		score += scoreTokenMatches(tokens, value, weight)
	}
	return score
}

func tokenizeQuery(query string) []string {
	normalized := normalizeText(query)
	if normalized == "" {
		return nil
	}
	matches := tokenizerPattern.FindAllString(normalized, -1)
	seen := make(map[string]struct{}, len(matches))
	tokens := make([]string, 0, len(matches))
	for _, match := range matches {
		if _, ok := seen[match]; ok {
			continue
		}
		seen[match] = struct{}{}
		tokens = append(tokens, match)
	}
	return tokens
}

func normalizeText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.Join(strings.Fields(value), " ")
	return value
}

func normalizeSearchRequest(request SearchRequest) SearchRequest {
	request.Query = strings.TrimSpace(request.Query)
	if request.TopK <= 0 {
		request.TopK = defaultTopK
	}
	return request
}

func validateSearchRequest(request SearchRequest) error {
	if request.Query == "" {
		return ErrInvalidSearchRequest
	}
	if request.TopK <= 0 {
		return ErrInvalidSearchRequest
	}
	return nil
}
