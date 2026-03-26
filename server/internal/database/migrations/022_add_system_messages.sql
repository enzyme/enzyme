-- +goose Up
ALTER TABLE messages ADD COLUMN type TEXT NOT NULL DEFAULT 'user';
ALTER TABLE messages ADD COLUMN system_event TEXT;
CREATE INDEX idx_messages_type ON messages(channel_id, type);

-- +goose Down
DROP INDEX IF EXISTS idx_messages_type;
-- Note: SQLite with modernc/sqlite supports ALTER TABLE DROP COLUMN
ALTER TABLE messages DROP COLUMN system_event;
ALTER TABLE messages DROP COLUMN type;
