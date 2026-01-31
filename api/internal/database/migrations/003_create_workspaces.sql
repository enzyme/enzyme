-- +goose Up
CREATE TABLE workspaces (
    id TEXT PRIMARY KEY,
    slug TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    icon_url TEXT,
    settings TEXT DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_workspaces_slug ON workspaces(slug);

-- +goose Down
DROP TABLE workspaces;
