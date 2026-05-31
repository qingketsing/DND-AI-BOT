package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

type PGOutboxEventStore struct {
	db *sql.DB
}

func NewPGOutboxEventStore(db *sql.DB) *PGOutboxEventStore {
	return &PGOutboxEventStore{db: db}
}

func (s *PGOutboxEventStore) Create(ctx context.Context, event model.OutboxEvent) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO outbox_events (
			id, aggregate_type, aggregate_id, event_type, payload_json,
			status, attempt_count, last_error, created_at, published_at, next_retry_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, event.ID, event.AggregateType, event.AggregateID, event.EventType, []byte(event.PayloadJSON), string(event.Status), event.AttemptCount, event.LastError, event.CreatedAt, event.PublishedAt, event.NextRetryAt, event.UpdatedAt)
	return err
}

func (s *PGOutboxEventStore) GetPending(ctx context.Context, limit int) ([]model.OutboxEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, aggregate_type, aggregate_id, event_type, payload_json,
		       status, attempt_count, last_error, created_at, published_at, next_retry_at, updated_at
		FROM outbox_events
		WHERE status = $1
		   OR (status = $2 AND (next_retry_at IS NULL OR next_retry_at <= NOW()))
		ORDER BY created_at ASC, id ASC
		LIMIT $3
	`, string(model.OutboxEventPending), string(model.OutboxEventFailed), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]model.OutboxEvent, 0)
	for rows.Next() {
		var (
			event       model.OutboxEvent
			status      string
			payloadJSON []byte
			publishedAt sql.NullTime
			nextRetryAt sql.NullTime
		)
		if err := rows.Scan(
			&event.ID,
			&event.AggregateType,
			&event.AggregateID,
			&event.EventType,
			&payloadJSON,
			&status,
			&event.AttemptCount,
			&event.LastError,
			&event.CreatedAt,
			&publishedAt,
			&nextRetryAt,
			&event.UpdatedAt,
		); err != nil {
			return nil, err
		}
		event.PayloadJSON = json.RawMessage(append([]byte(nil), payloadJSON...))
		event.Status = model.OutboxEventStatus(status)
		if publishedAt.Valid {
			value := publishedAt.Time
			event.PublishedAt = &value
		}
		if nextRetryAt.Valid {
			value := nextRetryAt.Time
			event.NextRetryAt = &value
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func (s *PGOutboxEventStore) MarkPublished(ctx context.Context, id string, publishedAt time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE outbox_events
		SET status = $2, published_at = $3, next_retry_at = NULL, last_error = '', updated_at = $3
		WHERE id = $1
	`, id, string(model.OutboxEventPublished), publishedAt)
	return mapOutboxEventExecResult(result, err)
}

func (s *PGOutboxEventStore) MarkFailedAttempt(ctx context.Context, id string, failedAt time.Time, nextRetryAt time.Time, lastError string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE outbox_events
		SET status = $2, attempt_count = attempt_count + 1, last_error = $3, next_retry_at = $4, updated_at = $5
		WHERE id = $1
	`, id, string(model.OutboxEventFailed), lastError, nextRetryAt, failedAt)
	return mapOutboxEventExecResult(result, err)
}

func (s *PGOutboxEventStore) ListPublishedWithQueuedJobs(ctx context.Context, limit int) ([]repository.OutboxJobRepairCandidate, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT e.id, e.aggregate_type, e.aggregate_id, e.event_type, e.payload_json,
		       e.status, e.attempt_count, e.last_error, e.created_at, e.published_at, e.next_retry_at, e.updated_at,
		       j.id, j.message_id, j.session_id, j.user_id, j.status,
		       j.attempt_count, j.max_attempts, j.worker_id,
		       j.queued_at, j.started_at, j.finished_at,
		       j.last_error_code, j.last_error_message, j.latency_ms,
		       j.next_retry_at, j.heartbeat_at, j.created_at, j.updated_at
		FROM outbox_events e
		JOIN message_jobs j ON j.id = e.aggregate_id
		WHERE e.status = $1 AND j.status = $2
		ORDER BY e.published_at ASC, e.id ASC
		LIMIT $3
	`, string(model.OutboxEventPublished), string(model.MessageJobQueued), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candidates := make([]repository.OutboxJobRepairCandidate, 0)
	for rows.Next() {
		var (
			event            model.OutboxEvent
			job              model.MessageJob
			eventStatus      string
			jobStatus        string
			payloadJSON      []byte
			publishedAt      sql.NullTime
			eventNextRetryAt sql.NullTime
			startedAt        sql.NullTime
			finishedAt       sql.NullTime
			jobNextRetryAt   sql.NullTime
			heartbeatAt      sql.NullTime
		)
		if err := rows.Scan(
			&event.ID,
			&event.AggregateType,
			&event.AggregateID,
			&event.EventType,
			&payloadJSON,
			&eventStatus,
			&event.AttemptCount,
			&event.LastError,
			&event.CreatedAt,
			&publishedAt,
			&eventNextRetryAt,
			&event.UpdatedAt,
			&job.ID,
			&job.MessageID,
			&job.SessionID,
			&job.UserID,
			&jobStatus,
			&job.AttemptCount,
			&job.MaxAttempts,
			&job.WorkerID,
			&job.QueuedAt,
			&startedAt,
			&finishedAt,
			&job.LastErrorCode,
			&job.LastErrorMessage,
			&job.LatencyMS,
			&jobNextRetryAt,
			&heartbeatAt,
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, err
		}
		event.PayloadJSON = json.RawMessage(append([]byte(nil), payloadJSON...))
		event.Status = model.OutboxEventStatus(eventStatus)
		if publishedAt.Valid {
			value := publishedAt.Time
			event.PublishedAt = &value
		}
		if eventNextRetryAt.Valid {
			value := eventNextRetryAt.Time
			event.NextRetryAt = &value
		}
		job.Status = model.MessageJobStatus(jobStatus)
		if startedAt.Valid {
			value := startedAt.Time
			job.StartedAt = &value
		}
		if finishedAt.Valid {
			value := finishedAt.Time
			job.FinishedAt = &value
		}
		if jobNextRetryAt.Valid {
			value := jobNextRetryAt.Time
			job.NextRetryAt = &value
		}
		if heartbeatAt.Valid {
			value := heartbeatAt.Time
			job.HeartbeatAt = &value
		}
		candidates = append(candidates, repository.OutboxJobRepairCandidate{
			Event: event,
			Job:   job,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return candidates, nil
}

func mapOutboxEventExecResult(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return repository.ErrOutboxEventNotFound
	}
	return nil
}
