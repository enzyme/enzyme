-- +goose Up
CREATE TABLE notification_preferences (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id TEXT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    notify_level TEXT NOT NULL DEFAULT 'mentions'
        CHECK (notify_level IN ('all', 'mentions', 'none')),
    email_enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(user_id, channel_id)
);

CREATE INDEX idx_notification_preferences_user_id ON notification_preferences(user_id);
CREATE INDEX idx_notification_preferences_channel_id ON notification_preferences(channel_id);

-- +goose Down
DROP INDEX IF EXISTS idx_notification_preferences_channel_id;
DROP INDEX IF EXISTS idx_notification_preferences_user_id;
DROP TABLE IF EXISTS notification_preferences;
