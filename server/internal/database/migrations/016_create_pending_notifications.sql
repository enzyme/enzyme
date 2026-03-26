-- +goose Up
CREATE TABLE pending_notifications (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    message_id TEXT NOT NULL,
    notification_type TEXT NOT NULL
        CHECK (notification_type IN ('mention', 'dm', 'channel', 'here', 'everyone')),
    created_at TEXT NOT NULL,
    send_after TEXT NOT NULL,
    UNIQUE(user_id, message_id)
);

CREATE INDEX idx_pending_notifications_user_id ON pending_notifications(user_id);
CREATE INDEX idx_pending_notifications_send_after ON pending_notifications(send_after);

-- +goose Down
DROP INDEX IF EXISTS idx_pending_notifications_send_after;
DROP INDEX IF EXISTS idx_pending_notifications_user_id;
DROP TABLE IF EXISTS pending_notifications;
