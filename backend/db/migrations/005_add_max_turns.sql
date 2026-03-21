-- +goose Up
ALTER TABLE sessions ADD COLUMN max_turns INTEGER NOT NULL DEFAULT 6;

-- +goose Down
ALTER TABLE sessions DROP COLUMN IF EXISTS max_turns;
