-- +goose Up
-- information only: it blocks nothing. Current users were made by an admin or an IdP.
ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMP;
UPDATE users SET email_verified_at = NOW();

-- one-time links for an invite or a password reset. A used token is deleted.
CREATE TABLE user_tokens (
    token_hash TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    purpose TEXT NOT NULL CHECK (purpose IN ('invite', 'password_reset')),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL,
    -- a new link replaces the old one
    UNIQUE (user_id, purpose)
);

-- +goose Down
DROP TABLE user_tokens;
ALTER TABLE users DROP COLUMN email_verified_at;
