package rag

import "strings"

func buildDraftEntry(query Query, candidates []CandidateChunk, result PrelabelResult) DraftGoldsetEntry {
	entry := DraftGoldsetEntry{
		QueryID:                   query.ID,
		KnowledgeBase:             query.KnowledgeBase,
		CandidateChunkIDs:         make([]string, 0, len(candidates)),
		PredictedRelevantChunkIDs: append([]string(nil), result.RelevantChunkIDs...),
		ReviewStatus:              "draft",
		Notes:                     strings.TrimSpace(result.Reason),
	}
	for _, candidate := range candidates {
		entry.CandidateChunkIDs = append(entry.CandidateChunkIDs, candidate.ChunkID)
	}
	return entry
}
