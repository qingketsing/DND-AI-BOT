ALTER TABLE message_jobs
    ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS heartbeat_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_message_jobs_retryable_next_retry_at
    ON message_jobs(status, next_retry_at)
    WHERE status = 'retryable_failed';

CREATE INDEX IF NOT EXISTS idx_message_jobs_processing_updated_at
    ON message_jobs(status, updated_at)
    WHERE status = 'processing';

ALTER TABLE outbox_events
    ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_outbox_events_dispatch_due
    ON outbox_events(status, next_retry_at, created_at)
    WHERE status IN ('pending', 'failed');
