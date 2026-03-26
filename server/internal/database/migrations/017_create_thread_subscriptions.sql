-- +goose Up
CREATE TABLE thread_subscriptions (
    id TEXT PRIMARY KEY,
    thread_parent_id TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'subscribed'
        CHECK (status IN ('subscribed', 'unsubscribed')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(thread_parent_id, user_id)
);

CREATE INDEX idx_thread_subscriptions_thread ON thread_subscriptions(thread_parent_id);
CREATE INDEX idx_thread_subscriptions_user ON thread_subscriptions(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_thread_subscriptions_user;
DROP INDEX IF EXISTS idx_thread_subscriptions_thread;
DROP TABLE IF EXISTS thread_subscriptions;
