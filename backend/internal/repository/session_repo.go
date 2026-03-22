package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/naoki-watanabe/casual-talker/backend/internal/domain"
)

// SessionRepository defines data access methods for courses, themes, sessions,
// turns, and feedback.
type SessionRepository interface {
	// Courses & Themes
	ListCourses(ctx context.Context) ([]domain.Course, error)
	GetCourse(ctx context.Context, id string) (*domain.Course, error)
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

	// Past session summaries (for conversation variety)
	GetPastSessionTopics(ctx context.Context, userID, themeID string, limit int) ([]string, error)

	// Stats
	GetUserStats(ctx context.Context, userID string) (*domain.UserStats, error)
	GetUserLanguageStats(ctx context.Context, userID string) ([]domain.LanguageStat, error)
	GetUserSessionDates(ctx context.Context, userID string) ([]time.Time, error)
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
		SELECT id, title, description, target_language, sort_order, created_at
		FROM courses
		ORDER BY sort_order ASC`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []domain.Course
	for rows.Next() {
		c, err := scanCourse(rows)
		if err != nil {
			return nil, err
		}
		courses = append(courses, *c)
	}
	return courses, rows.Err()
}

// GetCourse retrieves a single course by its UUID.
// Returns ErrNotFound if no course exists with that ID.
func (r *PgxSessionRepository) GetCourse(ctx context.Context, id string) (*domain.Course, error) {
	const q = `
		SELECT id, title, description, target_language, sort_order, created_at
		FROM courses
		WHERE id = $1`

	row := r.pool.QueryRow(ctx, q, id)
	c, err := scanCourse(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return c, nil
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

// --- Stats ---

// GetUserStats aggregates total_sessions, total_practice_minutes,
// total_user_turns, and pronunciation_fixes for a user from the sessions and
// turns tables. No additional tables are required.
func (r *PgxSessionRepository) GetUserStats(ctx context.Context, userID string) (*domain.UserStats, error) {
	const q = `
		SELECT
			(SELECT COUNT(*)
			 FROM sessions
			 WHERE user_id = $1 AND status = 'completed') AS total_sessions,

			(SELECT COALESCE(
				SUM(EXTRACT(EPOCH FROM (ended_at - started_at)) / 60),
				0
			)
			 FROM sessions
			 WHERE user_id = $1 AND status = 'completed' AND ended_at IS NOT NULL)
			AS total_practice_minutes,

			(SELECT COUNT(*)
			 FROM turns t
			 JOIN sessions s ON t.session_id = s.id
			 WHERE s.user_id = $1 AND t.user_text IS NOT NULL AND t.user_text != '')
			AS total_user_turns,

			(SELECT COUNT(*)
			 FROM turns t
			 JOIN sessions s ON t.session_id = s.id
			 WHERE s.user_id = $1
			   AND t.interpreted_text IS NOT NULL
			   AND t.interpreted_text != t.user_text)
			AS pronunciation_fixes`

	var stats domain.UserStats
	var practiceMinutesF float64
	err := r.pool.QueryRow(ctx, q, userID).Scan(
		&stats.TotalSessions,
		&practiceMinutesF,
		&stats.TotalUserTurns,
		&stats.PronunciationFixes,
	)
	if err != nil {
		return nil, err
	}
	stats.TotalPracticeMinutes = int(practiceMinutesF)
	return &stats, nil
}

// GetUserLanguageStats returns per-language session counts and last practice
// timestamps for all languages the user has completed sessions in.
func (r *PgxSessionRepository) GetUserLanguageStats(ctx context.Context, userID string) ([]domain.LanguageStat, error) {
	const q = `
		SELECT c.target_language,
		       COUNT(*)       AS sessions,
		       MAX(s.ended_at) AS last_practiced
		FROM sessions s
		JOIN themes th ON s.theme_id = th.id
		JOIN courses c  ON th.course_id = c.id
		WHERE s.user_id = $1 AND s.status = 'completed'
		GROUP BY c.target_language`

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []domain.LanguageStat
	for rows.Next() {
		var ls domain.LanguageStat
		if err := rows.Scan(&ls.Language, &ls.Sessions, &ls.LastPracticed); err != nil {
			return nil, err
		}
		stats = append(stats, ls)
	}
	return stats, rows.Err()
}

// GetUserSessionDates returns the distinct calendar dates (UTC timestamps at
// midnight of Asia/Tokyo day boundaries) of all completed sessions for a user,
// ordered newest first. The streak calculation is performed on the handler side
// using the JST date components extracted in SQL.
func (r *PgxSessionRepository) GetUserSessionDates(ctx context.Context, userID string) ([]time.Time, error) {
	const q = `
		SELECT DISTINCT
			DATE_TRUNC('day', ended_at AT TIME ZONE 'Asia/Tokyo') AT TIME ZONE 'Asia/Tokyo' AS session_day
		FROM sessions
		WHERE user_id = $1 AND status = 'completed' AND ended_at IS NOT NULL
		ORDER BY session_day DESC`

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		dates = append(dates, d)
	}
	return dates, rows.Err()
}

// --- scan helpers ---

// pgxRow is a common interface satisfied by both pgx.Row and pgx.Rows so that
// the scan helpers can be reused for both QueryRow and Query results.
type pgxRow interface {
	Scan(dest ...any) error
}

func scanCourse(row pgxRow) (*domain.Course, error) {
	var c domain.Course
	err := row.Scan(&c.ID, &c.Title, &c.Description, &c.TargetLanguage, &c.SortOrder, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
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

// GetPastSessionTopics returns short summaries of the user's previous sessions
// on a given theme. Each summary is the AI's first message from the session,
// which indicates what the conversation was about. This helps the prompt avoid
// repeating the same conversation starters.
func (r *PgxSessionRepository) GetPastSessionTopics(ctx context.Context, userID, themeID string, limit int) ([]string, error) {
	query := `
		SELECT t.ai_text
		FROM turns t
		JOIN sessions s ON t.session_id = s.id
		WHERE s.user_id = $1
		  AND s.theme_id = $2
		  AND s.status = 'completed'
		  AND t.turn_number = 1
		ORDER BY s.created_at DESC
		LIMIT $3`

	rows, err := r.pool.Query(ctx, query, userID, themeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topics []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return nil, err
		}
		topics = append(topics, text)
	}
	return topics, rows.Err()
}
