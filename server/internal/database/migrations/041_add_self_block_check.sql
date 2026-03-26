-- +goose Up
-- Add CHECK constraint to prevent users from blocking themselves.
PRAGMA foreign_keys = OFF;

ALTER TABLE user_blocks RENAME TO user_blocks_old;

CREATE TABLE user_blocks (
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    blocker_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (workspace_id, blocker_id, blocked_id),
    CHECK (blocker_id != blocked_id)
);

INSERT INTO user_blocks SELECT * FROM user_blocks_old;

DROP TABLE user_blocks_old;

CREATE INDEX idx_user_blocks_workspace_blocker ON user_blocks(workspace_id, blocker_id);
CREATE INDEX idx_user_blocks_workspace_blocked ON user_blocks(workspace_id, blocked_id);

PRAGMA foreign_keys = ON;

-- +goose Down
-- Remove the CHECK constraint by recreating without it.
PRAGMA foreign_keys = OFF;

ALTER TABLE user_blocks RENAME TO user_blocks_old;

CREATE TABLE user_blocks (
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    blocker_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (workspace_id, blocker_id, blocked_id)
);

INSERT INTO user_blocks SELECT * FROM user_blocks_old;

DROP TABLE user_blocks_old;

CREATE INDEX idx_user_blocks_workspace_blocker ON user_blocks(workspace_id, blocker_id);
CREATE INDEX idx_user_blocks_workspace_blocked ON user_blocks(workspace_id, blocked_id);

PRAGMA foreign_keys = ON;
