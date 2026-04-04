CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    channel TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
