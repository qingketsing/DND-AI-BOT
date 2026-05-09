package model

import (
	"fmt"
	"time"
)

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
	UserID    string          `json:"user_id"`
	Title     string          `json:"title"`
	Channel   Channel         `json:"channel"`
	History   []HistoryRecord `json:"history"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// 历史记录包含用户体，消息体，序号（防止由于时间戳导致的和系统输出的一些信息顺序错乱），来源
type HistoryRecord struct {
	ID               string        `json:"id"`
	User             SessionUser   `json:"user"`
	Message          Message       `json:"message"`
	Sequence         int64         `json:"sequence"`
	Source           MessageSource `json:"source"`
	SourceJobID      string        `json:"source_job_id,omitempty"`
	ReplyToMessageID string        `json:"reply_to_message_id,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
}

// SessionUser is the lightweight user snapshot stored in session history.
type SessionUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// 同上，这里的消息体也是用于快照保存
type Message struct {
	Content string `json:"content"`
}

var systemUser = SessionUser{
	ID:   "system",
	Name: "system",
}

// 创建新的会话
func NewSession(id string, userID string, channel Channel, now time.Time) *Session {
	return &Session{
		ID:        id,
		UserID:    userID,
		Title:     "新会话",
		Channel:   channel,
		History:   make([]HistoryRecord, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// 在会话历史记录中加入新的用户消息记录
func (s *Session) AppendUserMessage(user SessionUser, content string, now time.Time) HistoryRecord {
	record := s.newRecord(MessageSourceUser, user, content, now)
	s.appendRecord(record)
	return record
}

// 在会话历史记录中加入新的Agent消息记录
func (s *Session) AppendAgentMessage(user SessionUser, content string, now time.Time) HistoryRecord {
	record := s.newRecord(MessageSourceAgent, user, content, now)
	s.appendRecord(record)
	return record
}

// 在会话历史记录中加入新的 assistant 回复，并显式记录关联的用户消息和任务。
func (s *Session) AppendAssistantReply(user SessionUser, content string, replyToMessageID string, sourceJobID string, now time.Time) HistoryRecord {
	record := s.newRecord(MessageSourceAgent, user, content, now)
	record.ReplyToMessageID = replyToMessageID
	record.SourceJobID = sourceJobID
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

func (s *Session) newRecord(source MessageSource, user SessionUser, content string, now time.Time) HistoryRecord {
	sequence := s.NextSequence()
	return HistoryRecord{
		ID:        fmt.Sprintf("%s-msg-%d", s.ID, sequence),
		User:      user,
		Message:   Message{Content: content},
		Sequence:  sequence,
		Source:    source,
		CreatedAt: now,
	}
}
