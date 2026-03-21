package handler

import (
	"io"
	"net/http"

	openailib "github.com/sashabaranov/go-openai"

	oai "github.com/naoki-watanabe/casual-talker/backend/internal/openai"
)

const (
	maxAudioUploadBytes = 25 << 20 // 25 MiB — Whisper API limit
)

// SpeechHandler exposes HTTP endpoints for speech-to-text and text-to-speech.
type SpeechHandler struct {
	openai *oai.Client
}

// NewSpeechHandler creates a new SpeechHandler.
func NewSpeechHandler(client *oai.Client) *SpeechHandler {
	return &SpeechHandler{openai: client}
}

// STT handles POST /api/v1/speech/stt.
// It accepts a multipart/form-data request with an "audio" field containing
// the audio file, forwards it to OpenAI Whisper, and returns the transcript.
func (h *SpeechHandler) STT(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxAudioUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		writeError(w, http.StatusBadRequest, "audio field is required")
		return
	}
	defer file.Close()

	// Use a fixed filename to prevent the client from influencing the format
	// detection via a crafted filename.
	_ = header // filename from client is intentionally ignored
	req := openailib.AudioRequest{
		Model:    openailib.Whisper1,
		Reader:   file,
		FilePath: "audio.webm",
	}

	resp, err := h.openai.Underlying().CreateTranscription(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "transcription failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"text": resp.Text})
}

// TTS handles POST /api/v1/speech/tts.
// It generates speech from the provided text using OpenAI TTS and streams the
// MP3 audio directly to the client.
func (h *SpeechHandler) TTS(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text  string  `json:"text"  validate:"required,max=4096"`
		Speed float64 `json:"speed"`
	}

	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Default speed to 1.0 when not provided or out of the valid range.
	if req.Speed <= 0 || req.Speed > 4.0 {
		req.Speed = 1.0
	}

	ttsReq := openailib.CreateSpeechRequest{
		Model:          openailib.TTSModel1,
		Input:          req.Text,
		Voice:          openailib.VoiceNova,
		ResponseFormat: openailib.SpeechResponseFormatMp3,
		Speed:          req.Speed,
	}

	resp, err := h.openai.Underlying().CreateSpeech(r.Context(), ttsReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "speech synthesis failed")
		return
	}
	defer resp.Close()

	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, resp); err != nil {
		// The response header is already sent; we can only log the error.
		// Returning here avoids a superfluous write attempt.
		return
	}
}
