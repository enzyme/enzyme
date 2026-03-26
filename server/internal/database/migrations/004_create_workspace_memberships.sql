-- +goose Up
CREATE TABLE workspace_memberships (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member', 'guest')),
    display_name_override TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, workspace_id)
);

CREATE INDEX idx_workspace_memberships_workspace ON workspace_memberships(workspace_id, user_id);
CREATE INDEX idx_workspace_memberships_user ON workspace_memberships(user_id);

-- +goose Down
DROP TABLE workspace_memberships;
