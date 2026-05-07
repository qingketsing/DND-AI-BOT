ALTER TABLE session_messages
    ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS reply_to_message_id TEXT,
    ADD COLUMN IF NOT EXISTS source_job_id TEXT;

UPDATE session_messages
SET role = CASE source
    WHEN 'user' THEN 'user'
    WHEN 'agent' THEN 'assistant'
    WHEN 'system' THEN 'system'
    ELSE source
END
WHERE role = '';

CREATE UNIQUE INDEX IF NOT EXISTS uq_session_messages_assistant_reply_to_message_id
    ON session_messages(reply_to_message_id)
    WHERE role = 'assistant' AND reply_to_message_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_session_messages_assistant_source_job_id
    ON session_messages(source_job_id)
    WHERE role = 'assistant' AND source_job_id IS NOT NULL;
