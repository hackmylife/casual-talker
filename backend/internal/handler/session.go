package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/naoki-watanabe/casual-talker/backend/internal/domain"
	"github.com/naoki-watanabe/casual-talker/backend/internal/middleware"
	oai "github.com/naoki-watanabe/casual-talker/backend/internal/openai"
	"github.com/naoki-watanabe/casual-talker/backend/internal/repository"
)

// SessionHandler exposes HTTP endpoints for courses, themes, sessions, turns,
// and feedback.
type SessionHandler struct {
	repo      repository.SessionRepository
	authRepo  repository.AuthRepository
	oaiClient *oai.Client
}

// NewSessionHandler creates a new SessionHandler.
func NewSessionHandler(repo repository.SessionRepository, authRepo repository.AuthRepository, oaiClient *oai.Client) *SessionHandler {
	return &SessionHandler{repo: repo, authRepo: authRepo, oaiClient: oaiClient}
}

// maxTurnsForLevel returns the number of conversation turns appropriate for
// the user's current level. Lower levels get fewer turns to reduce cognitive
// load; higher levels get more turns to allow richer conversation practice.
func maxTurnsForLevel(level int) int {
	switch level {
	case 1:
		return 6
	case 2:
		return 8
	case 3:
		return 12
	case 4:
		return 16
	case 5:
		return 20
	default:
		return 6
	}
}

// --- request / response types ---

type createSessionRequest struct {
	ThemeID    string `json:"theme_id"   validate:"required"`
	Difficulty int    `json:"difficulty" validate:"required,min=1,max=5"`
}

type completeSessionRequest struct {
	TurnCount int `json:"turn_count" validate:"min=0"`
}

// completeSessionResponse is the body returned by PUT /sessions/{id}/complete.
// It always includes the updated session; feedback is included when generation
// succeeds and nil otherwise (the client can fall back to GET .../feedback).
type completeSessionResponse struct {
	Session  sessionResponse   `json:"session"`
	Feedback *feedbackResponse `json:"feedback,omitempty"`
}

// --- response mappers ---

type courseResponse struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Description    *string `json:"description"`
	TargetLanguage string  `json:"target_language"`
	SortOrder      int     `json:"sort_order"`
}

type themeResponse struct {
	ID             string          `json:"id"`
	CourseID       string          `json:"course_id"`
	Title          string          `json:"title"`
	Description    *string         `json:"description"`
	TargetPhrases  domain.RawJSON  `json:"target_phrases"`
	BaseVocabulary domain.RawJSON  `json:"base_vocabulary"`
	DifficultyMin  int             `json:"difficulty_min"`
	DifficultyMax  int             `json:"difficulty_max"`
	SortOrder      int             `json:"sort_order"`
}

type sessionResponse struct {
	ID         string  `json:"id"`
	UserID     string  `json:"user_id"`
	ThemeID    string  `json:"theme_id"`
	Difficulty int     `json:"difficulty"`
	Status     string  `json:"status"`
	TurnCount  int     `json:"turn_count"`
	MaxTurns   int     `json:"max_turns"`
	StartedAt  string  `json:"started_at"`
	EndedAt    *string `json:"ended_at"`
}

type turnResponse struct {
	ID              string  `json:"id"`
	SessionID       string  `json:"session_id"`
	TurnNumber      int     `json:"turn_number"`
	AIText          string  `json:"ai_text"`
	AIAudioURL      *string `json:"ai_audio_url"`
	UserText        *string `json:"user_text"`
	UserAudioURL    *string `json:"user_audio_url"`
	InterpretedText *string `json:"interpreted_text"`
	HintUsed        bool    `json:"hint_used"`
	RepeatUsed      bool    `json:"repeat_used"`
	JaHelpUsed      bool    `json:"ja_help_used"`
	ExampleUsed     bool    `json:"example_used"`
	CreatedAt       string  `json:"created_at"`
}

