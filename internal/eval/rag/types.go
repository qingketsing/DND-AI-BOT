package rag

// ModelConfig defines one LLM used by rag evaluation helpers.
type ModelConfig struct {
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	APIKey         string `json:"api_key"`
	BaseURL        string `json:"base_url"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// Query defines one benchmark retrieval query.
type Query struct {
	ID            string `json:"id"`
	KnowledgeBase string `json:"knowledge_base"`
	Query         string `json:"query"`
	QueryType     string `json:"query_type"`
	Difficulty    string `json:"difficulty,omitempty"`
	Notes         string `json:"notes,omitempty"`
}

// DraftGoldsetEntry stores LLM-assisted draft relevance annotations.
type DraftGoldsetEntry struct {
	QueryID                  string   `json:"query_id"`
	KnowledgeBase            string   `json:"knowledge_base"`
	CandidateChunkIDs        []string `json:"candidate_chunk_ids"`
	PredictedRelevantChunkIDs []string `json:"predicted_relevant_chunk_ids"`
	ReviewStatus             string   `json:"review_status"`
	Notes                    string   `json:"notes,omitempty"`
}

// GoldsetEntry stores a human-approved relevance set for one query.
type GoldsetEntry struct {
	QueryID          string   `json:"query_id"`
	KnowledgeBase    string   `json:"knowledge_base"`
	RelevantChunkIDs []string `json:"relevant_chunk_ids"`
	ReviewStatus     string   `json:"review_status"`
	Notes            string   `json:"notes,omitempty"`
}

// QueryEvalRecord stores one query/backend retrieval evaluation result.
type QueryEvalRecord struct {
	QueryID           string             `json:"query_id"`
	KnowledgeBase     string             `json:"knowledge_base"`
	QueryType         string             `json:"query_type"`
	Backend           string             `json:"backend"`
	RetrievedChunkIDs []string           `json:"retrieved_chunk_ids,omitempty"`
	RelevantChunkIDs  []string           `json:"relevant_chunk_ids,omitempty"`
	RecallAtK         map[int]float64    `json:"recall_at_k"`
	MRR               float64            `json:"mrr"`
	FirstRelevantRank int                `json:"first_relevant_rank"`
}

// EvalReport is the top-level serialized evaluation report.
type EvalReport struct {
	QueryCount int           `json:"query_count"`
	Metrics    MetricsReport `json:"metrics"`
}

// MetricSummary stores averaged retrieval metrics for one group/backend.
type MetricSummary struct {
	QueryCount int             `json:"query_count"`
	RecallAtK  map[int]float64 `json:"recall_at_k"`
	MRR        float64         `json:"mrr"`
}

// MetricsReport stores overall and grouped averages.
type MetricsReport struct {
	Overall         map[string]MetricSummary            `json:"overall"`
	ByKnowledgeBase map[string]map[string]MetricSummary `json:"by_knowledge_base"`
	ByQueryType     map[string]map[string]MetricSummary `json:"by_query_type"`
	Records         []QueryEvalRecord                  `json:"records,omitempty"`
}

// CandidateChunk is the reduced candidate payload shown to the prelabel model.
type CandidateChunk struct {
	ChunkID string `json:"chunk_id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// PrelabelInput is the prompt context for one draft goldset prediction.
type PrelabelInput struct {
	Query      Query            `json:"query"`
	Candidates []CandidateChunk `json:"candidates"`
}

// PrelabelResult is the structured LLM output for one draft prediction.
type PrelabelResult struct {
	RelevantChunkIDs []string `json:"relevant_chunk_ids"`
	Reason           string   `json:"reason"`
}
