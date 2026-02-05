-- +goose Up
ALTER TABLE workspace_memberships ADD COLUMN sort_order INTEGER;
CREATE INDEX idx_workspace_memberships_sort ON workspace_memberships(user_id, sort_order);

-- +goose Down
DROP INDEX IF EXISTS idx_workspace_memberships_sort;
ALTER TABLE workspace_memberships DROP COLUMN sort_order;
