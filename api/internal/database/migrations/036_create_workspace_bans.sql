-- +goose Up
CREATE TABLE workspace_bans (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    banned_by TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT,
    hide_messages INTEGER NOT NULL DEFAULT 0,
    expires_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(workspace_id, user_id)
);
CREATE INDEX idx_workspace_bans_workspace ON workspace_bans(workspace_id);
CREATE INDEX idx_workspace_bans_user ON workspace_bans(user_id);

-- +goose Down
DROP TABLE workspace_bans;
