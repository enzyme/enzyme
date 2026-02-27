-- +goose Up
-- This migration converts user_blocks from global scope to workspace-scoped.
-- The old table (from migration 037) had no workspace_id column. Since blocks
-- are semantically different when workspace-scoped (they enable role gating
-- and per-workspace isolation), existing global blocks cannot be meaningfully
-- migrated. Any blocks created between migrations 037 and 039 are intentionally
-- discarded.
DROP TABLE IF EXISTS user_blocks;

CREATE TABLE user_blocks (
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    blocker_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (workspace_id, blocker_id, blocked_id)
);

CREATE INDEX idx_user_blocks_workspace_blocker ON user_blocks(workspace_id, blocker_id);
CREATE INDEX idx_user_blocks_workspace_blocked ON user_blocks(workspace_id, blocked_id);

-- +goose Down
DROP TABLE IF EXISTS user_blocks;

CREATE TABLE user_blocks (
    blocker_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (blocker_id, blocked_id)
);

CREATE INDEX idx_user_blocks_blocker ON user_blocks(blocker_id);
CREATE INDEX idx_user_blocks_blocked ON user_blocks(blocked_id);
