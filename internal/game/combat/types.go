package combat

// CombatSide 表示战斗单位的阵营。
type CombatSide string

const (
	CombatSideParty   CombatSide = "party"
	CombatSideEnemy   CombatSide = "enemy"
	CombatSideNeutral CombatSide = "neutral"
)

// CombatStatus 表示战斗单位当前的生命状态。
type CombatStatus string

const (
	CombatStatusActive CombatStatus = "active"
	CombatStatusDown   CombatStatus = "down"
	CombatStatusDead   CombatStatus = "dead"
)

// EffectType 表示战斗中可附着在单位身上的最小状态效果类型。
type EffectType string

const (
	EffectPoisoned      EffectType = "poisoned"
	EffectStunned       EffectType = "stunned"
	EffectProne         EffectType = "prone"
	EffectInvisible     EffectType = "invisible"
	EffectCharmed       EffectType = "charmed"
	EffectConcentrating EffectType = "concentrating"
)
