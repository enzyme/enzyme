-- +goose Up
ALTER TABLE scheduled_messages ADD COLUMN status TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE scheduled_messages ADD COLUMN retry_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE scheduled_messages ADD COLUMN last_error TEXT;
CREATE INDEX idx_scheduled_messages_status_scheduled_for ON scheduled_messages(status, scheduled_for);

-- +goose Down
DROP INDEX IF EXISTS idx_scheduled_messages_status_scheduled_for;
ALTER TABLE scheduled_messages DROP COLUMN status;
ALTER TABLE scheduled_messages DROP COLUMN retry_count;
ALTER TABLE scheduled_messages DROP COLUMN last_error;
