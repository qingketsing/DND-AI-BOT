package postgres

import (
	"context"
	"database/sql"
	"errors"
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
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, job.ID, job.MessageID, job.SessionID, job.UserID, string(job.Status), job.AttemptCount, job.MaxAttempts, job.WorkerID, job.QueuedAt, job.StartedAt, job.FinishedAt, job.LastErrorCode, job.LastErrorMessage, job.LatencyMS, job.CreatedAt, job.UpdatedAt)
	return err
}

func (s *PGMessageJobStore) GetByID(ctx context.Context, jobID string) (*model.MessageJob, error) {
	return s.getOne(ctx, `
		SELECT id, message_id, session_id, user_id, status,
		       attempt_count, max_attempts, worker_id,
		       queued_at, started_at, finished_at,
		       last_error_code, last_error_message, latency_ms,
		       created_at, updated_at
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
		       created_at, updated_at
		FROM message_jobs
		WHERE message_id = $1
	`, messageID)
}

func (s *PGMessageJobStore) MarkPublished(ctx context.Context, jobID string, publishedAt time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE message_jobs
		SET status = $2, updated_at = $3
		WHERE id = $1
	`, jobID, string(model.MessageJobPublished), publishedAt)
	return mapMessageJobExecResult(result, err)
}

func (s *PGMessageJobStore) MarkProcessing(ctx context.Context, jobID string, workerID string, startedAt time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE message_jobs
		SET status = $2, worker_id = $3, started_at = $4, updated_at = $4
		WHERE id = $1
	`, jobID, string(model.MessageJobProcessing), workerID, startedAt)
	return mapMessageJobExecResult(result, err)
}

func (s *PGMessageJobStore) MarkCompleted(ctx context.Context, jobID string, finishedAt time.Time, latencyMS int64) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE message_jobs
		SET status = $2, finished_at = $3, latency_ms = $4, updated_at = $3
		WHERE id = $1
	`, jobID, string(model.MessageJobCompleted), finishedAt, latencyMS)
	return mapMessageJobExecResult(result, err)
}

func (s *PGMessageJobStore) MarkRetryableFailed(ctx context.Context, jobID string, finishedAt time.Time, errorCode string, errorMessage string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE message_jobs
		SET status = $2, finished_at = $3, last_error_code = $4, last_error_message = $5, updated_at = $3
		WHERE id = $1
	`, jobID, string(model.MessageJobRetryableFailed), finishedAt, errorCode, errorMessage)
	return mapMessageJobExecResult(result, err)
}

func (s *PGMessageJobStore) MarkFailed(ctx context.Context, jobID string, finishedAt time.Time, errorCode string, errorMessage string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE message_jobs
		SET status = $2, finished_at = $3, last_error_code = $4, last_error_message = $5, updated_at = $3
		WHERE id = $1
	`, jobID, string(model.MessageJobFailed), finishedAt, errorCode, errorMessage)
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

func (s *PGMessageJobStore) getOne(ctx context.Context, query string, arg string) (*model.MessageJob, error) {
	var (
		job               model.MessageJob
		status            string
		startedAt         sql.NullTime
		finishedAt        sql.NullTime
	)
	err := s.db.QueryRowContext(ctx, query, arg).Scan(
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
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrMessageJobNotFound
	}
	if err != nil {
		return nil, err
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
	return &job, nil
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
