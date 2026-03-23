-- +goose Up
CREATE TABLE device_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    platform TEXT NOT NULL CHECK (platform IN ('fcm', 'apns')),
    device_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(user_id, token)
);
CREATE INDEX idx_device_tokens_token ON device_tokens(token);

-- +goose Down
DROP TABLE device_tokens;
