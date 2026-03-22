package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	openailib "github.com/sashabaranov/go-openai"

	"github.com/naoki-watanabe/casual-talker/backend/internal/domain"
	"github.com/naoki-watanabe/casual-talker/backend/internal/middleware"
	oai "github.com/naoki-watanabe/casual-talker/backend/internal/openai"
	"github.com/naoki-watanabe/casual-talker/backend/internal/repository"
)

const (
	chatModel       = openailib.GPT4oMini
	chatTemperature = 0.7
)

// ChatHandler exposes HTTP endpoints for AI-driven conversation streaming and
// hint generation.
type ChatHandler struct {
	openai      *oai.Client
	sessionRepo repository.SessionRepository
}

// NewChatHandler creates a new ChatHandler.
func NewChatHandler(client *oai.Client, sessionRepo repository.SessionRepository) *ChatHandler {
	return &ChatHandler{
		openai:      client,
		sessionRepo: sessionRepo,
	}
}

// --- request types ---

type streamRequest struct {
	SessionID       string `json:"session_id" validate:"required"`
	Message         string `json:"message"`
	InterpretedText string `json:"interpreted_text"` // what the user likely meant (optional)
}

type hintRequest struct {
	SessionID  string `json:"session_id"  validate:"required"`
	TurnNumber int    `json:"turn_number" validate:"min=0"`
}

type interpretRequest struct {
	SessionID string `json:"session_id" validate:"required"`
	RawText   string `json:"raw_text"   validate:"required"`
}

type interpretResponse struct {
	Interpreted string `json:"interpreted"`
	IsDifferent bool   `json:"is_different"`
}

// --- handlers ---

