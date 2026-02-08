package game

import (
	"fmt"
	"strings"
	"sync"
)

// Character 极简角色卡
type Character struct {
	Name  string `json:"name"`
	Class string `json:"class"` // 职业: 战士, 法师...
	HP    int    `json:"hp"`
	MaxHP int    `json:"max_hp"`
	STR   int    `json:"str"` // 力量
}

// GroupState 管理一个群内的游戏状态
type GroupState struct {
	GroupID    int64
	Characters map[string]*Character // Key: Character Name (lowercase)
	Mutex      sync.RWMutex
}

// StateManager 全局游戏状态管理器
type StateManager struct {
	groups map[int64]*GroupState
	mutex  sync.RWMutex
}

var GlobalGameState *StateManager

// GroupStateData 用于导出的数据结构
type GroupStateData struct {
	GroupID    int64
	Characters map[string]*Character
}

func InitGameState() {
	GlobalGameState = &StateManager{
		groups: make(map[int64]*GroupState),
	}
}

// GetGroupState 获取或创建群组状态
func (m *StateManager) GetGroupState(groupID int64) *GroupState {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if state, exists := m.groups[groupID]; exists {
		return state
	}

	newState := &GroupState{
		GroupID:    groupID,
		Characters: make(map[string]*Character),
	}
	m.groups[groupID] = newState
	return newState
}

// AddCharacter 添加角色
func (g *GroupState) AddCharacter(char *Character) {
	g.Mutex.Lock()
	defer g.Mutex.Unlock()
	g.Characters[strings.ToLower(char.Name)] = char
}

// GetCharacter 获取角色
func (g *GroupState) GetCharacter(name string) *Character {
	g.Mutex.RLock()
	defer g.Mutex.RUnlock()
	return g.Characters[strings.ToLower(name)]
}

// RemoveCharacter 移除角色
func (g *GroupState) RemoveCharacter(name string) {
	g.Mutex.Lock()
	defer g.Mutex.Unlock()
	delete(g.Characters, strings.ToLower(name))
}

// GetStatusSummary生成状态摘要，用于注入 Prompt
func (g *GroupState) GetStatusSummary() string {
	g.Mutex.RLock()
	defer g.Mutex.RUnlock()

	if len(g.Characters) == 0 {
		return "当前没有玩家角色。"
	}

	var sb strings.Builder
	sb.WriteString("【当前角色状态】:\n")
	for _, char := range g.Characters {
		sb.WriteString(fmt.Sprintf("- %s (%s): HP %d/%d, STR %d\n",
			char.Name, char.Class, char.HP, char.MaxHP, char.STR))
	}
	return sb.String()
}

// GetCharacterStatus 获取单个角色的详细状态
func (g *GroupState) GetCharacterStatus(name string) string {
	char := g.GetCharacter(name)
	if char == nil {
		return fmt.Sprintf("找不到角色: %s", name)
	}

	return fmt.Sprintf("【角色详情】\nName: %s\nClass: %s\nHP: %d/%d\nSTR: %d",
		char.Name, char.Class, char.HP, char.MaxHP, char.STR)
}

// ExportData 导出所有游戏状态
func (m *StateManager) ExportData() map[int64]*GroupStateData {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	data := make(map[int64]*GroupStateData)
	for id, gs := range m.groups {
		gs.Mutex.RLock()
		charsCopy := make(map[string]*Character)
		for k, v := range gs.Characters {
			// Deep copy character struct
			cVal := *v
			charsCopy[k] = &cVal
		}

		data[id] = &GroupStateData{
			GroupID:    gs.GroupID,
			Characters: charsCopy,
		}
		gs.Mutex.RUnlock()
	}
	return data
}

// ImportData 导入游戏状态
func (m *StateManager) ImportData(data map[int64]*GroupStateData) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for id, gData := range data {
		newState := &GroupState{
			GroupID:    gData.GroupID,
			Characters: make(map[string]*Character),
		}

		for k, v := range gData.Characters {
			cVal := *v // Copy value
			newState.Characters[k] = &cVal
		}
		m.groups[id] = newState
	}
}
