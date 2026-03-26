-- +goose Up
CREATE TABLE channel_memberships (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id TEXT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    channel_role TEXT CHECK (channel_role IN ('admin', 'poster', 'viewer')),
    last_read_message_id TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, channel_id)
);

CREATE INDEX idx_channel_memberships_channel ON channel_memberships(channel_id, user_id);
CREATE INDEX idx_channel_memberships_user ON channel_memberships(user_id);

-- +goose Down
DROP TABLE channel_memberships;
