-- +goose Up
CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    channel_id TEXT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    content TEXT NOT NULL,
    thread_parent_id TEXT REFERENCES messages(id) ON DELETE CASCADE,
    reply_count INTEGER NOT NULL DEFAULT 0,
    last_reply_at TEXT,
    edited_at TEXT,
    deleted_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_messages_channel ON messages(channel_id, id);
CREATE INDEX idx_messages_thread ON messages(thread_parent_id, id);

-- +goose Down
DROP TABLE messages;
