-- +goose Up
ALTER TABLE workspaces DROP COLUMN slug;

-- +goose Down
ALTER TABLE workspaces ADD COLUMN slug TEXT;
UPDATE workspaces SET slug = id;
CREATE UNIQUE INDEX idx_workspaces_slug ON workspaces(slug);
