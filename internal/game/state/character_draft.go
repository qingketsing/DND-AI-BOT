package state

// CharacterDraft 表示角色创建流程中的半成品状态。
type CharacterDraft struct {
	Name          string   `json:"name"`
	Race          string   `json:"race"`
	Class         string   `json:"class"`
	Level         int      `json:"level"`
	AbilityMethod string   `json:"ability_method"`
	PendingFields []string `json:"pending_fields"`
}
