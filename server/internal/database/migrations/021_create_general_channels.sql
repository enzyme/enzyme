-- +goose Up
-- Create #general channel for existing workspaces that don't have one
-- This is idempotent - it won't create duplicates

-- For each workspace that doesn't have a default channel, create one
-- Note: We use a random-ish ID based on workspace_id hash since we can't generate ULIDs in SQL
INSERT INTO channels (id, workspace_id, name, type, is_default, created_at, updated_at)
SELECT
    'GEN-' || substr(w.id, 1, 22),
    w.id,
    'general',
    'public',
    1,
    datetime('now'),
    datetime('now')
FROM workspaces w
WHERE NOT EXISTS (
    SELECT 1 FROM channels c
    WHERE c.workspace_id = w.id AND c.is_default = 1
);

-- Add all workspace members to the new #general channels
INSERT INTO channel_memberships (id, user_id, channel_id, channel_role, created_at, updated_at)
SELECT
    'GM-' || wm.id,
    wm.user_id,
    c.id,
    CASE WHEN wm.role = 'owner' THEN 'admin' ELSE 'poster' END,
    datetime('now'),
    datetime('now')
FROM workspace_memberships wm
JOIN channels c ON c.workspace_id = wm.workspace_id AND c.is_default = 1
WHERE NOT EXISTS (
    SELECT 1 FROM channel_memberships cm
    WHERE cm.user_id = wm.user_id AND cm.channel_id = c.id
);

-- +goose Down
-- Remove channel memberships for default channels created by this migration
DELETE FROM channel_memberships WHERE channel_id IN (
    SELECT id FROM channels WHERE is_default = 1 AND id LIKE 'GEN-%'
);

-- Remove default channels created by this migration
DELETE FROM channels WHERE is_default = 1 AND id LIKE 'GEN-%';
