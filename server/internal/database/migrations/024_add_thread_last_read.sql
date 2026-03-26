-- +goose Up
ALTER TABLE thread_subscriptions ADD COLUMN last_read_reply_id TEXT;

-- +goose Down
ALTER TABLE thread_subscriptions DROP COLUMN last_read_reply_id;
