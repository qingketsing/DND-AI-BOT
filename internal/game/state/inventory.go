package state

// InventoryItem 表示背包中的一个物品条目。
type InventoryItem struct {
	ID       string
	ItemID   string
	Name     string
	Quantity int
}

// ItemEffect 表示物品的单个效果参数。
type ItemEffect struct {
	Type   ItemEffectType
	Target string
	Value  int
	Key    string
}

// ItemDefinition 表示物品模板及其效果定义。
type ItemDefinition struct {
	ItemID  string
	Name    string
	Effects []ItemEffect
}
