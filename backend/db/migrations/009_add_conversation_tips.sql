-- +goose Up
ALTER TABLE feedbacks ADD COLUMN conversation_tips JSONB DEFAULT '[]';

-- +goose Down
ALTER TABLE feedbacks DROP COLUMN IF EXISTS conversation_tips;
