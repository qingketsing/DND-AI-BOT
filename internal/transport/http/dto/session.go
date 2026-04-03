package dto

import (
	"time"

	"DND-AI-BOT/internal/model"
)

// CreateSessionRequest 定义创建会话接口的请求体。
type CreateSessionRequest struct {
	Channel string `json:"channel"`
}

// SendMessageRequest 定义发送消息接口的请求体。
type SendMessageRequest struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	Content  string `json:"content"`
}

// SessionResponse 定义会话接口的统一响应结构。
type SessionResponse struct {
	ID        string             `json:"id"`
	Channel   string             `json:"channel"`
	History   []HistoryRecordDTO `json:"history"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

// HistoryRecordDTO 表示单条历史消息的传输结构。
type HistoryRecordDTO struct {
	ID        string     `json:"id"`
	User      UserDTO    `json:"user"`
	Message   MessageDTO `json:"message"`
	Sequence  int64      `json:"sequence"`
	Source    string     `json:"source"`
	CreatedAt time.Time  `json:"created_at"`
}

// UserDTO 表示 HTTP 响应中的用户信息。
type UserDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// MessageDTO 表示 HTTP 响应中的消息内容。
type MessageDTO struct {
	Content string `json:"content"`
}

// ErrorResponse 定义统一错误响应格式。
type ErrorResponse struct {
	Error ErrorDTO `json:"error"`
}

// ErrorDTO 描述错误码和面向调用方的错误信息。
type ErrorDTO struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ToSessionResponse 将领域会话模型转换为 HTTP 响应。
func ToSessionResponse(session *model.Session) SessionResponse {
	history := make([]HistoryRecordDTO, len(session.History))
	for i, record := range session.History {
		history[i] = ToHistoryRecordDTO(record)
	}

	return SessionResponse{
		ID:        session.ID,
		Channel:   string(session.Channel),
		History:   history,
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
	}
}

// ToHistoryRecordDTO 将领域历史记录转换为 HTTP 响应项。
func ToHistoryRecordDTO(record model.HistoryRecord) HistoryRecordDTO {
	return HistoryRecordDTO{
		ID: record.ID,
		User: UserDTO{
			ID:   record.User.ID,
			Name: record.User.Name,
		},
		Message: MessageDTO{
			Content: record.Message.Content,
		},
		Sequence:  record.Sequence,
		Source:    string(record.Source),
		CreatedAt: record.CreatedAt,
	}
}

// NewErrorResponse 创建统一错误响应体。
func NewErrorResponse(code string, message string) ErrorResponse {
	return ErrorResponse{
		Error: ErrorDTO{
			Code:    code,
			Message: message,
		},
	}
}
