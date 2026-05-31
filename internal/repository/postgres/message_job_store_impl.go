package postgres

import (
	"context"
	"database/sql"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

type PGMessageJobStore struct {
	db *sql.DB
}

func NewPGMessageJobStore(db *sql.DB) *PGMessageJobStore {
	return &PGMessageJobStore{db: db}
}

func (s *PGMessageJobStore) Create(ctx context.Context, job model.MessageJob) error {
	_, err := s.db.ExecContext(ctx, `
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

func (s *PGMessageJobStore) GetByID(ctx context.Context, jobID string) (*model.MessageJob, error) {
	return s.getOne(ctx, `
		SELECT id, message_id, session_id, user_id, status,
		       attempt_count, max_attempts, worker_id,
		       queued_at, started_at, finished_at,
		       last_error_code, last_error_message, latency_ms,
		       next_retry_at, heartbeat_at, created_at, updated_at
		FROM message_jobs
		WHERE id = $1
	`, jobID)
}

func (s *PGMessageJobStore) GetByMessageID(ctx context.Context, messageID string) (*model.MessageJob, error) {
	return s.getOne(ctx, `
		SELECT id, message_id, session_id, user_id, status,
		       attempt_count, max_attempts, worker_id,
		       queued_at, started_at, finished_at,
		       last_error_code, last_error_message, latency_ms,
		       next_retry_at, heartbeat_at, created_at, updated_at
		FROM message_jobs
		WHERE message_id = $1
	`, messageID)
}

func (s *PGMessageJobStore) MarkPublished(ctx context.Context, jobID string, publishedAt time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE message_jobs
		SET status = $2, next_retry_at = NULL, updated_at = $3
		WHERE id = $1 AND status = $4
	`, jobID, string(model.MessageJobPublished), publishedAt, string(model.MessageJobQueued))
	return mapMessageJobExecResult(result, err)
}

func (s *PGMessageJobStore) MarkProcessing(ctx context.Context, jobID string, workerID string, startedAt time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE message_jobs
		SET status = $2, worker_id = $3, started_at = $4, heartbeat_at = $4, next_retry_at = NULL, updated_at = $4
		WHERE id = $1 AND status = $5
	`, jobID, string(model.MessageJobProcessing), workerID, startedAt, string(model.MessageJobPublished))
	return mapMessageJobExecResult(result, err)
}

func (s *PGMessageJobStore) MarkCompleted(ctx context.Context, jobID string, finishedAt time.Time, latencyMS int64) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE message_jobs
		SET status = $2, finished_at = $3, latency_ms = $4, next_retry_at = NULL, updated_at = $3
		WHERE id = $1 AND status = $5
	`, jobID, string(model.MessageJobCompleted), finishedAt, latencyMS, string(model.MessageJobProcessing))
	return mapMessageJobExecResult(result, err)
}

func (s *PGMessageJobStore) MarkRetryableFailed(ctx context.Context, jobID string, finishedAt time.Time, errorCode string, errorMessage string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE message_jobs
		SET status = $2, finished_at = $3, last_error_code = $4, last_error_message = $5, updated_at = $3
		WHERE id = $1 AND status IN ($6, $7)
	`, jobID, string(model.MessageJobRetryableFailed), finishedAt, errorCode, errorMessage, string(model.MessageJobPublished), string(model.MessageJobProcessing))
	return mapMessageJobExecResult(result, err)
}

func (s *PGMessageJobStore) MarkFailed(ctx context.Context, jobID string, finishedAt time.Time, errorCode string, errorMessage string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE message_jobs
		SET status = $2, finished_at = $3, last_error_code = $4, last_error_message = $5, next_retry_at = NULL, updated_at = $3
		WHERE id = $1 AND status IN ($6, $7)
	`, jobID, string(model.MessageJobFailed), finishedAt, errorCode, errorMessage, string(model.MessageJobProcessing), string(model.MessageJobRetryableFailed))
	return mapMessageJobExecResult(result, err)
}

func (s *PGMessageJobStore) IncrementAttempt(ctx context.Context, jobID string) error {
	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx, `
		UPDATE message_jobs
		SET attempt_count = attempt_count + 1, updated_at = $2
		WHERE id = $1
	`, jobID, now)
	return mapMessageJobExecResult(result, err)
}

