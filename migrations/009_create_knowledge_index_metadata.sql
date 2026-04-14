CREATE TABLE IF NOT EXISTS knowledge_index_metadata (
  knowledge_base TEXT PRIMARY KEY,
  embedding_provider TEXT NOT NULL,
  embedding_model TEXT NOT NULL,
  embedding_dim INTEGER NOT NULL,
  built_at TIMESTAMPTZ NOT NULL
);
