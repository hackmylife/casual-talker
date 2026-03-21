package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/naoki-watanabe/casual-talker/backend/internal/domain"
)

// SessionRepository defines data access methods for courses, themes, sessions,
// turns, and feedback.
type SessionRepository interface {
	// Courses & Themes
	ListCourses(ctx context.Context) ([]domain.Course, error)
	ListThemesByCourse(ctx context.Context, courseID string) ([]domain.Theme, error)
	GetTheme(ctx context.Context, id string) (*domain.Theme, error)

	// Sessions
	CreateSession(ctx context.Context, userID, themeID string, difficulty, maxTurns int) (*domain.Session, error)
	GetSession(ctx context.Context, id string) (*domain.Session, error)
	ListSessionsByUser(ctx context.Context, userID string, limit, offset int) ([]domain.Session, error)
	CompleteSession(ctx context.Context, id string, turnCount int) error

	// Turns
	CreateTurn(ctx context.Context, turn *domain.Turn) (*domain.Turn, error)
	ListTurnsBySession(ctx context.Context, sessionID string) ([]domain.Turn, error)

	// Feedback
	CreateFeedback(ctx context.Context, fb *domain.Feedback) (*domain.Feedback, error)
	GetFeedbackBySession(ctx context.Context, sessionID string) (*domain.Feedback, error)
}

// PgxSessionRepository is a pgx-backed implementation of SessionRepository.
type PgxSessionRepository struct {
	pool *pgxpool.Pool
}

// NewPgxSessionRepository creates a new PgxSessionRepository with the given
// connection pool.
func NewPgxSessionRepository(pool *pgxpool.Pool) *PgxSessionRepository {
	return &PgxSessionRepository{pool: pool}
}

// --- Courses & Themes ---

// ListCourses returns all courses ordered by sort_order ascending.
func (r *PgxSessionRepository) ListCourses(ctx context.Context) ([]domain.Course, error) {
	const q = `
		SELECT id, title, description, sort_order, created_at
		FROM courses
		ORDER BY sort_order ASC`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []domain.Course
	for rows.Next() {
		var c domain.Course
		if err := rows.Scan(&c.ID, &c.Title, &c.Description, &c.SortOrder, &c.CreatedAt); err != nil {
			return nil, err
		}
		courses = append(courses, c)
	}
	return courses, rows.Err()
}

