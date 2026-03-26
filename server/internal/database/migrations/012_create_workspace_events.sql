-- +goose Up
CREATE TABLE workspace_events (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    payload TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_workspace_events_workspace ON workspace_events(workspace_id, id);

-- +goose Down
DROP TABLE workspace_events;
