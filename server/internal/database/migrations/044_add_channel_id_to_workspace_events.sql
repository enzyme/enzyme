-- +goose Up
ALTER TABLE workspace_events ADD COLUMN channel_id TEXT REFERENCES channels(id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE workspace_events DROP COLUMN channel_id;
