package postgres

import (
	"context"
	"database/sql"
	"errors"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
	"github.com/jackc/pgx/v5/pgconn"
)

type PGSessionStore struct {
	db *sql.DB
}

type sessionMessageAsyncFields struct {
	sourceJobID      string
	replyToMessageID string
}

const (
	sessionMessagesReplyToConstraint   = "uq_session_messages_assistant_reply_to_message_id"
	sessionMessagesSourceJobConstraint = "uq_session_messages_assistant_source_job_id"
)

type idempotentSessionMessageConflictError struct {
	cause error
}

func (e *idempotentSessionMessageConflictError) Error() string {
	return e.cause.Error()
}

func (e *idempotentSessionMessageConflictError) Unwrap() error {
	return e.cause
}

func (e *idempotentSessionMessageConflictError) IdempotentSuccess() bool {
	return true
}

// NewPGSessionStore 创建基于 database/sql 的 Session PG 存储实现。
func NewPGSessionStore(db *sql.DB) *PGSessionStore {
	return &PGSessionStore{db: db}
}

// UpsertSession 使用 UPSERT 语句将会话数据写入数据库，确保在高并发场景下数据的一致性和完整性。注意，这里是全量重写历史记录，适用于历史记录较短的情况。如果历史记录较长，可能需要优化为增量更新。
func (s *PGSessionStore) UpsertSession(ctx context.Context, session *model.Session) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := upsertSessionTx(ctx, tx, session); err != nil {
		return err
	}
	if err := replaceSessionMessagesTx(ctx, tx, session, true); err != nil {
		return err
	}

	return tx.Commit()
}

// EnqueueAsyncMessage 在单一事务内写入 session、queued job 与 pending outbox event。
func (s *PGSessionStore) EnqueueAsyncMessage(ctx context.Context, session *model.Session, job model.MessageJob, event model.OutboxEvent) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := upsertSessionTx(ctx, tx, session); err != nil {
		return err
	}
	if err := replaceSessionMessagesTx(ctx, tx, session, false); err != nil {
		return err
	}
	if err := insertMessageJobTx(ctx, tx, job); err != nil {
		return err
	}
	if err := insertOutboxEventTx(ctx, tx, event); err != nil {
		return err
	}

	return tx.Commit()
}

func loadSessionMessageAsyncFields(ctx context.Context, tx *sql.Tx, sessionID string) (map[string]sessionMessageAsyncFields, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, source_job_id, reply_to_message_id
		FROM session_messages
		WHERE session_id = $1
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fieldsByMessageID := make(map[string]sessionMessageAsyncFields)
	for rows.Next() {
		var (
			messageID        string
			sourceJobID      sql.NullString
			replyToMessageID sql.NullString
		)
		if err := rows.Scan(&messageID, &sourceJobID, &replyToMessageID); err != nil {
			return nil, err
		}
		fieldsByMessageID[messageID] = sessionMessageAsyncFields{
			sourceJobID:      sourceJobID.String,
			replyToMessageID: replyToMessageID.String,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return fieldsByMessageID, nil
}

// GetSession 从数据库中查询会话数据，并构建完整的 Session 对象返回。如果会话不存在，返回 repository.ErrSessionNotFound 错误。
func (s *PGSessionStore) GetSession(ctx context.Context, sessionID string) (*model.Session, error) {
	var (
		id        string
		userID    string
		title     string
		channel   string
		createdAt sql.NullTime
		updatedAt sql.NullTime
	)

	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, title, channel, created_at, updated_at
		FROM sessions
		WHERE id = $1
	`, sessionID).Scan(&id, &userID, &title, &channel, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, user_name, content, sequence, source, source_job_id, reply_to_message_id, created_at
		FROM session_messages
		WHERE session_id = $1
		ORDER BY sequence ASC
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]model.HistoryRecord, 0)
	for rows.Next() {
		var (
			recordID         string
			userID           string
			userName         string
			content          string
			sequence         int64
			source           string
			sourceJobID      sql.NullString
			replyToMessageID sql.NullString
			recordTime       sql.NullTime
		)

		if err := rows.Scan(&recordID, &userID, &userName, &content, &sequence, &source, &sourceJobID, &replyToMessageID, &recordTime); err != nil {
			return nil, err
		}

		history = append(history, model.HistoryRecord{
			ID: recordID,
			User: model.SessionUser{
				ID:   userID,
				Name: userName,
			},
			Message: model.Message{
				Content: content,
			},
			Sequence:         sequence,
			Source:           model.MessageSource(source),
			SourceJobID:      sourceJobID.String,
			ReplyToMessageID: replyToMessageID.String,
			CreatedAt:        recordTime.Time,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &model.Session{
		ID:        id,
		UserID:    userID,
		Title:     title,
		Channel:   model.Channel(channel),
		History:   history,
		CreatedAt: createdAt.Time,
		UpdatedAt: updatedAt.Time,
	}, nil
}

// ListSessionsByUserID 返回指定用户的所有会话。
func (s *PGSessionStore) ListSessionsByUserID(ctx context.Context, userID string) ([]*model.Session, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, title, channel, created_at, updated_at
		FROM sessions
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]*model.Session, 0)
	for rows.Next() {
		var (
			id            string
			sessionUserID string
			title         string
			channel       string
			createdAt     sql.NullTime
			updatedAt     sql.NullTime
		)

		if err := rows.Scan(&id, &sessionUserID, &title, &channel, &createdAt, &updatedAt); err != nil {
			return nil, err
		}

		sessions = append(sessions, &model.Session{
			ID:        id,
			UserID:    sessionUserID,
			Title:     title,
			Channel:   model.Channel(channel),
			History:   []model.HistoryRecord{},
			CreatedAt: createdAt.Time,
			UpdatedAt: updatedAt.Time,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}

// DeleteSession 删除指定会话及其消息历史。
func (s *PGSessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM session_messages WHERE session_id = $1`, sessionID); err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return repository.ErrSessionNotFound
	}

	return tx.Commit()
}

