-- +goose Up
CREATE TABLE moderation_log (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    actor_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action TEXT NOT NULL CHECK (action IN (
        'user.banned', 'user.unbanned',
        'message.deleted', 'member.removed',
        'member.role_changed', 'channel.archived'
    )),
    target_type TEXT NOT NULL CHECK (target_type IN ('user', 'message', 'channel')),
    target_id TEXT NOT NULL,
    metadata TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX idx_moderation_log_workspace ON moderation_log(workspace_id, created_at);

-- +goose Down
DROP TABLE moderation_log;
