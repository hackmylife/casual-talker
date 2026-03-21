-- +goose Up
ALTER TABLE feedbacks ADD COLUMN current_level JSONB DEFAULT '{}';
ALTER TABLE feedbacks ADD COLUMN next_level_advice TEXT DEFAULT '';

-- +goose Down
ALTER TABLE feedbacks DROP COLUMN IF EXISTS current_level;
ALTER TABLE feedbacks DROP COLUMN IF EXISTS next_level_advice;
