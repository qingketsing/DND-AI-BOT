package session

import (
	"sync"

	openai "github.com/sashabaranov/go-openai"
)

// Session 管理单个群组的对话上下文
type Session struct {
	GroupID   int64
	History   []openai.ChatCompletionMessage
	Summary   string // 长期记忆/剧情摘要
	MaxLength int
	Mutex     sync.RWMutex
}

// Manager 全局会话管理器
type Manager struct {
	sessions map[int64]*Session
	mutex    sync.RWMutex
}

var GlobalManager *Manager

func InitManager() {
	GlobalManager = &Manager{
		sessions: make(map[int64]*Session),
	}
}

// GetSession 获取或创建群组会话
func (m *Manager) GetSession(groupID int64) *Session {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if sess, exists := m.sessions[groupID]; exists {
		return sess
	}

	newSess := &Session{
		GroupID:   groupID,
		History:   make([]openai.ChatCompletionMessage, 0),
		MaxLength: 50, // 增加到 50，给总结机制留出缓冲空间（通常每20条触发总结）
	}
	m.sessions[groupID] = newSess
	return newSess
}

// AddMessage 添加消息并执行滑动窗口修剪
func (s *Session) AddMessage(role string, content string) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	msg := openai.ChatCompletionMessage{
		Role:    role,
		Content: content,
	}
	s.History = append(s.History, msg)

	// Sliding Window: 如果超出最大长度，移除最早的消息
	// 仍然保留这个作为最后的防线，防止内存溢出
	if len(s.History) > s.MaxLength {
		s.History = s.History[len(s.History)-s.MaxLength:]
	}
}

// GetSummary 获取当前摘要
func (s *Session) GetSummary() string {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()
	return s.Summary
}

// UpdateSummary 更新摘要并修剪历史
// keepCount: 保留最近的多少条消息作为上下文衔接
func (s *Session) UpdateSummary(newSummary string, keepCount int) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	
	s.Summary = newSummary
	
	// Prune History
	if len(s.History) > keepCount {
		s.History = s.History[len(s.History)-keepCount:]
	}
}

// GetHistory 获取当前历史记录副本
func (s *Session) GetHistory() []openai.ChatCompletionMessage {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()

	// 复制一份，防止外部修改影响内部状态
	copied := make([]openai.ChatCompletionMessage, len(s.History))
	copy(copied, s.History)
	return copied
}

// Clear 清空历史
func (s *Session) Clear() {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	s.History = make([]openai.ChatCompletionMessage, 0)
	s.Summary = ""
}
