-- +goose Up
ALTER TABLE channels ADD COLUMN is_default INTEGER NOT NULL DEFAULT 0;

-- +goose Down
-- SQLite does not support DROP COLUMN in older versions, so we recreate the table
CREATE TABLE channels_new (
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

INSERT INTO channels_new SELECT id, workspace_id, name, description, type, dm_participant_hash, archived_at, created_by, created_at, updated_at FROM channels;
DROP TABLE channels;
ALTER TABLE channels_new RENAME TO channels;

CREATE INDEX idx_channels_workspace ON channels(workspace_id);
CREATE INDEX idx_channels_dm_hash ON channels(workspace_id, dm_participant_hash);
