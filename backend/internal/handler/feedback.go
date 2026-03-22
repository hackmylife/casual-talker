package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	openailib "github.com/sashabaranov/go-openai"

	"github.com/naoki-watanabe/casual-talker/backend/internal/domain"
	"github.com/naoki-watanabe/casual-talker/backend/internal/middleware"
	oai "github.com/naoki-watanabe/casual-talker/backend/internal/openai"
	"github.com/naoki-watanabe/casual-talker/backend/internal/repository"
)

const (
	feedbackModel       = openailib.GPT4oMini
	feedbackTemperature = 0.3 // low temperature for deterministic JSON output
)

// FeedbackHandler exposes the feedback generation endpoint.
type FeedbackHandler struct {
	sessionRepo repository.SessionRepository
	oaiClient   *oai.Client
}

// NewFeedbackHandler creates a new FeedbackHandler.
func NewFeedbackHandler(sessionRepo repository.SessionRepository, oaiClient *oai.Client) *FeedbackHandler {
	return &FeedbackHandler{
		sessionRepo: sessionRepo,
		oaiClient:   oaiClient,
	}
}

// --- request types ---

type generateFeedbackRequest struct {
	SessionID string `json:"session_id" validate:"required"`
}

// --- handlers ---

// Generate handles POST /api/v1/feedback/generate.
//
// It loads the session, fetches all turns, calls GPT to produce structured
// feedback JSON, persists the result, and returns it to the caller.
// If the LLM response cannot be parsed as valid JSON, the raw text is stored
// in raw_llm_response and empty arrays are returned for the structured fields
// so the call still succeeds.
func (h *FeedbackHandler) Generate(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req generateFeedbackRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Verify session ownership.
	session, err := h.sessionRepo.GetSession(r.Context(), req.SessionID)
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

	fb, err := generateFeedback(r.Context(), h.sessionRepo, h.oaiClient, session.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "feedback generation failed")
		return
	}

	writeJSON(w, http.StatusCreated, toFeedbackResponse(*fb))
}

// generateFeedback encapsulates the shared logic used by both FeedbackHandler
// and SessionHandler: fetch turns, call the LLM, parse the response, and
// persist the feedback record.
//
// On LLM JSON parse failure the raw response text is stored so nothing is lost,
// and empty arrays are used for the structured fields so callers can always
// display something meaningful.
func generateFeedback(
	ctx context.Context,
	sessionRepo repository.SessionRepository,
	oaiClient *oai.Client,
	sessionID string,
) (*domain.Feedback, error) {
	turns, err := sessionRepo.ListTurnsBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Resolve the target language from the session's theme and course.
	targetLang := "en"
	session, err := sessionRepo.GetSession(ctx, sessionID)
	if err == nil {
		if theme, err := sessionRepo.GetTheme(ctx, session.ThemeID); err == nil {
			if course, err := sessionRepo.GetCourse(ctx, theme.CourseID); err == nil {
				targetLang = course.TargetLanguage
			}
		}
	}

	prompt := oai.BuildFeedbackPrompt(turns, targetLang)

	resp, err := oaiClient.Underlying().CreateChatCompletion(ctx, openailib.ChatCompletionRequest{
		Model: feedbackModel,
		Messages: []openailib.ChatCompletionMessage{
			{Role: openailib.ChatMessageRoleUser, Content: prompt},
		},
		Temperature: feedbackTemperature,
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		slog.Error("empty LLM response for feedback", "session_id", sessionID)
		return nil, errEmptyLLMResponse
	}

	raw := resp.Choices[0].Message.Content

	fb := buildFeedbackFromRaw(sessionID, raw)

	saved, err := sessionRepo.CreateFeedback(ctx, fb)
	if err != nil {
		return nil, err
	}

	return saved, nil
}

// llmFeedbackPayload matches the JSON structure the LLM is asked to produce.
type llmFeedbackPayload struct {
	Achievements       []string               `json:"achievements"`
	NaturalExpressions []naturalExpressionItem `json:"natural_expressions"`
	Improvements       []string               `json:"improvements"`
	ReviewPhrases      []string               `json:"review_phrases"`
	CurrentLevel       *levelInfo             `json:"current_level"`
	NextLevelAdvice    string                 `json:"next_level_advice"`
}

type naturalExpressionItem struct {
	Original string `json:"original"`
	Natural  string `json:"natural"`
}

// levelInfo represents the assessed speaking level returned by the LLM.
type levelInfo struct {
	Level       int    `json:"level"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// buildFeedbackFromRaw attempts to parse the raw LLM text as structured JSON.
// When parsing fails it stores the raw text and falls back to empty arrays and
// null level info so downstream code never has to handle nil slices.
func buildFeedbackFromRaw(sessionID, raw string) *domain.Feedback {
	fb := &domain.Feedback{
		SessionID:      sessionID,
		RawLLMResponse: &raw,
	}

	var payload llmFeedbackPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		slog.Warn("failed to parse LLM feedback JSON; storing raw response",
			"session_id", sessionID,
			"error", err,
			"raw", raw,
		)
		// Fall back to empty JSON values so the columns are never NULL.
		fb.Achievements = json.RawMessage("[]")
		fb.NaturalExpressions = json.RawMessage("[]")
		fb.Improvements = json.RawMessage("[]")
		fb.ReviewPhrases = json.RawMessage("[]")
		fb.CurrentLevel = json.RawMessage("{}")
		return fb
	}

	// Re-marshal each field individually so we store compact, canonical JSON.
	achievements, _ := json.Marshal(payload.Achievements)
	naturalExpressions, _ := json.Marshal(payload.NaturalExpressions)
	improvements, _ := json.Marshal(payload.Improvements)
	reviewPhrases, _ := json.Marshal(payload.ReviewPhrases)

	fb.Achievements = achievements
	fb.NaturalExpressions = naturalExpressions
	fb.Improvements = improvements
	fb.ReviewPhrases = reviewPhrases

	// current_level: marshal the level object if present; fall back to empty object.
	if payload.CurrentLevel != nil {
		if levelJSON, err := json.Marshal(payload.CurrentLevel); err == nil {
			fb.CurrentLevel = levelJSON
		} else {
			fb.CurrentLevel = json.RawMessage("{}")
		}
	} else {
		fb.CurrentLevel = json.RawMessage("{}")
	}

	// next_level_advice: store as a nullable string pointer.
	if payload.NextLevelAdvice != "" {
		advice := payload.NextLevelAdvice
		fb.NextLevelAdvice = &advice
	}

	return fb
}

// errEmptyLLMResponse is returned when the OpenAI response contains no choices.
var errEmptyLLMResponse = errString("empty response from model")

// errString is a simple string-based error type used for sentinel values.
type errString string

func (e errString) Error() string { return string(e) }
