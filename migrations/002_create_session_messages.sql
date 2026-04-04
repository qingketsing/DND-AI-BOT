CREATE TABLE IF NOT EXISTS session_messages (
    id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    user_name TEXT NOT NULL,
    content TEXT NOT NULL,
    sequence BIGINT NOT NULL,
    source TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (session_id, sequence),
    CONSTRAINT fk_session_messages_session
        FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);
