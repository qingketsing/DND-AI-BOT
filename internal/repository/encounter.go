package repository

import (
	"context"

	"DND-AI-BOT/internal/game/combat"
)

// EncounterRepository 定义战斗聚合的统一存取接口。
type EncounterRepository interface {
	Save(ctx context.Context, encounter *combat.Encounter) error
	LoadBySessionID(ctx context.Context, sessionID string) (*combat.Encounter, error)
}
