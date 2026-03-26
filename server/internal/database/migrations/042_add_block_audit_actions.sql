-- +goose Up
-- Add 'user.blocked' and 'user.unblocked' to moderation_log action CHECK constraint
PRAGMA foreign_keys = OFF;

ALTER TABLE moderation_log RENAME TO moderation_log_old;

CREATE TABLE moderation_log (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    actor_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action TEXT NOT NULL CHECK (action IN (
        'user.banned', 'user.unbanned',
        'user.blocked', 'user.unblocked',
        'message.deleted', 'member.removed',
        'member.role_changed', 'channel.archived'
    )),
    target_type TEXT NOT NULL CHECK (target_type IN ('user', 'message', 'channel')),
    target_id TEXT NOT NULL,
    metadata TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

INSERT INTO moderation_log SELECT * FROM moderation_log_old;

DROP TABLE moderation_log_old;

CREATE INDEX idx_moderation_log_workspace ON moderation_log(workspace_id, created_at);

PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;

ALTER TABLE moderation_log RENAME TO moderation_log_old;

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

INSERT INTO moderation_log SELECT * FROM moderation_log_old;

DROP TABLE moderation_log_old;

CREATE INDEX idx_moderation_log_workspace ON moderation_log(workspace_id, created_at);

PRAGMA foreign_keys = ON;