// Stream handles POST /api/v1/chat/stream.
//
// It loads the session and theme, builds a system prompt, then calls the
// OpenAI ChatCompletion streaming API, forwarding each delta to the client
// via Server-Sent Events. After the stream finishes, a background goroutine
// persists the completed turn to the database.
func (h *ChatHandler) Stream(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req streamRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Load and verify session ownership.
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

	// Load theme for prompt construction.
	theme, err := h.sessionRepo.GetTheme(r.Context(), session.ThemeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Load the parent course to determine the target language.
	course, err := h.sessionRepo.GetCourse(r.Context(), theme.CourseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	targetLang := course.TargetLanguage

	// Determine the current turn number from existing turns.
	turns, err := h.sessionRepo.ListTurnsBySession(r.Context(), session.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	currentTurn := len(turns) + 1

	// Build the message list for the ChatCompletion call.
	// When an interpreted version of the user's text is provided, pass that to
	// the model so it can respond to the intended meaning rather than the
	// phonetically transcribed text.
	userMsgForAI := req.Message
	if req.InterpretedText != "" && req.InterpretedText != req.Message {
		userMsgForAI = req.InterpretedText
	}
	// Fetch summaries of past sessions on the same theme to avoid repetitive
	// conversation starters.
	pastTopics, _ := h.sessionRepo.GetPastSessionTopics(r.Context(), userID, session.ThemeID, 5)
	systemPrompt := oai.BuildSystemPrompt(*theme, session.Difficulty, currentTurn, session.MaxTurns, targetLang, pastTopics)
	messages := buildChatMessages(systemPrompt, turns, userMsgForAI)

	// Log the message list for debugging conversation coherence.
	for i, m := range messages {
		if m.Role == "system" {
			slog.Debug("chat msg", "i", i, "role", m.Role, "len", len(m.Content))
		} else {
			slog.Debug("chat msg", "i", i, "role", m.Role, "content", m.Content)
		}
	}

	// Configure SSE headers. The X-Accel-Buffering header disables nginx
	// proxy buffering so chunks reach the client immediately.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// Open a streaming ChatCompletion request.
	streamReq := openailib.ChatCompletionRequest{
		Model:       chatModel,
		Messages:    messages,
		Temperature: chatTemperature,
		Stream:      true,
	}

	stream, err := h.openai.Underlying().CreateChatCompletionStream(r.Context(), streamReq)
	if err != nil {
		slog.Error("openai stream error", "error", err, "session_id", session.ID, "turn", currentTurn, "msg_count", len(messages))
		writeSSEError(w, flusher, "failed to start stream")
		return
	}
	defer stream.Close()

	var aiResponseBuilder strings.Builder

	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			writeSSEError(w, flusher, "stream error")
			return
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		content := chunk.Choices[0].Delta.Content
		if content == "" {
			continue
		}

		aiResponseBuilder.WriteString(content)

		payload, _ := json.Marshal(map[string]string{"content": content})
		fmt.Fprintf(w, "data: %s\n\n", payload)
		flusher.Flush()
	}

	// Signal completion to the client.
	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()

	// Persist the turn synchronously so subsequent requests can see it.
	// Always store the raw user text (before interpretation) so the turn
	// history reflects what the user actually said. When the interpreted text
	// differs from the raw input, store it separately for later review.
	aiText := aiResponseBuilder.String()
	turn := &domain.Turn{
		SessionID:  session.ID,
		TurnNumber: currentTurn,
		AIText:     aiText,
	}
	if req.Message != "" {
		turn.UserText = &req.Message
	}
	if req.InterpretedText != "" && req.InterpretedText != req.Message {
		turn.InterpretedText = &req.InterpretedText
	}
	if _, err := h.sessionRepo.CreateTurn(context.Background(), turn); err != nil {
		slog.Error("failed to persist turn", "session_id", session.ID, "turn", currentTurn, "error", err)
	}
}

// Interpret handles POST /api/v1/chat/interpret.
//
// It takes a raw STT-transcribed text string and asks the model to correct any
// pronunciation-related transcription errors common for Japanese speakers (e.g.
// L/R confusion, TH sounds, V/B confusion). The response includes the corrected
// text and a boolean indicating whether a correction was made.
//
// The call is intentionally non-streaming and uses a low temperature so that
// the result is fast and deterministic.
func (h *ChatHandler) Interpret(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req interpretRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Verify session ownership (keeps the endpoint consistent with the others).
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

	// Resolve target language from theme → course.
	interpretLang := "en"
	if theme, err := h.sessionRepo.GetTheme(r.Context(), session.ThemeID); err == nil {
		if course, err := h.sessionRepo.GetCourse(r.Context(), theme.CourseID); err == nil {
			interpretLang = course.TargetLanguage
		}
	}

	prompt := oai.BuildInterpretPrompt(req.RawText, interpretLang)

	resp, err := h.openai.Underlying().CreateChatCompletion(r.Context(), openailib.ChatCompletionRequest{
		Model: chatModel,
		Messages: []openailib.ChatCompletionMessage{
			{Role: openailib.ChatMessageRoleUser, Content: prompt},
		},
		Temperature: 0.2, // low temperature for deterministic correction
		MaxTokens:   200,
	})
	if err != nil {
		// On model failure fall back to returning the raw text unchanged.
		slog.Warn("interpret: openai call failed, returning raw text", "error", err)
		writeJSON(w, http.StatusOK, interpretResponse{Interpreted: req.RawText, IsDifferent: false})
		return
	}

	if len(resp.Choices) == 0 {
		writeJSON(w, http.StatusOK, interpretResponse{Interpreted: req.RawText, IsDifferent: false})
		return
	}

	raw := strings.TrimSpace(resp.Choices[0].Message.Content)

	var result interpretResponse
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		// JSON parse failure: return raw text as-is so the caller can proceed.
		slog.Warn("interpret: failed to parse model JSON, returning raw text", "error", err, "response", raw)
		writeJSON(w, http.StatusOK, interpretResponse{Interpreted: req.RawText, IsDifferent: false})
		return
	}

	// Guard against an empty interpreted field.
	if result.Interpreted == "" {
		result.Interpreted = req.RawText
		result.IsDifferent = false
	}

	writeJSON(w, http.StatusOK, result)
}

// Hint handles POST /api/v1/chat/hint.
//
// It retrieves the AI message for the requested turn and asks the model to
// generate a structured hint containing an English clue, a Japanese translation,
// and a sample answer.
func (h *ChatHandler) Hint(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req hintRequest
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

	// Find the AI message for the requested turn.
	turns, err := h.sessionRepo.ListTurnsBySession(r.Context(), session.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	var aiMessage string
	for _, t := range turns {
		if t.TurnNumber == req.TurnNumber {
			aiMessage = t.AIText
			break
		}
	}

	if aiMessage == "" {
		writeError(w, http.StatusNotFound, "turn not found")
		return
	}

	// Resolve target language from theme → course.
	hintLang := "en"
	if theme, err := h.sessionRepo.GetTheme(r.Context(), session.ThemeID); err == nil {
		if course, err := h.sessionRepo.GetCourse(r.Context(), theme.CourseID); err == nil {
			hintLang = course.TargetLanguage
		}
	}

	// Ask the model for a structured hint.
	hintPrompt := oai.BuildHintPrompt(aiMessage, session.Difficulty, hintLang)

	resp, err := h.openai.Underlying().CreateChatCompletion(r.Context(), openailib.ChatCompletionRequest{
		Model: chatModel,
		Messages: []openailib.ChatCompletionMessage{
			{Role: openailib.ChatMessageRoleUser, Content: hintPrompt},
		},
		Temperature: 0.3, // low temperature for deterministic JSON output
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hint generation failed")
		return
	}

	if len(resp.Choices) == 0 {
		writeError(w, http.StatusInternalServerError, "empty response from model")
		return
	}

	raw := resp.Choices[0].Message.Content

	// Validate that the model returned parseable JSON with the expected shape.
	var hint struct {
		Hint         string `json:"hint"`
		Japanese     string `json:"japanese"`
		SampleAnswer string `json:"sample_answer"`
	}
	if err := json.Unmarshal([]byte(raw), &hint); err != nil {
		writeError(w, http.StatusInternalServerError, "invalid hint format from model")
		return
	}

	writeJSON(w, http.StatusOK, hint)
}

// --- helpers ---

// buildChatMessages assembles the full message slice for the ChatCompletion
// call, including the system prompt, previous turn history, and the latest
// user message.
func buildChatMessages(systemPrompt string, turns []domain.Turn, userMessage string) []openailib.ChatCompletionMessage {
	msgs := []openailib.ChatCompletionMessage{
		{Role: openailib.ChatMessageRoleSystem, Content: systemPrompt},
	}

	for _, t := range turns {
		msgs = append(msgs, openailib.ChatCompletionMessage{
			Role:    openailib.ChatMessageRoleAssistant,
			Content: t.AIText,
		})
		if t.UserText != nil && *t.UserText != "" {
			msgs = append(msgs, openailib.ChatCompletionMessage{
				Role:    openailib.ChatMessageRoleUser,
				Content: *t.UserText,
			})
		}
	}

	if userMessage != "" {
		msgs = append(msgs, openailib.ChatCompletionMessage{
			Role:    openailib.ChatMessageRoleUser,
			Content: userMessage,
		})
	}

	return msgs
}

// writeSSEError sends an error event over the SSE channel and flushes it.
func writeSSEError(w http.ResponseWriter, flusher http.Flusher, msg string) {
	payload, _ := json.Marshal(map[string]string{"error": msg})
	fmt.Fprintf(w, "data: %s\n\n", payload)
	flusher.Flush()
}
