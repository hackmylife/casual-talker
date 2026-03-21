-- name: UpsertPhraseProgress :one
INSERT INTO phrase_progress (user_id, phrase, translation_ja, times_used, times_struggled, last_used_at)
VALUES ($1, $2, $3, $4, $5, now())
ON CONFLICT (user_id, phrase)
DO UPDATE SET
    times_used = phrase_progress.times_used + $4,
    times_struggled = phrase_progress.times_struggled + $5,
    last_used_at = now()
RETURNING *;

-- name: ListPhraseProgressByUser :many
SELECT * FROM phrase_progress WHERE user_id = $1 ORDER BY last_used_at DESC LIMIT $2 OFFSET $3;

-- name: ListWeakPhrasesByUser :many
SELECT * FROM phrase_progress WHERE user_id = $1 AND times_struggled > 0 ORDER BY times_struggled DESC LIMIT $2;
