package search

import "sort"

const defaultRRFK = 60.0

// FusionStrategy 定义多路召回结果融合策略。
type FusionStrategy interface {
	Fuse(fts []ScoredCandidate, vector []ScoredCandidate, topK int) []SearchResult
}

// RRFFusion 使用 Reciprocal Rank Fusion 融合召回结果。
type RRFFusion struct {
	k float64
}

func NewRRFFusion(k float64) *RRFFusion {
	if k <= 0 {
		k = defaultRRFK
	}
	return &RRFFusion{k: k}
}

func (f *RRFFusion) Fuse(fts []ScoredCandidate, vector []ScoredCandidate, topK int) []SearchResult {
	if topK <= 0 {
		topK = defaultTopK
	}

	type accumulator struct {
		result SearchResult
		score  float64
	}

	combined := make(map[string]*accumulator, len(fts)+len(vector))
	merge := func(candidates []ScoredCandidate) {
		for rank, candidate := range candidates {
			acc, ok := combined[candidate.ChunkID]
			if !ok {
				acc = &accumulator{
					result: SearchResult{
						ChunkID:       candidate.ChunkID,
						DocumentID:    candidate.DocumentID,
						KnowledgeBase: candidate.KnowledgeBase,
						SourceType:    candidate.SourceType,
						DocType:       candidate.DocType,
						Title:         candidate.Title,
						Content:       candidate.Content,
						SectionPath:   append([]string(nil), candidate.SectionPath...),
						Tags:          append([]string(nil), candidate.Tags...),
						Aliases:       append([]string(nil), candidate.Aliases...),
						Position:      candidate.Position,
						ChunkStrategy: candidate.ChunkStrategy,
					},
				}
				combined[candidate.ChunkID] = acc
			}
			acc.score += 1.0 / (f.k + float64(rank+1))
		}
	}

	merge(fts)
	merge(vector)

	results := make([]SearchResult, 0, len(combined))
	for _, acc := range combined {
		acc.result.Score = acc.score
		results = append(results, acc.result)
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].ChunkID < results[j].ChunkID
		}
		return results[i].Score > results[j].Score
	})

	if len(results) > topK {
		results = results[:topK]
	}
	return results
}
