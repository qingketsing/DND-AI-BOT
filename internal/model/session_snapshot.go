package model

import "time"

type SessionSnapshot struct {
	ID        string                  `json:"id"`
	Channel   Channel                 `json:"channel"`
	History   []HistoryRecordSnapshot `json:"history"`
	CreatedAt time.Time               `json:"created_at"`
	UpdatedAt time.Time               `json:"updated_at"`
}

type HistoryRecordSnapshot struct {
	ID        string          `json:"id"`
	User      UserSnapshot    `json:"user"`
	Message   MessageSnapshot `json:"message"`
	Sequence  int64           `json:"sequence"`
	Source    MessageSource   `json:"source"`
	CreatedAt time.Time       `json:"created_at"`
}

type UserSnapshot struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type MessageSnapshot struct {
	Content string `json:"content"`
}

func (s *Session) ToSnapshot() SessionSnapshot {
	return SessionSnapshot{
		ID:        s.ID,
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
			ID: record.ID,
			User: UserSnapshot{
				ID:   record.User.ID,
				Name: record.User.Name,
			},
			Message: MessageSnapshot{
				Content: record.Message.Content,
			},
			Sequence:  record.Sequence,
			Source:    record.Source,
			CreatedAt: record.CreatedAt,
		}
	}

	return snapshots
}

// 将历史记录快照转换回原始格式，进行深复制以确保恢复后的会话数据与快照数据完全独立
func restoreHistoryRecords(records []HistoryRecordSnapshot) []HistoryRecord {
	restored := make([]HistoryRecord, len(records))
	for i, record := range records {
		restored[i] = HistoryRecord{
			ID: record.ID,
			User: User{
				ID:   record.User.ID,
				Name: record.User.Name,
			},
			Message: Message{
				Content: record.Message.Content,
			},
			Sequence:  record.Sequence,
			Source:    record.Source,
			CreatedAt: record.CreatedAt,
		}
	}

	return restored
}
