package soak

// ModelConfig defines a chat model used by soak evaluation.
type ModelConfig struct {
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	APIKey         string `json:"api_key"`
	BaseURL        string `json:"base_url"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// ScenarioConfig describes the long-session test objective.
type ScenarioConfig struct {
	Name       string   `json:"name"`
	Objective  string   `json:"objective"`
	SeedPrompt string   `json:"seed_prompt"`
	Milestones []string `json:"milestones"`
}

// SoakConfig is the top-level configuration for a long-session evaluation run.
type SoakConfig struct {
	BaseURL        string         `json:"base_url"`
	SessionID      string         `json:"session_id"`
	UserToken      string         `json:"user_token"`
	Rounds         int            `json:"rounds"`
	Scenario       ScenarioConfig `json:"scenario"`
	PlayerModel    ModelConfig    `json:"player_model"`
	JudgeModel     ModelConfig    `json:"judge_model"`
	TimeoutSeconds int            `json:"timeout_seconds"`
	OutputPath     string         `json:"output_path"`
}

// RoundRecord stores all observable data for one evaluation round.
type RoundRecord struct {
	Round          int      `json:"round"`
	UserInput      string   `json:"user_input"`
	AgentReply     string   `json:"agent_reply"`
	LatencyMS      int64    `json:"latency_ms"`
	HTTPStatus     int      `json:"http_status"`
	Success        bool     `json:"success"`
	Score          float64  `json:"score"`
	FailureReasons []string `json:"failure_reasons"`
	JudgeComment   string   `json:"judge_comment"`
}

// JudgeResult is the structured LLM-as-judge output.
type JudgeResult struct {
	Success        bool     `json:"success"`
	Score          float64  `json:"score"`
	FailureReasons []string `json:"failure_reasons"`
	Comment        string   `json:"comment"`
}

// SoakReport summarizes the whole run.
type SoakReport struct {
	SessionID           string         `json:"session_id"`
	Rounds              int            `json:"rounds"`
	SuccessRounds       int            `json:"success_rounds"`
	SuccessRate         float64        `json:"success_rate"`
	AvgLatencyMS        int64          `json:"avg_latency_ms"`
	FailureReasonCounts map[string]int `json:"failure_reason_counts"`
	Records             []RoundRecord  `json:"records"`
}
