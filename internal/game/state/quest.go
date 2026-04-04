package state

// QuestProgress 表示任务在当前游戏中的最小进度信息。
type QuestProgress struct {
	ID          string
	Title       string
	Status      QuestStatus
	Description string
}
