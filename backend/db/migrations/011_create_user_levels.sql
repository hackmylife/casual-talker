-- +goose Up
CREATE TABLE user_levels (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id),
    language   TEXT NOT NULL,
    level      INTEGER NOT NULL DEFAULT 1 CHECK (level BETWEEN 1 AND 5),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, language)
);

-- Migrate existing user levels (level > 1) to the new table, defaulting to English.
INSERT INTO user_levels (user_id, language, level)
SELECT id, 'en', level FROM users WHERE level > 1;

-- +goose Down
DROP TABLE IF EXISTS user_levels;
