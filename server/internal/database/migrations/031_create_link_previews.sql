-- +goose Up

-- URL-level cache (shared across messages, keyed on URL)
CREATE TABLE link_preview_cache (
    url TEXT PRIMARY KEY,
    title TEXT,
    description TEXT,
    image_url TEXT,
    site_name TEXT,
    fetched_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    fetch_error TEXT
);

-- Per-message preview (max 1 per message via UNIQUE constraint)
CREATE TABLE link_previews (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    title TEXT,
    description TEXT,
    image_url TEXT,
    site_name TEXT,
    created_at TEXT NOT NULL,
    UNIQUE(message_id)
);
CREATE INDEX idx_link_previews_message ON link_previews(message_id);

-- +goose Down
DROP INDEX IF EXISTS idx_link_previews_message;
DROP TABLE IF EXISTS link_previews;
DROP TABLE IF EXISTS link_preview_cache;