func (s *PGMessageJobStore) ListStaleProcessing(ctx context.Context, cutoff time.Time, limit int) ([]model.MessageJob, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, message_id, session_id, user_id, status,
		       attempt_count, max_attempts, worker_id,
		       queued_at, started_at, finished_at,
		       last_error_code, last_error_message, latency_ms,
		       next_retry_at, heartbeat_at, created_at, updated_at
		FROM message_jobs
		WHERE status = $1
		  AND COALESCE(heartbeat_at, updated_at) < $2
		ORDER BY updated_at ASC, id ASC
		LIMIT $3
	`, string(model.MessageJobProcessing), cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMessageJobRows(rows)
}

func (s *PGMessageJobStore) ListRetryDue(ctx context.Context, now time.Time, limit int) ([]model.MessageJob, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, message_id, session_id, user_id, status,
		       attempt_count, max_attempts, worker_id,
		       queued_at, started_at, finished_at,
		       last_error_code, last_error_message, latency_ms,
		       next_retry_at, heartbeat_at, created_at, updated_at
		FROM message_jobs
		WHERE status = $1
		  AND next_retry_at IS NOT NULL
		  AND next_retry_at <= $2
		  AND attempt_count < max_attempts
		ORDER BY next_retry_at ASC, id ASC
		LIMIT $3
	`, string(model.MessageJobRetryableFailed), now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMessageJobRows(rows)
}

func (s *PGMessageJobStore) MarkRetryScheduled(ctx context.Context, jobID string, failedAt time.Time, nextRetryAt time.Time, errorCode string, errorMessage string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE message_jobs
		SET status = $2, finished_at = $3, next_retry_at = $4,
		    last_error_code = $5, last_error_message = $6, updated_at = $3
		WHERE id = $1 AND status IN ($7, $8)
	`, jobID, string(model.MessageJobRetryableFailed), failedAt, nextRetryAt, errorCode, errorMessage, string(model.MessageJobPublished), string(model.MessageJobProcessing))
	return mapMessageJobExecResult(result, err)
}

func (s *PGMessageJobStore) RequeueRetryableWithOutbox(ctx context.Context, job model.MessageJob, event model.OutboxEvent, requeuedAt time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		UPDATE message_jobs
		SET status = $2, worker_id = '', started_at = NULL, finished_at = NULL,
		    next_retry_at = NULL, heartbeat_at = NULL, updated_at = $3
		WHERE id = $1 AND status = $4
	`, job.ID, string(model.MessageJobQueued), requeuedAt, string(model.MessageJobRetryableFailed))
	if err := mapMessageJobExecResult(result, err); err != nil {
		return err
	}
	if err := insertOutboxEventTx(ctx, tx, event); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *PGMessageJobStore) MarkHeartbeat(ctx context.Context, jobID string, heartbeatAt time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE message_jobs
		SET heartbeat_at = $2, updated_at = $2
		WHERE id = $1 AND status = $3
	`, jobID, heartbeatAt, string(model.MessageJobProcessing))
	return mapMessageJobExecResult(result, err)
}

func (s *PGMessageJobStore) RepairPublished(ctx context.Context, jobID string, repairedAt time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE message_jobs
		SET status = $2, updated_at = $3
		WHERE id = $1 AND status = $4
	`, jobID, string(model.MessageJobPublished), repairedAt, string(model.MessageJobQueued))
	return mapMessageJobExecResult(result, err)
}

func (s *PGMessageJobStore) getOne(ctx context.Context, query string, arg string) (*model.MessageJob, error) {
	rows, err := s.db.QueryContext(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs, err := scanMessageJobRows(rows)
	if err != nil {
		return nil, err
	}
	if len(jobs) == 0 {
		return nil, repository.ErrMessageJobNotFound
	}
	return &jobs[0], nil
}

func scanMessageJobRows(rows *sql.Rows) ([]model.MessageJob, error) {
	jobs := make([]model.MessageJob, 0)
	for rows.Next() {
		job, err := scanMessageJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return jobs, nil
}

type messageJobScanner interface {
	Scan(dest ...any) error
}

func scanMessageJob(scanner messageJobScanner) (model.MessageJob, error) {
	var (
		job         model.MessageJob
		status      string
		startedAt   sql.NullTime
		finishedAt  sql.NullTime
		nextRetryAt sql.NullTime
		heartbeatAt sql.NullTime
	)
	err := scanner.Scan(
		&job.ID,
		&job.MessageID,
		&job.SessionID,
		&job.UserID,
		&status,
		&job.AttemptCount,
		&job.MaxAttempts,
		&job.WorkerID,
		&job.QueuedAt,
		&startedAt,
		&finishedAt,
		&job.LastErrorCode,
		&job.LastErrorMessage,
		&job.LatencyMS,
		&nextRetryAt,
		&heartbeatAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		return model.MessageJob{}, err
	}
	job.Status = model.MessageJobStatus(status)
	if startedAt.Valid {
		value := startedAt.Time
		job.StartedAt = &value
	}
	if finishedAt.Valid {
		value := finishedAt.Time
		job.FinishedAt = &value
	}
	if nextRetryAt.Valid {
		value := nextRetryAt.Time
		job.NextRetryAt = &value
	}
	if heartbeatAt.Valid {
		value := heartbeatAt.Time
		job.HeartbeatAt = &value
	}
	return job, nil
}

func mapMessageJobExecResult(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return repository.ErrMessageJobNotFound
	}
	return nil
}
