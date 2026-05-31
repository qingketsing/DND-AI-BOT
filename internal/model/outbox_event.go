package model

import (
	"encoding/json"
	"time"
)

type OutboxEventStatus string

const (
	OutboxEventPending   OutboxEventStatus = "pending"
	OutboxEventPublished OutboxEventStatus = "published"
	OutboxEventFailed    OutboxEventStatus = "failed"
)

type OutboxEvent struct {
	ID            string            `json:"id"`
	AggregateType string            `json:"aggregate_type"`
	AggregateID   string            `json:"aggregate_id"`
	EventType     string            `json:"event_type"`
	PayloadJSON   json.RawMessage   `json:"payload_json"`
	Status        OutboxEventStatus `json:"status"`
	AttemptCount  int               `json:"attempt_count"`
	LastError     string            `json:"last_error"`
	CreatedAt     time.Time         `json:"created_at"`
	PublishedAt   *time.Time        `json:"published_at,omitempty"`
	NextRetryAt   *time.Time        `json:"next_retry_at,omitempty"`
	UpdatedAt     time.Time         `json:"updated_at"`
}
