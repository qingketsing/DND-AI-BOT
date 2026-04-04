package state

import "errors"

// QuestStatus 表示任务当前进度状态。
type QuestStatus string

const (
	QuestStatusActive    QuestStatus = "active"
	QuestStatusCompleted QuestStatus = "completed"
	QuestStatusFailed    QuestStatus = "failed"
)

// ItemEffectType 表示物品效果的最小分类。
type ItemEffectType string

const (
	ItemEffectHeal    ItemEffectType = "heal"
	ItemEffectBuff    ItemEffectType = "buff"
	ItemEffectDamage  ItemEffectType = "damage"
	ItemEffectQuest   ItemEffectType = "quest"
	ItemEffectUtility ItemEffectType = "utility"
)

var (
	// ErrItemNotFound 表示未找到对应的物品定义。
	ErrItemNotFound = errors.New("item not found")
)
