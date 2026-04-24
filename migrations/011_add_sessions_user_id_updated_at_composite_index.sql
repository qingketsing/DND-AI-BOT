CREATE INDEX IF NOT EXISTS idx_sessions_user_id_updated_at_desc
    ON sessions(user_id, updated_at DESC);
