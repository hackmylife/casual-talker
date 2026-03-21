-- +goose Up
ALTER TABLE turns ADD COLUMN interpreted_text TEXT;

-- +goose Down
ALTER TABLE turns DROP COLUMN IF EXISTS interpreted_text;
