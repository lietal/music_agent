ALTER TABLE conversations ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
CREATE INDEX idx_conversations_updated_at ON conversations (user_id, updated_at DESC);
