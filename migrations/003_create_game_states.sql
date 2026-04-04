CREATE TABLE IF NOT EXISTS game_states (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL UNIQUE,
    current_scene TEXT NOT NULL,
    player_data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
