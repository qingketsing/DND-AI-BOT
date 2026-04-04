package state

// ItemDefinitionStore 定义物品效果查询接口。
type ItemDefinitionStore interface {
	GetItemDefinition(itemID string) (ItemDefinition, error)
	GetItemEffects(itemID string) ([]ItemEffect, error)
}

// InMemoryItemDefinitionStore 以内存方式保存物品模板定义。
type InMemoryItemDefinitionStore struct {
	items map[string]ItemDefinition
}

// NewInMemoryItemDefinitionStore 创建最小内存物品定义仓库。
func NewInMemoryItemDefinitionStore(items []ItemDefinition) *InMemoryItemDefinitionStore {
	store := &InMemoryItemDefinitionStore{
		items: make(map[string]ItemDefinition, len(items)),
	}
	for _, item := range items {
		store.items[item.ItemID] = item
	}

	return store
}

// GetItemDefinition 返回指定物品的完整定义。
func (s *InMemoryItemDefinitionStore) GetItemDefinition(itemID string) (ItemDefinition, error) {
	item, ok := s.items[itemID]
	if !ok {
		return ItemDefinition{}, ErrItemNotFound
	}

	return item, nil
}

// GetItemEffects 返回指定物品的效果列表。
func (s *InMemoryItemDefinitionStore) GetItemEffects(itemID string) ([]ItemEffect, error) {
	item, err := s.GetItemDefinition(itemID)
	if err != nil {
		return nil, err
	}

	effects := make([]ItemEffect, len(item.Effects))
	copy(effects, item.Effects)
	return effects, nil
}
