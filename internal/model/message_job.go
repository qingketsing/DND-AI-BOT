package model

import "time"

type MessageJobStatus string

const (
	MessageJobQueued          MessageJobStatus = "queued"
	MessageJobPublished       MessageJobStatus = "published"
	MessageJobProcessing      MessageJobStatus = "processing"
	MessageJobCompleted       MessageJobStatus = "completed"
	MessageJobRetryableFailed MessageJobStatus = "retryable_failed"
	MessageJobFailed          MessageJobStatus = "failed"
	MessageJobCancelled       MessageJobStatus = "cancelled"
)

type MessageJob struct {
	ID               string           `json:"id"`
	MessageID        string           `json:"message_id"`
	SessionID        string           `json:"session_id"`
	UserID           string           `json:"user_id"`
	Status           MessageJobStatus `json:"status"`
	AttemptCount     int              `json:"attempt_count"`
	MaxAttempts      int              `json:"max_attempts"`
	WorkerID         string           `json:"worker_id"`
	QueuedAt         time.Time        `json:"queued_at"`
	StartedAt        *time.Time       `json:"started_at,omitempty"`
	FinishedAt       *time.Time       `json:"finished_at,omitempty"`
	NextRetryAt      *time.Time       `json:"next_retry_at,omitempty"`
	HeartbeatAt      *time.Time       `json:"heartbeat_at,omitempty"`
	LastErrorCode    string           `json:"last_error_code"`
	LastErrorMessage string           `json:"last_error_message"`
	LatencyMS        int64            `json:"latency_ms"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}
