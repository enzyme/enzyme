-- +goose Up
ALTER TABLE messages ADD COLUMN pinned_at TEXT;
ALTER TABLE messages ADD COLUMN pinned_by TEXT REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX idx_messages_pinned ON messages(channel_id, pinned_at) WHERE pinned_at IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_messages_pinned;
ALTER TABLE messages DROP COLUMN pinned_by;
ALTER TABLE messages DROP COLUMN pinned_at;