func messageRoleFromSource(source model.MessageSource) string {
	switch source {
	case model.MessageSourceUser:
		return "user"
	case model.MessageSourceAgent:
		return "assistant"
	case model.MessageSourceSystem:
		return "system"
	default:
		return string(source)
	}
}

func upsertSessionTx(ctx context.Context, tx *sql.Tx, session *model.Session) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, title, channel, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE
		SET user_id = EXCLUDED.user_id,
		    title = EXCLUDED.title,
		    channel = EXCLUDED.channel,
		    updated_at = EXCLUDED.updated_at
	`, session.ID, session.UserID, session.Title, string(session.Channel), session.CreatedAt, session.UpdatedAt)
	return err
}

func replaceSessionMessagesTx(ctx context.Context, tx *sql.Tx, session *model.Session, preserveExisting bool) error {
	existingAsyncFields := make(map[string]sessionMessageAsyncFields)
	if preserveExisting {
		var err error
		existingAsyncFields, err = loadSessionMessageAsyncFields(ctx, tx, session.ID)
		if err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM session_messages WHERE session_id = $1`, session.ID); err != nil {
		return err
	}

	for _, record := range session.History {
		if preserveExisting {
			fields := existingAsyncFields[record.ID]
			if record.SourceJobID == "" {
				record.SourceJobID = fields.sourceJobID
			}
			if record.ReplyToMessageID == "" {
				record.ReplyToMessageID = fields.replyToMessageID
			}
		}
		if err := insertSessionMessageTx(ctx, tx, session.ID, record); err != nil {
			return err
		}
	}

	return nil
}

func insertSessionMessageTx(ctx context.Context, tx *sql.Tx, sessionID string, record model.HistoryRecord) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO session_messages (
			id, session_id, user_id, user_name, content, sequence, source, role, source_job_id, reply_to_message_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`,
		record.ID,
		sessionID,
		record.User.ID,
		record.User.Name,
		record.Message.Content,
		record.Sequence,
		string(record.Source),
		messageRoleFromSource(record.Source),
		nullableString(record.SourceJobID),
		nullableString(record.ReplyToMessageID),
		record.CreatedAt,
	)
	return normalizeSessionMessageWriteError(err)
}

func insertMessageJobTx(ctx context.Context, tx *sql.Tx, job model.MessageJob) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO message_jobs (
			id, message_id, session_id, user_id, status,
			attempt_count, max_attempts, worker_id,
			queued_at, started_at, finished_at,
			last_error_code, last_error_message, latency_ms,
			next_retry_at, heartbeat_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`, job.ID, job.MessageID, job.SessionID, job.UserID, string(job.Status), job.AttemptCount, job.MaxAttempts, job.WorkerID, job.QueuedAt, job.StartedAt, job.FinishedAt, job.LastErrorCode, job.LastErrorMessage, job.LatencyMS, job.NextRetryAt, job.HeartbeatAt, job.CreatedAt, job.UpdatedAt)
	return err
}

func insertOutboxEventTx(ctx context.Context, tx *sql.Tx, event model.OutboxEvent) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO outbox_events (
			id, aggregate_type, aggregate_id, event_type, payload_json,
			status, attempt_count, last_error, created_at, published_at, next_retry_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, event.ID, event.AggregateType, event.AggregateID, event.EventType, []byte(event.PayloadJSON), string(event.Status), event.AttemptCount, event.LastError, event.CreatedAt, event.PublishedAt, event.NextRetryAt, event.UpdatedAt)
	return err
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func normalizeSessionMessageWriteError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case sessionMessagesReplyToConstraint, sessionMessagesSourceJobConstraint:
			return &idempotentSessionMessageConflictError{cause: err}
		}
	}
	return err
}