type feedbackResponse struct {
	ID                 string         `json:"id"`
	SessionID          string         `json:"session_id"`
	Achievements       domain.RawJSON `json:"achievements"`
	NaturalExpressions domain.RawJSON `json:"natural_expressions"`
	Improvements       domain.RawJSON `json:"improvements"`
	ConversationTips   domain.RawJSON `json:"conversation_tips"`
	ReviewPhrases      domain.RawJSON `json:"review_phrases"`
	CurrentLevel       domain.RawJSON `json:"current_level"`
	NextLevelAdvice    *string        `json:"next_level_advice"`
	CreatedAt          string         `json:"created_at"`
}

// --- conversion helpers ---

func toCourseResponse(c domain.Course) courseResponse {
	return courseResponse{
		ID:             c.ID,
		Title:          c.Title,
		Description:    c.Description,
		TargetLanguage: c.TargetLanguage,
		SortOrder:      c.SortOrder,
	}
}

func toThemeResponse(t domain.Theme) themeResponse {
	return themeResponse{
		ID:             t.ID,
		CourseID:       t.CourseID,
		Title:          t.Title,
		Description:    t.Description,
		TargetPhrases:  domain.RawJSON(t.TargetPhrases),
		BaseVocabulary: domain.RawJSON(t.BaseVocabulary),
		DifficultyMin:  t.DifficultyMin,
		DifficultyMax:  t.DifficultyMax,
		SortOrder:      t.SortOrder,
	}
}

func toSessionResponse(s domain.Session) sessionResponse {
	var endedAt *string
	if s.EndedAt != nil {
		ts := s.EndedAt.Format("2006-01-02T15:04:05Z07:00")
		endedAt = &ts
	}
	return sessionResponse{
		ID:         s.ID,
		UserID:     s.UserID,
		ThemeID:    s.ThemeID,
		Difficulty: s.Difficulty,
		Status:     s.Status,
		TurnCount:  s.TurnCount,
		MaxTurns:   s.MaxTurns,
		StartedAt:  s.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		EndedAt:    endedAt,
	}
}

