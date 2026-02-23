-- +goose Up
ALTER TABLE link_previews ADD COLUMN type TEXT NOT NULL DEFAULT 'external';
ALTER TABLE link_previews ADD COLUMN linked_message_id TEXT;
ALTER TABLE link_previews ADD COLUMN linked_channel_id TEXT;
ALTER TABLE link_previews ADD COLUMN linked_channel_name TEXT;
ALTER TABLE link_previews ADD COLUMN linked_channel_type TEXT;
ALTER TABLE link_previews ADD COLUMN message_author_id TEXT;
ALTER TABLE link_previews ADD COLUMN message_author_name TEXT;
ALTER TABLE link_previews ADD COLUMN message_author_avatar_url TEXT;
ALTER TABLE link_previews ADD COLUMN message_author_gravatar_url TEXT;
ALTER TABLE link_previews ADD COLUMN message_content TEXT;
ALTER TABLE link_previews ADD COLUMN message_created_at TEXT;

-- +goose Down
ALTER TABLE link_previews DROP COLUMN message_created_at;
ALTER TABLE link_previews DROP COLUMN message_content;
ALTER TABLE link_previews DROP COLUMN message_author_gravatar_url;
ALTER TABLE link_previews DROP COLUMN message_author_avatar_url;
ALTER TABLE link_previews DROP COLUMN message_author_name;
ALTER TABLE link_previews DROP COLUMN message_author_id;
ALTER TABLE link_previews DROP COLUMN linked_channel_type;
ALTER TABLE link_previews DROP COLUMN linked_channel_name;
ALTER TABLE link_previews DROP COLUMN linked_channel_id;
ALTER TABLE link_previews DROP COLUMN linked_message_id;
ALTER TABLE link_previews DROP COLUMN type;
