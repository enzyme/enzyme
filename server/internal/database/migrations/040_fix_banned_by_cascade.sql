-- +goose Up
-- Fix: change banned_by from ON DELETE CASCADE to ON DELETE SET NULL
-- so that deleting the banning admin does not silently remove ban records.
PRAGMA foreign_keys = OFF;

ALTER TABLE workspace_bans RENAME TO workspace_bans_old;

CREATE TABLE workspace_bans (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    banned_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    reason TEXT,
    hide_messages INTEGER NOT NULL DEFAULT 0,
    expires_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(workspace_id, user_id)
);

INSERT INTO workspace_bans SELECT * FROM workspace_bans_old;

DROP TABLE workspace_bans_old;

CREATE INDEX idx_workspace_bans_workspace ON workspace_bans(workspace_id);
CREATE INDEX idx_workspace_bans_user ON workspace_bans(user_id);

PRAGMA foreign_keys = ON;

-- +goose Down
-- Revert: restore banned_by as NOT NULL with ON DELETE CASCADE
PRAGMA foreign_keys = OFF;

ALTER TABLE workspace_bans RENAME TO workspace_bans_old;

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

INSERT INTO workspace_bans SELECT * FROM workspace_bans_old;

DROP TABLE workspace_bans_old;

CREATE INDEX idx_workspace_bans_workspace ON workspace_bans(workspace_id);
CREATE INDEX idx_workspace_bans_user ON workspace_bans(user_id);

PRAGMA foreign_keys = ON;
