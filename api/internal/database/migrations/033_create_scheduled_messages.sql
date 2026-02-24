-- +goose Up
CREATE TABLE scheduled_messages (
    id TEXT PRIMARY KEY,
    channel_id TEXT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL DEFAULT '',
    thread_parent_id TEXT REFERENCES messages(id) ON DELETE CASCADE,
    also_send_to_channel BOOLEAN NOT NULL DEFAULT FALSE,
    attachment_ids TEXT NOT NULL DEFAULT '[]',
    scheduled_for TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX idx_scheduled_messages_user ON scheduled_messages(user_id);
CREATE INDEX idx_scheduled_messages_due ON scheduled_messages(scheduled_for);
CREATE INDEX idx_scheduled_messages_channel ON scheduled_messages(channel_id, user_id);

-- +goose Down
DROP TABLE scheduled_messages;
