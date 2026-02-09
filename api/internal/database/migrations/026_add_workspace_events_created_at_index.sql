-- +goose Up
CREATE INDEX idx_workspace_events_created_at ON workspace_events(created_at);

-- +goose Down
DROP INDEX idx_workspace_events_created_at;