// ListThemesByCourse returns all themes for a course ordered by sort_order.
func (r *PgxSessionRepository) ListThemesByCourse(ctx context.Context, courseID string) ([]domain.Theme, error) {
	const q = `
		SELECT id, course_id, title, description,
		       target_phrases, base_vocabulary,
		       difficulty_min, difficulty_max, sort_order, created_at
		FROM themes
		WHERE course_id = $1
		ORDER BY sort_order ASC`

	rows, err := r.pool.Query(ctx, q, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var themes []domain.Theme
	for rows.Next() {
		t, err := scanTheme(rows)
		if err != nil {
			return nil, err
		}
		themes = append(themes, *t)
	}
	return themes, rows.Err()
}

// GetTheme retrieves a single theme by its UUID.
// Returns ErrNotFound if no theme exists with that ID.
func (r *PgxSessionRepository) GetTheme(ctx context.Context, id string) (*domain.Theme, error) {
	const q = `
		SELECT id, course_id, title, description,
		       target_phrases, base_vocabulary,
		       difficulty_min, difficulty_max, sort_order, created_at
		FROM themes
		WHERE id = $1`

	row := r.pool.QueryRow(ctx, q, id)
	t, err := scanTheme(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return t, nil
}

// --- Sessions ---

// CreateSession inserts a new active session with the given max_turns and returns it.
func (r *PgxSessionRepository) CreateSession(ctx context.Context, userID, themeID string, difficulty, maxTurns int) (*domain.Session, error) {
	const q = `
		INSERT INTO sessions (user_id, theme_id, difficulty, max_turns)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, theme_id, difficulty, status,
		          started_at, ended_at, turn_count, max_turns, created_at`

	row := r.pool.QueryRow(ctx, q, userID, themeID, difficulty, maxTurns)
	return scanSession(row)
}

// GetSession retrieves a session by its UUID.
// Returns ErrNotFound if no session exists with that ID.
func (r *PgxSessionRepository) GetSession(ctx context.Context, id string) (*domain.Session, error) {
	const q = `
		SELECT id, user_id, theme_id, difficulty, status,
		       started_at, ended_at, turn_count, max_turns, created_at
		FROM sessions
		WHERE id = $1`

	row := r.pool.QueryRow(ctx, q, id)
	s, err := scanSession(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s, nil
}

// ListSessionsByUser returns paginated sessions for a user, newest first.
func (r *PgxSessionRepository) ListSessionsByUser(ctx context.Context, userID string, limit, offset int) ([]domain.Session, error) {
	const q = `
		SELECT id, user_id, theme_id, difficulty, status,
		       started_at, ended_at, turn_count, max_turns, created_at
		FROM sessions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, q, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []domain.Session
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, *s)
	}
	return sessions, rows.Err()
}

// CompleteSession marks a session as completed and records the final turn count.
func (r *PgxSessionRepository) CompleteSession(ctx context.Context, id string, turnCount int) error {
	const q = `
		UPDATE sessions
		SET status = 'completed', ended_at = now(), turn_count = $2
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, q, id, turnCount)
	return err
}

// --- Turns ---

// CreateTurn inserts a new turn record and returns it with the generated ID and
// created_at timestamp filled in. interpreted_text is stored when the STT
// transcription was corrected by the interpret step.
func (r *PgxSessionRepository) CreateTurn(ctx context.Context, turn *domain.Turn) (*domain.Turn, error) {
	const q = `
		INSERT INTO turns (
			session_id, turn_number, ai_text, ai_audio_url,
			user_text, user_audio_url, interpreted_text,
			hint_used, repeat_used, ja_help_used, example_used
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, session_id, turn_number, ai_text, ai_audio_url,
		          user_text, user_audio_url, interpreted_text,
		          hint_used, repeat_used, ja_help_used, example_used, created_at`

	row := r.pool.QueryRow(ctx, q,
		turn.SessionID, turn.TurnNumber, turn.AIText, turn.AIAudioURL,
		turn.UserText, turn.UserAudioURL, turn.InterpretedText,
		turn.HintUsed, turn.RepeatUsed, turn.JaHelpUsed, turn.ExampleUsed,
	)
	return scanTurn(row)
}

// ListTurnsBySession returns all turns for a session ordered by turn_number.
func (r *PgxSessionRepository) ListTurnsBySession(ctx context.Context, sessionID string) ([]domain.Turn, error) {
	const q = `
		SELECT id, session_id, turn_number, ai_text, ai_audio_url,
		       user_text, user_audio_url, interpreted_text,
		       hint_used, repeat_used, ja_help_used, example_used, created_at
		FROM turns
		WHERE session_id = $1
		ORDER BY turn_number ASC`

	rows, err := r.pool.Query(ctx, q, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var turns []domain.Turn
	for rows.Next() {
		t, err := scanTurn(rows)
		if err != nil {
			return nil, err
		}
		turns = append(turns, *t)
	}
	return turns, rows.Err()
}

// --- Feedback ---

// CreateFeedback inserts a feedback record and returns it with the generated
// ID and created_at timestamp filled in. current_level and next_level_advice
// are included when the LLM returns them.
func (r *PgxSessionRepository) CreateFeedback(ctx context.Context, fb *domain.Feedback) (*domain.Feedback, error) {
	const q = `
		INSERT INTO feedbacks (
			session_id, achievements, natural_expressions,
			improvements, review_phrases,
			current_level, next_level_advice,
			raw_llm_response
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, session_id, achievements, natural_expressions,
		          improvements, review_phrases,
		          current_level, next_level_advice,
		          raw_llm_response, created_at`

	row := r.pool.QueryRow(ctx, q,
		fb.SessionID,
		fb.Achievements,
		fb.NaturalExpressions,
		fb.Improvements,
		fb.ReviewPhrases,
		fb.CurrentLevel,
		fb.NextLevelAdvice,
		fb.RawLLMResponse,
	)
	return scanFeedback(row)
}

// GetFeedbackBySession retrieves the feedback for a completed session.
// Returns ErrNotFound if no feedback has been generated yet.
func (r *PgxSessionRepository) GetFeedbackBySession(ctx context.Context, sessionID string) (*domain.Feedback, error) {
	const q = `
		SELECT id, session_id, achievements, natural_expressions,
		       improvements, review_phrases,
		       current_level, next_level_advice,
		       raw_llm_response, created_at
		FROM feedbacks
		WHERE session_id = $1`

	row := r.pool.QueryRow(ctx, q, sessionID)
	fb, err := scanFeedback(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return fb, nil
}

// --- scan helpers ---

// pgxRow is a common interface satisfied by both pgx.Row and pgx.Rows so that
// the scan helpers can be reused for both QueryRow and Query results.
type pgxRow interface {
	Scan(dest ...any) error
}

func scanTheme(row pgxRow) (*domain.Theme, error) {
	var t domain.Theme
	err := row.Scan(
		&t.ID, &t.CourseID, &t.Title, &t.Description,
		&t.TargetPhrases, &t.BaseVocabulary,
		&t.DifficultyMin, &t.DifficultyMax, &t.SortOrder, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func scanSession(row pgxRow) (*domain.Session, error) {
	var s domain.Session
	err := row.Scan(
		&s.ID, &s.UserID, &s.ThemeID, &s.Difficulty, &s.Status,
		&s.StartedAt, &s.EndedAt, &s.TurnCount, &s.MaxTurns, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func scanTurn(row pgxRow) (*domain.Turn, error) {
	var t domain.Turn
	err := row.Scan(
		&t.ID, &t.SessionID, &t.TurnNumber, &t.AIText, &t.AIAudioURL,
		&t.UserText, &t.UserAudioURL, &t.InterpretedText,
		&t.HintUsed, &t.RepeatUsed, &t.JaHelpUsed, &t.ExampleUsed,
		&t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func scanFeedback(row pgxRow) (*domain.Feedback, error) {
	var fb domain.Feedback
	err := row.Scan(
		&fb.ID, &fb.SessionID,
		&fb.Achievements, &fb.NaturalExpressions,
		&fb.Improvements, &fb.ReviewPhrases,
		&fb.CurrentLevel, &fb.NextLevelAdvice,
		&fb.RawLLMResponse, &fb.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &fb, nil
}
