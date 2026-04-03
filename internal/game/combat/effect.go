package combat

// StatusEffect 表示附着在战斗单位上的最小状态效果。
type StatusEffect struct {
	ID       string
	Type     EffectType
	Source   string
	Duration int
}
