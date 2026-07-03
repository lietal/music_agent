DROP INDEX IF EXISTS idx_conversations_updated_at;
ALTER TABLE conversations DROP COLUMN IF EXISTS updated_at;
