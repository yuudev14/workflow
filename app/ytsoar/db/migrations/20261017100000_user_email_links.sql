-- +goose Up
-- information only: it blocks nothing. Current users were made by an admin or an IdP.
ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMP;
UPDATE users SET email_verified_at = NOW();

-- one-time email links, as Devise does it. The tokens are sha256 hashes. Use clears them.
ALTER TABLE users
    ADD COLUMN invitation_token TEXT UNIQUE,
    ADD COLUMN invitation_sent_at TIMESTAMP,
    ADD COLUMN reset_password_token TEXT UNIQUE,
    ADD COLUMN reset_password_sent_at TIMESTAMP;

-- +goose Down
ALTER TABLE users
    DROP COLUMN reset_password_sent_at,
    DROP COLUMN reset_password_token,
    DROP COLUMN invitation_sent_at,
    DROP COLUMN invitation_token,
    DROP COLUMN email_verified_at;
