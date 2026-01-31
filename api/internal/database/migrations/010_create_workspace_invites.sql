-- +goose Up
CREATE TABLE workspace_invites (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    code TEXT UNIQUE NOT NULL,
    invited_email TEXT,
    role TEXT NOT NULL CHECK (role IN ('admin', 'member', 'guest')) DEFAULT 'member',
    created_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    max_uses INTEGER,
    use_count INTEGER NOT NULL DEFAULT 0,
    expires_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_workspace_invites_code ON workspace_invites(code);
CREATE INDEX idx_workspace_invites_workspace ON workspace_invites(workspace_id);

-- +goose Down
DROP TABLE workspace_invites;
