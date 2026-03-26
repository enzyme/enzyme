-- +goose Up
CREATE TABLE user_presence (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('online', 'away', 'dnd', 'offline')) DEFAULT 'offline',
    last_seen_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, workspace_id)
);

CREATE INDEX idx_user_presence_workspace ON user_presence(workspace_id);

-- +goose Down
DROP TABLE user_presence;
