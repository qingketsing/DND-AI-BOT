package combat

import "errors"

var (
	// ErrCombatantNotFound 表示没有找到目标战斗单位。
	ErrCombatantNotFound = errors.New("combatant not found")
	// ErrInvalidDamage 表示伤害值不合法。
	ErrInvalidDamage = errors.New("invalid damage")
	// ErrInvalidHeal 表示治疗值不合法。
	ErrInvalidHeal = errors.New("invalid heal")
)
