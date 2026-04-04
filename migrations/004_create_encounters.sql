CREATE TABLE IF NOT EXISTS encounters (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL UNIQUE,
    round INT NOT NULL,
    turn_index INT NOT NULL,
    encounter_data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
