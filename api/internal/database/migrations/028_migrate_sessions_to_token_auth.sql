-- +goose Up
DELETE FROM sessions;
ALTER TABLE sessions DROP COLUMN data;
ALTER TABLE sessions ADD COLUMN user_id TEXT NOT NULL DEFAULT '' CHECK(user_id != '');

-- +goose Down
ALTER TABLE sessions DROP COLUMN user_id;
ALTER TABLE sessions ADD COLUMN data BLOB NOT NULL DEFAULT '';
