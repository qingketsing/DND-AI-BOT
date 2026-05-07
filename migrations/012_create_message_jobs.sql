CREATE TABLE IF NOT EXISTS message_jobs (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    status TEXT NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    worker_id TEXT NOT NULL DEFAULT '',
    queued_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    last_error_code TEXT NOT NULL DEFAULT '',
    last_error_message TEXT NOT NULL DEFAULT '',
    latency_ms BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT chk_message_jobs_status
        CHECK (status IN ('queued', 'published', 'processing', 'completed', 'retryable_failed', 'failed', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_message_jobs_status_created_at
    ON message_jobs(status, created_at);
CREATE INDEX IF NOT EXISTS idx_message_jobs_session_id_created_at
    ON message_jobs(session_id, created_at);
CREATE INDEX IF NOT EXISTS idx_message_jobs_message_id
    ON message_jobs(message_id);
CREATE INDEX IF NOT EXISTS idx_message_jobs_user_id_created_at
    ON message_jobs(user_id, created_at);
