-- +goose Up
CREATE TABLE email_verifications (
    token TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_email_verifications_user_id ON email_verifications(user_id);

-- +goose Down
DROP TABLE email_verifications;
