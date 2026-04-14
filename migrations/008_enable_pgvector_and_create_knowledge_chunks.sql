CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS knowledge_chunks (
  id TEXT PRIMARY KEY,
  knowledge_base TEXT NOT NULL,
  source_id TEXT NOT NULL,
  title TEXT NOT NULL,
  aliases TEXT[] NOT NULL DEFAULT '{}',
  content TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  content_tsv TSVECTOR NOT NULL,
  embedding VECTOR(1024),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_kb ON knowledge_chunks (knowledge_base);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_tsv ON knowledge_chunks USING GIN (content_tsv);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_embedding_ivfflat
ON knowledge_chunks
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);
