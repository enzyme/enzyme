-- +goose Up
ALTER TABLE messages ADD COLUMN mentions TEXT NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE messages DROP COLUMN mentions;
