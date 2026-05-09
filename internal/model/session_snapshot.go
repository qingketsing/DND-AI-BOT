package model

import "time"

type SessionSnapshot struct {
	ID        string                  `json:"id"`
	UserID    string                  `json:"user_id"`
	Title     string                  `json:"title"`
	Channel   Channel                 `json:"channel"`
	History   []HistoryRecordSnapshot `json:"history"`
	CreatedAt time.Time               `json:"created_at"`
	UpdatedAt time.Time               `json:"updated_at"`
}

type HistoryRecordSnapshot struct {
	ID               string          `json:"id"`
	User             SessionUser     `json:"user"`
	Message          MessageSnapshot `json:"message"`
	Sequence         int64           `json:"sequence"`
	Source           MessageSource   `json:"source"`
	SourceJobID      string          `json:"source_job_id,omitempty"`
	ReplyToMessageID string          `json:"reply_to_message_id,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

type MessageSnapshot struct {
	Content string `json:"content"`
}

func (s *Session) ToSnapshot() SessionSnapshot {
	return SessionSnapshot{
		ID:        s.ID,
		UserID:    s.UserID,
		Title:     s.Title,
		Channel:   s.Channel,
		History:   toHistoryRecordSnapshots(s.History),
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

// 从快照恢复会话，恢复后的会话与原快照完全独立，修改恢复后的会话不会影响原快照数据
func RestoreSession(snapshot SessionSnapshot) *Session {
	return &Session{
		ID:        snapshot.ID,
		UserID:    snapshot.UserID,
		Title:     snapshot.Title,
		Channel:   snapshot.Channel,
		History:   restoreHistoryRecords(snapshot.History),
		CreatedAt: snapshot.CreatedAt,
		UpdatedAt: snapshot.UpdatedAt,
	}
}

// 将历史记录转换为快照格式，进行深复制以确保快照数据与原数据完全独立
func toHistoryRecordSnapshots(records []HistoryRecord) []HistoryRecordSnapshot {
	snapshots := make([]HistoryRecordSnapshot, len(records))
	for i, record := range records {
		snapshots[i] = HistoryRecordSnapshot{
			ID:   record.ID,
			User: record.User,
			Message: MessageSnapshot{
				Content: record.Message.Content,
			},
			Sequence:         record.Sequence,
			Source:           record.Source,
			SourceJobID:      record.SourceJobID,
			ReplyToMessageID: record.ReplyToMessageID,
			CreatedAt:        record.CreatedAt,
		}
	}

	return snapshots
}

// 将历史记录快照转换回原始格式，进行深复制以确保恢复后的会话数据与快照数据完全独立
func restoreHistoryRecords(records []HistoryRecordSnapshot) []HistoryRecord {
	restored := make([]HistoryRecord, len(records))
	for i, record := range records {
		restored[i] = HistoryRecord{
			ID:   record.ID,
			User: record.User,
			Message: Message{
				Content: record.Message.Content,
			},
			Sequence:         record.Sequence,
			Source:           record.Source,
			SourceJobID:      record.SourceJobID,
			ReplyToMessageID: record.ReplyToMessageID,
			CreatedAt:        record.CreatedAt,
		}
	}

	return restored
}
