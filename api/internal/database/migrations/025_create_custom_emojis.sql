-- +goose Up
CREATE TABLE custom_emojis (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_by TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content_type TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    storage_path TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE UNIQUE INDEX idx_custom_emojis_workspace_name ON custom_emojis(workspace_id, name);
CREATE INDEX idx_custom_emojis_workspace ON custom_emojis(workspace_id);

-- +goose Down
DROP TABLE custom_emojis;
