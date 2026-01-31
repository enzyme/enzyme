-- +goose Up
CREATE TABLE channels (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL CHECK (type IN ('public', 'private', 'dm', 'group_dm')),
    dm_participant_hash TEXT,
    archived_at TEXT,
    created_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_channels_workspace ON channels(workspace_id);
CREATE INDEX idx_channels_dm_hash ON channels(workspace_id, dm_participant_hash);

-- +goose Down
DROP TABLE channels;
