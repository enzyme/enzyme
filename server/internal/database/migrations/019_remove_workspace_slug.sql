-- +goose Up
-- Cannot use ALTER TABLE DROP COLUMN because slug has UNIQUE constraint
-- Must recreate table instead
PRAGMA foreign_keys = OFF;

DROP INDEX IF EXISTS idx_workspaces_slug;

CREATE TABLE workspaces_new (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    icon_url TEXT,
    settings TEXT DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO workspaces_new (id, name, icon_url, settings, created_at, updated_at)
SELECT id, name, icon_url, settings, created_at, updated_at FROM workspaces;

DROP TABLE workspaces;
ALTER TABLE workspaces_new RENAME TO workspaces;

PRAGMA foreign_keys = ON;

-- +goose Down
ALTER TABLE workspaces ADD COLUMN slug TEXT;
UPDATE workspaces SET slug = id;
CREATE UNIQUE INDEX idx_workspaces_slug ON workspaces(slug);
