package model

import "time"

// SessionMemory 表示一个会话的长期摘要记忆。
type SessionMemory struct {
	SessionID        string    `json:"session_id"`
	CharacterSummary string    `json:"character_summary"`
	SceneSummary     string    `json:"scene_summary"`
	CurrentObjective string    `json:"current_objective"`
	RecentKeyEvents  []string  `json:"recent_key_events"`
	UpdatedAt        time.Time `json:"updated_at"`
}
