-- +goose Up
CREATE TABLE sessions (
    token TEXT PRIMARY KEY,
    data BLOB NOT NULL,
    expiry TEXT NOT NULL
);

CREATE INDEX idx_sessions_expiry ON sessions(expiry);

-- +goose Down
DROP TABLE sessions;
