package openai

import (
	"strings"
	"testing"

	"github.com/naoki-watanabe/casual-talker/backend/internal/domain"
)

// newTestTheme returns a minimal domain.Theme suitable for use in tests.
func newTestTheme(title string) domain.Theme {
	return domain.Theme{
		ID:            "theme-test",
		CourseID:      "course-test",
		Title:         title,
		TargetPhrases: []byte(`["Hello","How are you?"]`),
	}
}

// --- BuildSystemPrompt ---

func TestBuildSystemPrompt_ContainsEnglish(t *testing.T) {
	theme := newTestTheme("Greetings")
	prompt := BuildSystemPrompt(theme, 1, 1, 10, "en", nil)
	if !strings.Contains(prompt, "English") {
		t.Errorf("expected prompt to contain %q for lang 'en', got:\n%s", "English", prompt)
	}
}

func TestBuildSystemPrompt_ContainsJapanese(t *testing.T) {
	theme := newTestTheme("あいさつ")
	prompt := BuildSystemPrompt(theme, 1, 1, 10, "ja", nil)
	if !strings.Contains(prompt, "Japanese") {
		t.Errorf("expected prompt to contain %q for lang 'ja', got:\n%s", "Japanese", prompt)
	}
}

func TestBuildSystemPrompt_LastTurn(t *testing.T) {
	theme := newTestTheme("Greetings")
	maxTurns := 5
	// Current turn equals maxTurns → last turn.
	prompt := BuildSystemPrompt(theme, 1, maxTurns, maxTurns, "en", nil)
	if !strings.Contains(prompt, "LAST turn") {
		t.Errorf("expected prompt to contain 'LAST turn' on final turn, got:\n%s", prompt)
	}
}

func TestBuildSystemPrompt_SecondToLastTurn(t *testing.T) {
	theme := newTestTheme("Greetings")
	maxTurns := 5
	// Current turn is maxTurns-1 → second-to-last.
	prompt := BuildSystemPrompt(theme, 1, maxTurns-1, maxTurns, "en", nil)
	if !strings.Contains(prompt, "second-to-last") {
		t.Errorf("expected prompt to contain 'second-to-last' on penultimate turn, got:\n%s", prompt)
	}
}

func TestBuildSystemPrompt_ContainsPastTopics(t *testing.T) {
	theme := newTestTheme("Greetings")
	pastTopics := []string{
		"Hi! My name is Alex. Nice to meet you!",
		"Hello! I'm Mia. What brings you here today?",
	}
	prompt := BuildSystemPrompt(theme, 1, 1, 10, "en", pastTopics)
	for _, topic := range pastTopics {
		if !strings.Contains(prompt, topic) {
			t.Errorf("expected prompt to contain past topic %q, got:\n%s", topic, prompt)
		}
	}
}

func TestBuildSystemPrompt_ContainsLevelNumber(t *testing.T) {
	theme := newTestTheme("Greetings")
	for _, level := range []int{1, 2, 3, 4, 5} {
		prompt := BuildSystemPrompt(theme, level, 1, 10, "en", nil)
		levelStr := string(rune('0' + level))
		if !strings.Contains(prompt, levelStr) {
			t.Errorf("level %d: expected prompt to contain %q, got:\n%s", level, levelStr, prompt)
		}
	}
}

// --- BuildInterpretPrompt ---

func TestBuildInterpretPrompt_English(t *testing.T) {
	prompt := BuildInterpretPrompt("rike dis", "en")
	if !strings.Contains(prompt, "English") {
		t.Errorf("expected prompt to contain 'English', got:\n%s", prompt)
	}
}

func TestBuildInterpretPrompt_Korean(t *testing.T) {
	prompt := BuildInterpretPrompt("머거요", "ko")
	if !strings.Contains(prompt, "Korean") {
		t.Errorf("expected prompt to contain 'Korean', got:\n%s", prompt)
	}
}

func TestBuildInterpretPrompt_ContainsRawText(t *testing.T) {
	rawText := "rike dis very much"
	prompt := BuildInterpretPrompt(rawText, "en")
	if !strings.Contains(prompt, rawText) {
		t.Errorf("expected prompt to contain raw text %q, got:\n%s", rawText, prompt)
	}
}

// --- BuildFeedbackPrompt ---

func TestBuildFeedbackPrompt_ContainsTurnConversation(t *testing.T) {
	userText := "I like sushi."
	turns := []domain.Turn{
		{AIText: "Do you like sushi?", UserText: &userText},
	}
	prompt := BuildFeedbackPrompt(turns, "en")
	if !strings.Contains(prompt, "Do you like sushi?") {
		t.Errorf("expected prompt to contain AI text, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, userText) {
		t.Errorf("expected prompt to contain user text %q, got:\n%s", userText, prompt)
	}
}

func TestBuildFeedbackPrompt_ContainsTargetLanguageName(t *testing.T) {
	turns := []domain.Turn{
		{AIText: "Do you like sushi?"},
	}
	tests := []struct {
		lang     string
		wantLang string
	}{
		{"en", "English"},
		{"ko", "Korean"},
		{"it", "Italian"},
		{"ja", "Japanese"},
	}
	for _, tc := range tests {
		prompt := BuildFeedbackPrompt(turns, tc.lang)
		if !strings.Contains(prompt, tc.wantLang) {
			t.Errorf("lang=%q: expected prompt to contain %q, got:\n%s", tc.lang, tc.wantLang, prompt)
		}
	}
}

func TestBuildFeedbackPrompt_ContainsLevelLabels(t *testing.T) {
	turns := []domain.Turn{
		{AIText: "Hello!"},
	}
	prompt := BuildFeedbackPrompt(turns, "en")
	for _, label := range defaultLevelLabels {
		if !strings.Contains(prompt, label) {
			t.Errorf("expected prompt to contain level label %q, got:\n%s", label, prompt)
		}
	}
}

// --- BuildHintPrompt ---

func TestBuildHintPrompt_ContainsAIMessage(t *testing.T) {
	aiMessage := "What do you like to do on weekends?"
	prompt := BuildHintPrompt(aiMessage, 2, "en")
	if !strings.Contains(prompt, aiMessage) {
		t.Errorf("expected hint prompt to contain AI message %q, got:\n%s", aiMessage, prompt)
	}
}
