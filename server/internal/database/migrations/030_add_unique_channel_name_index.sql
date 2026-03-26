-- +goose Up
CREATE UNIQUE INDEX idx_channels_workspace_name ON channels(workspace_id, name) WHERE type IN ('public', 'private');

-- +goose Down
DROP INDEX IF EXISTS idx_channels_workspace_name;
