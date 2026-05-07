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
			status, attempt_count, last_error, created_at, published_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, event.ID, event.AggregateType, event.AggregateID, event.EventType, []byte(event.PayloadJSON), string(event.Status), event.AttemptCount, event.LastError, event.CreatedAt, event.PublishedAt, event.UpdatedAt)
	return err
}

func (s *PGOutboxEventStore) GetPending(ctx context.Context, limit int) ([]model.OutboxEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, aggregate_type, aggregate_id, event_type, payload_json,
		       status, attempt_count, last_error, created_at, published_at, updated_at
		FROM outbox_events
		WHERE status IN ($1, $2)
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
		SET status = $2, published_at = $3, last_error = '', updated_at = $3
		WHERE id = $1
	`, id, string(model.OutboxEventPublished), publishedAt)
	return mapOutboxEventExecResult(result, err)
}

func (s *PGOutboxEventStore) MarkFailedAttempt(ctx context.Context, id string, failedAt time.Time, lastError string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE outbox_events
		SET status = $2, attempt_count = attempt_count + 1, last_error = $3, updated_at = $4
		WHERE id = $1
	`, id, string(model.OutboxEventFailed), lastError, failedAt)
	return mapOutboxEventExecResult(result, err)
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
