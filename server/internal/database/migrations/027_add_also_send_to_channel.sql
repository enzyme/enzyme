-- +goose Up
ALTER TABLE messages ADD COLUMN also_send_to_channel BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE messages DROP COLUMN also_send_to_channel;
