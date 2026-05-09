package dto

import (
	"time"

	"DND-AI-BOT/internal/service"
)

// EnqueueMessageResponse 定义异步消息入队响应。
type EnqueueMessageResponse struct {
	MessageID string `json:"message_id"`
	JobID     string `json:"job_id"`
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
}

// MessageStatusResponse 定义异步消息状态查询响应。
type MessageStatusResponse struct {
	MessageID      string                `json:"message_id"`
	SessionID      string                `json:"session_id"`
	Status         string                `json:"status"`
	Job            MessageJobResponse    `json:"job"`
	AssistantReply *AssistantReplyRecord `json:"assistant_reply,omitempty"`
}

// MessageJobResponse 定义任务状态详情响应。
type MessageJobResponse struct {
	ID               string  `json:"id"`
	Status           string  `json:"status"`
	AttemptCount     int     `json:"attempt_count"`
	QueuedAt         string  `json:"queued_at"`
	StartedAt        *string `json:"started_at,omitempty"`
	FinishedAt       *string `json:"finished_at,omitempty"`
	LatencyMS        int64   `json:"latency_ms"`
	LastErrorCode    string  `json:"last_error_code,omitempty"`
	LastErrorMessage string  `json:"last_error_message,omitempty"`
}

// AssistantReplyRecord 定义 assistant 回复响应。
type AssistantReplyRecord struct {
	MessageID        string `json:"message_id"`
	Content          string `json:"content"`
	ReplyToMessageID string `json:"reply_to_message_id"`
	SourceJobID      string `json:"source_job_id"`
}

func ToEnqueueMessageResponse(result *service.EnqueueMessageResult) EnqueueMessageResponse {
	return EnqueueMessageResponse{
		MessageID: result.MessageID,
		JobID:     result.JobID,
		SessionID: result.SessionID,
		Status:    result.Status,
	}
}

func ToMessageStatusResponse(result *service.MessageStatusResult) MessageStatusResponse {
	response := MessageStatusResponse{
		MessageID: result.MessageID,
		SessionID: result.SessionID,
		Status:    result.Status,
		Job: MessageJobResponse{
			ID:               result.Job.ID,
			Status:           result.Job.Status,
			AttemptCount:     result.Job.AttemptCount,
			QueuedAt:         result.Job.QueuedAt.UTC().Format(timeLayoutRFC3339Nano),
			StartedAt:        formatOptionalTime(result.Job.StartedAt),
			FinishedAt:       formatOptionalTime(result.Job.FinishedAt),
			LatencyMS:        result.Job.LatencyMS,
			LastErrorCode:    result.Job.LastErrorCode,
			LastErrorMessage: result.Job.LastErrorMessage,
		},
	}
	if result.AssistantReply != nil {
		response.AssistantReply = &AssistantReplyRecord{
			MessageID:        result.AssistantReply.MessageID,
			Content:          result.AssistantReply.Content,
			ReplyToMessageID: result.AssistantReply.ReplyToMessageID,
			SourceJobID:      result.AssistantReply.SourceJobID,
		}
	}
	return response
}

const timeLayoutRFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"

func formatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(timeLayoutRFC3339Nano)
	return &formatted
}
