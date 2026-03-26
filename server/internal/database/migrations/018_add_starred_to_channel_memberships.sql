-- +goose Up
ALTER TABLE channel_memberships ADD COLUMN is_starred INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE channel_memberships DROP COLUMN is_starred;
