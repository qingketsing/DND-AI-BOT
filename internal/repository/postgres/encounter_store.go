package postgres

import (
	"context"

	"DND-AI-BOT/internal/game/combat"
)

// EncounterStore 定义 PostgreSQL 战斗真相源接口。
type EncounterStore interface {
	UpsertEncounter(ctx context.Context, encounter *combat.Encounter) error
	GetEncounterBySessionID(ctx context.Context, sessionID string) (*combat.Encounter, error)
}