func toTurnResponse(t domain.Turn) turnResponse {
	return turnResponse{
		ID:              t.ID,
		SessionID:       t.SessionID,
		TurnNumber:      t.TurnNumber,
		AIText:          t.AIText,
		AIAudioURL:      t.AIAudioURL,
		UserText:        t.UserText,
		UserAudioURL:    t.UserAudioURL,
		InterpretedText: t.InterpretedText,
		HintUsed:        t.HintUsed,
		RepeatUsed:      t.RepeatUsed,
		JaHelpUsed:      t.JaHelpUsed,
		ExampleUsed:     t.ExampleUsed,
		CreatedAt:       t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toFeedbackResponse(fb domain.Feedback) feedbackResponse {
	return feedbackResponse{
		ID:                 fb.ID,
		SessionID:          fb.SessionID,
		Achievements:       domain.RawJSON(fb.Achievements),
		NaturalExpressions: domain.RawJSON(fb.NaturalExpressions),
		Improvements:       domain.RawJSON(fb.Improvements),
		ConversationTips:   domain.RawJSON(fb.ConversationTips),
		ReviewPhrases:      domain.RawJSON(fb.ReviewPhrases),
		CurrentLevel:       domain.RawJSON(fb.CurrentLevel),
		NextLevelAdvice:    fb.NextLevelAdvice,
		CreatedAt:          fb.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// --- handlers ---

// ListCourses handles GET /api/v1/courses.
func (h *SessionHandler) ListCourses(w http.ResponseWriter, r *http.Request) {
	courses, err := h.repo.ListCourses(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	resp := make([]courseResponse, 0, len(courses))
	for _, c := range courses {
		resp = append(resp, toCourseResponse(c))
	}
	writeJSON(w, http.StatusOK, resp)
}

// ListThemes handles GET /api/v1/courses/{courseID}/themes.
func (h *SessionHandler) ListThemes(w http.ResponseWriter, r *http.Request) {
	courseID := chi.URLParam(r, "courseID")

	themes, err := h.repo.ListThemesByCourse(r.Context(), courseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	resp := make([]themeResponse, 0, len(themes))
	for _, t := range themes {
		resp = append(resp, toThemeResponse(t))
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetTheme handles GET /api/v1/themes/{id}.
func (h *SessionHandler) GetTheme(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	theme, err := h.repo.GetTheme(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "theme not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, toThemeResponse(*theme))
}

// Create handles POST /api/v1/sessions.
//
// It looks up the user's current level to determine max_turns for the session,
// so that lower-level learners face shorter sessions and higher-level learners
// get more turns for richer practice.
func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createSessionRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Fetch the user's level to compute the appropriate number of turns.
	// On failure fall back to the default (level 1) rather than aborting.
	userLevel := 1
	if h.authRepo != nil {
		if u, err := h.authRepo.GetUserByID(r.Context(), userID); err == nil {
			userLevel = u.Level
		} else {
			slog.Warn("failed to fetch user level; using default", "user_id", userID, "error", err)
		}
	}
	mt := maxTurnsForLevel(userLevel)

	session, err := h.repo.CreateSession(r.Context(), userID, req.ThemeID, req.Difficulty, mt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, toSessionResponse(*session))
}

// List handles GET /api/v1/sessions.
// Supports query parameters: limit (default 20, max 100) and offset (default 0).
func (h *SessionHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit, offset := parsePagination(r)

	sessions, err := h.repo.ListSessionsByUser(r.Context(), userID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	resp := make([]sessionResponse, 0, len(sessions))
	for _, s := range sessions {
		resp = append(resp, toSessionResponse(s))
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get handles GET /api/v1/sessions/{id}.
func (h *SessionHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")

	session, err := h.repo.GetSession(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if session.UserID != userID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	writeJSON(w, http.StatusOK, toSessionResponse(*session))
}

// Complete handles PUT /api/v1/sessions/{id}/complete.
//
// It marks the session as completed, then synchronously generates feedback via
// the LLM and persists it. The response body includes both the updated session
// and the generated feedback so the client does not need a second request.
// If feedback generation fails the session is still completed; the feedback
// field is omitted from the response and the client can retry via
// GET /sessions/{id}/feedback later.
func (h *SessionHandler) Complete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")

	session, err := h.repo.GetSession(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if session.UserID != userID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req completeSessionRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Mark the session as completed and record the final turn count.
	if err := h.repo.CompleteSession(r.Context(), id, req.TurnCount); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Reload the session so the response reflects the updated status and
	// ended_at timestamp written by CompleteSession.
	updated, err := h.repo.GetSession(r.Context(), id)
	if err != nil {
		// The session was completed successfully; return what we have.
		writeJSON(w, http.StatusOK, completeSessionResponse{Session: toSessionResponse(*session)})
		return
	}

	resp := completeSessionResponse{Session: toSessionResponse(*updated)}

	// Generate feedback synchronously. A failure here is non-fatal: the
	// session completion already succeeded so we log and continue.
	if h.oaiClient != nil {
		fb, fbErr := generateFeedback(r.Context(), h.repo, h.oaiClient, id)
		if fbErr != nil {
			slog.Error("feedback generation failed after session complete",
				"session_id", id,
				"error", fbErr,
			)
		} else {
			fbResp := toFeedbackResponse(*fb)
			resp.Feedback = &fbResp
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// ListTurns handles GET /api/v1/sessions/{id}/turns.
func (h *SessionHandler) ListTurns(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")

	session, err := h.repo.GetSession(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if session.UserID != userID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	turns, err := h.repo.ListTurnsBySession(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	resp := make([]turnResponse, 0, len(turns))
	for _, t := range turns {
		resp = append(resp, toTurnResponse(t))
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetFeedback handles GET /api/v1/sessions/{id}/feedback.
func (h *SessionHandler) GetFeedback(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")

	session, err := h.repo.GetSession(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if session.UserID != userID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	fb, err := h.repo.GetFeedbackBySession(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "feedback not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, toFeedbackResponse(*fb))
}

// statsResponse is the body returned by GET /users/me/stats.
type statsResponse struct {
	TotalSessions        int                       `json:"total_sessions"`
	TotalPracticeMinutes int                       `json:"total_practice_minutes"`
	TotalUserTurns       int                       `json:"total_user_turns"`
	CurrentStreak        int                       `json:"current_streak"`
	PronunciationFixes   int                       `json:"pronunciation_fixes"`
	Languages            map[string]languageStats  `json:"languages"`
}

// languageStats holds per-language statistics within statsResponse.
type languageStats struct {
	Sessions      int        `json:"sessions"`
	LastPracticed *time.Time `json:"last_practiced"`
}

// Stats handles GET /api/v1/users/me/stats.
// It aggregates practice statistics from the sessions and turns tables without
// reading or writing any additional tables.
func (h *SessionHandler) Stats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ctx := r.Context()

	// Aggregate base stats (total sessions, minutes, turns, pronunciation fixes).
	stats, err := h.repo.GetUserStats(ctx, userID)
	if err != nil {
		slog.Error("failed to fetch user stats", "user_id", userID, "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Per-language breakdown.
	langStats, err := h.repo.GetUserLanguageStats(ctx, userID)
	if err != nil {
		slog.Error("failed to fetch user language stats", "user_id", userID, "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Session dates for streak calculation (JST day boundaries, newest first).
	sessionDates, err := h.repo.GetUserSessionDates(ctx, userID)
	if err != nil {
		slog.Error("failed to fetch user session dates", "user_id", userID, "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	streak := calculateStreak(sessionDates)

	// Build the languages map.
	languages := make(map[string]languageStats, len(langStats))
	for _, ls := range langStats {
		languages[ls.Language] = languageStats{
			Sessions:      ls.Sessions,
			LastPracticed: ls.LastPracticed,
		}
	}

	resp := statsResponse{
		TotalSessions:        stats.TotalSessions,
		TotalPracticeMinutes: stats.TotalPracticeMinutes,
		TotalUserTurns:       stats.TotalUserTurns,
		CurrentStreak:        streak,
		PronunciationFixes:   stats.PronunciationFixes,
		Languages:            languages,
	}

	writeJSON(w, http.StatusOK, resp)
}

// calculateStreak computes the number of consecutive days the user practiced,
// counting back from today (JST). If the user has not yet practiced today the
// streak is counted from yesterday. Returns 0 when the dates slice is empty.
func calculateStreak(dates []time.Time) int {
	if len(dates) == 0 {
		return 0
	}

	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		// Fall back to UTC+9 fixed offset when the timezone database is absent.
		jst = time.FixedZone("JST", 9*60*60)
	}

	// Normalise today to midnight JST for consistent day-boundary comparison.
	nowJST := time.Now().In(jst)
	todayJST := time.Date(nowJST.Year(), nowJST.Month(), nowJST.Day(), 0, 0, 0, 0, jst)

	// Build a set of practice dates (as midnight JST) for O(1) lookup.
	dateSet := make(map[time.Time]struct{}, len(dates))
	for _, d := range dates {
		// Normalise each date to midnight JST to match todayJST.
		djst := d.In(jst)
		midnight := time.Date(djst.Year(), djst.Month(), djst.Day(), 0, 0, 0, 0, jst)
		dateSet[midnight] = struct{}{}
	}

	// Determine whether to start counting from today or yesterday.
	cursor := todayJST
	if _, practicedToday := dateSet[cursor]; !practicedToday {
		cursor = cursor.AddDate(0, 0, -1)
	}

	// Walk backwards day-by-day until a gap is found.
	streak := 0
	for {
		if _, ok := dateSet[cursor]; !ok {
			break
		}
		streak++
		cursor = cursor.AddDate(0, 0, -1)
	}

	return streak
}

// --- helpers ---

// parsePagination extracts limit and offset query parameters.
// Limit is clamped to [1, 100]; offset defaults to 0.
func parsePagination(r *http.Request) (limit, offset int) {
	limit = 20
	offset = 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 100 {
				n = 100
			}
			limit = n
		}
	}

	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	return limit, offset
}
