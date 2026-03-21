-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);

-- +goose Down
DROP INDEX IF EXISTS idx_refresh_tokens_token_hash;
