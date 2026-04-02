package model

import "time"

// MessageSource定义为字符串
type MessageSource string

// Channel定义为会话的接入渠道。
type Channel string

const (
	MessageSourceUser   MessageSource = "user"
	MessageSourceAgent  MessageSource = "agent"
	MessageSourceSystem MessageSource = "system"

	ChannelWeb Channel = "web"
	ChannelBot Channel = "bot"
)

// Session结构体
type Session struct {
	ID        string          `json:"id"`
	Channel   Channel         `json:"channel"`
	History   []HistoryRecord `json:"history"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// 历史记录包含用户体，消息体，序号（防止由于时间戳导致的和系统输出的一些信息顺序错乱），来源
type HistoryRecord struct {
	ID        string        `json:"id"`
	User      User          `json:"user"`
	Message   Message       `json:"message"`
	Sequence  int64         `json:"sequence"`
	Source    MessageSource `json:"source"`
	CreatedAt time.Time     `json:"created_at"`
}

// 这里的用户只是快照保存使用的用户体
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// 同上，这里的消息体也是用于快照保存
type Message struct {
	Content string `json:"content"`
}

var systemUser = User{
	ID:   "system",
	Name: "system",
}

// 创建新的会话
func NewSession(id string, channel Channel, now time.Time) *Session {
	return &Session{
		ID:        id,
		Channel:   channel,
		History:   make([]HistoryRecord, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// 在会话历史记录中加入新的用户消息记录
func (s *Session) AppendUserMessage(user User, content string, now time.Time) HistoryRecord {
	record := s.newRecord(MessageSourceUser, user, content, now)
	s.appendRecord(record)
	return record
}

// 在会话历史记录中加入新的Agent消息记录
func (s *Session) AppendAgentMessage(user User, content string, now time.Time) HistoryRecord {
	record := s.newRecord(MessageSourceAgent, user, content, now)
	s.appendRecord(record)
	return record
}

// 在会话历史记录中加入新的系统消息记录
func (s *Session) AppendSystemMessage(content string, now time.Time) HistoryRecord {
	record := s.newRecord(MessageSourceSystem, systemUser, content, now)
	s.appendRecord(record)
	return record
}

// 返回历史消息的副本
func (s *Session) HistoryRecords() []HistoryRecord {
	history := make([]HistoryRecord, len(s.History))
	copy(history, s.History)
	return history
}

// 获取历史消息中的最后一条消息
func (s *Session) LastRecord() (HistoryRecord, bool) {
	if len(s.History) == 0 {
		return HistoryRecord{}, false
	}

	return s.History[len(s.History)-1], true
}

// 生成下一个消息序号
func (s *Session) NextSequence() int64 {
	return int64(len(s.History) + 1)
}

func (s *Session) appendRecord(record HistoryRecord) {
	s.History = append(s.History, record)
	s.UpdatedAt = record.CreatedAt
}

func (s *Session) newRecord(source MessageSource, user User, content string, now time.Time) HistoryRecord {
	return HistoryRecord{
		ID:        user.ID,
		User:      user,
		Message:   Message{Content: content},
		Sequence:  s.NextSequence(),
		Source:    source,
		CreatedAt: now,
	}
}
