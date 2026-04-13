CREATE TABLE IF NOT EXISTS session_memories (
  session_id TEXT PRIMARY KEY REFERENCES sessions(id) ON DELETE CASCADE,
  character_summary TEXT NOT NULL DEFAULT '',
  scene_summary TEXT NOT NULL DEFAULT '',
  current_objective TEXT NOT NULL DEFAULT '',
  recent_key_events JSONB NOT NULL DEFAULT '[]'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL
);
