-- name: CreateSession :one
INSERT INTO sessions (user_id, theme_id, difficulty)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetSession :one
SELECT * FROM sessions WHERE id = $1;

-- name: ListSessionsByUser :many
SELECT * FROM sessions WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: CompleteSession :exec
UPDATE sessions SET status = 'completed', ended_at = now(), turn_count = $2 WHERE id = $1;

-- name: AbandonSession :exec
UPDATE sessions SET status = 'abandoned', ended_at = now() WHERE id = $1;

-- name: CreateTurn :one
INSERT INTO turns (session_id, turn_number, ai_text, user_text, hint_used, repeat_used, ja_help_used, example_used)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListTurnsBySession :many
SELECT * FROM turns WHERE session_id = $1 ORDER BY turn_number ASC;

-- name: CreateFeedback :one
INSERT INTO feedbacks (session_id, achievements, natural_expressions, improvements, review_phrases, raw_llm_response)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetFeedbackBySession :one
SELECT * FROM feedbacks WHERE session_id = $1;
