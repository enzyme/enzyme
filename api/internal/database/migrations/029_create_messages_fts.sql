-- +goose Up

-- FTS5 virtual table for full-text search on messages
-- Uses external content approach (content="" with content_rowid) to avoid duplicating data
CREATE VIRTUAL TABLE messages_fts USING fts5(
    content,
    content='messages',
    content_rowid='rowid',
    tokenize='porter unicode61 remove_diacritics 2'
);

-- Triggers to keep FTS index in sync with messages table
-- +goose StatementBegin
CREATE TRIGGER messages_fts_insert AFTER INSERT ON messages BEGIN
    INSERT INTO messages_fts(rowid, content) VALUES (NEW.rowid, NEW.content);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER messages_fts_delete AFTER DELETE ON messages BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, content) VALUES ('delete', OLD.rowid, OLD.content);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER messages_fts_update AFTER UPDATE OF content ON messages BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, content) VALUES ('delete', OLD.rowid, OLD.content);
    INSERT INTO messages_fts(rowid, content) VALUES (NEW.rowid, NEW.content);
END;
-- +goose StatementEnd

-- Backfill existing non-deleted, non-system messages
INSERT INTO messages_fts(rowid, content)
SELECT rowid, content FROM messages
WHERE deleted_at IS NULL AND type != 'system';

-- +goose Down
DROP TRIGGER IF EXISTS messages_fts_insert;
DROP TRIGGER IF EXISTS messages_fts_delete;
DROP TRIGGER IF EXISTS messages_fts_update;
DROP TABLE IF EXISTS messages_fts;
