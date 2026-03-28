-- +goose Up
-- Covering index for unread message counting. The correlated subqueries in
-- GetWorkspaceNotificationSummaries filter on channel_id, thread_parent_id,
-- deleted_at, and id. This composite index lets SQLite satisfy those filters
-- with an index scan instead of a full table scan.
CREATE INDEX idx_messages_unread_scan ON messages(channel_id, thread_parent_id, deleted_at, id);

-- +goose Down
DROP INDEX IF EXISTS idx_messages_unread_scan;
