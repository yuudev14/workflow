-- +goose Up
-- information only: it blocks nothing. Current users were made by an admin or an IdP.
ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMP;
UPDATE users SET email_verified_at = NOW();

-- one open link for an invite or a password reset. Use clears it, and a new link replaces it.
ALTER TABLE users
    ADD COLUMN link_token_hash TEXT UNIQUE,
    ADD COLUMN link_purpose TEXT CHECK (link_purpose IN ('invite', 'password_reset')),
    ADD COLUMN link_sent_at TIMESTAMP;

-- +goose Down
ALTER TABLE users
    DROP COLUMN link_sent_at,
    DROP COLUMN link_purpose,
    DROP COLUMN link_token_hash,
    DROP COLUMN email_verified_at;
