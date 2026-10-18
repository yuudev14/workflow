-- +goose Up
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    -- sha256 of the key, so a database leak gives no usable key
    key_hash TEXT NOT NULL UNIQUE,
    -- the start of the key, so people can tell keys apart in the list
    prefix TEXT NOT NULL,
    created_by UUID REFERENCES users (id) ON DELETE SET NULL,
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX api_keys_user_id_idx ON api_keys (user_id);

-- +goose Down
DROP TABLE api_keys;
