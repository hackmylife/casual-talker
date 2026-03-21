package openai

import (
	"fmt"
	"strings"

	"github.com/naoki-watanabe/casual-talker/backend/internal/domain"
)

// BuildSystemPrompt constructs the system prompt sent to the AI at the start
// of every chat completion request. It injects the theme context, difficulty
// level, and turn progress so the model adapts its language accordingly.
func BuildSystemPrompt(theme domain.Theme, level int, turnNumber, maxTurns int) string {
	targetPhrases := string(theme.TargetPhrases)

	var sb strings.Builder
	sb.WriteString("You are a friendly English conversation partner for Japanese beginners.\n\n")
	sb.WriteString("Rules:\n")
	sb.WriteString("- Always speak first in each turn\n")
	sb.WriteString(fmt.Sprintf("- Use simple English appropriate for level %d (1-5)\n", level))
	sb.WriteString("- Keep responses to 1-2 sentences\n")
	sb.WriteString(fmt.Sprintf("- Topic: %s\n", theme.Title))
	sb.WriteString(fmt.Sprintf("- Target phrases: %s\n", targetPhrases))
	sb.WriteString("- If the user struggles, simplify your language\n")
	sb.WriteString("- Never correct grammar directly during conversation\n")
	sb.WriteString("- Be encouraging and patient\n\n")

	sb.WriteString("Level Guidelines:\n")
	sb.WriteString("- Level 1: Single words, very short phrases. Ask yes/no questions.\n")
	sb.WriteString("- Level 2: Basic sentences (I am / I like). Ask simple what/where questions.\n")
	sb.WriteString("- Level 3: Add reasons. Ask \"why\" questions.\n")
	sb.WriteString("- Level 4: 2-3 sentences. Encourage elaboration.\n")
	sb.WriteString("- Level 5: Free-form discussion with natural transitions.\n\n")

	sb.WriteString(fmt.Sprintf("Turn: %d of %d\n", turnNumber, maxTurns))
	sb.WriteString("When turn reaches maxTurns, wrap up the conversation naturally.")

	return sb.String()
}

// BuildHintPrompt produces a prompt that asks the AI to generate a structured
// hint for the user based on the AI's most recent message.
func BuildHintPrompt(aiMessage string, level int) string {
	return fmt.Sprintf(
		`The user is a Japanese English learner at level %d (1-5) and needs help responding to the following AI message:

"%s"

Please provide a JSON object with exactly these three fields:
{
  "hint": "A simple hint or clue to guide the user (in English, keep it short)",
  "japanese": "Japanese translation of the AI message above",
  "sample_answer": "A natural, level-appropriate sample response the user could say"
}

Respond with valid JSON only. No markdown, no explanation.`,
		level, aiMessage,
	)
}

// BuildInterpretPrompt constructs a prompt that asks the model to detect and
// correct pronunciation-related transcription errors in a raw STT string
// produced by a Japanese English learner.
func BuildInterpretPrompt(rawText string) string {
	return fmt.Sprintf(
		`You are an English pronunciation error corrector for Japanese speakers learning English.

CRITICAL RULES:
- The output MUST be in English. NEVER translate to Japanese, Korean, Chinese, or any other language.
- ONLY fix pronunciation-related transcription errors. Do NOT change the meaning or rephrase.
- Do NOT fix grammar. Only fix words that are clearly wrong due to pronunciation issues.
- If the text is already correct English (even with grammar mistakes), return it as-is with is_different=false.

Common Japanese speaker pronunciation errors in STT transcription:
- L/R confusion: "rike" → "like", "runch" → "lunch", "lamen" → "ramen"
- TH sounds: "sink" → "think", "dis" → "this"
- V/B confusion: "berry" → "very"
- SI/SHI confusion: "shi" → "si"
- Added vowels: "desuku" → "desk"

Return ONLY a JSON object: {"interpreted": "corrected English text", "is_different": true/false}
No explanation, no markdown, no translation.

Raw transcription: "%s"`, rawText,
	)
}

// BuildFeedbackPrompt generates a prompt that asks the AI to analyse the
// completed session turns and produce structured feedback.
//
// The prompt instructs the model to return a strict JSON object with six
// fields. natural_expressions is an array of objects, not strings, so the
// caller must parse it accordingly. current_level is an object describing
// the student's assessed level, and next_level_advice is a Japanese string
// with actionable guidance toward the next level.
func BuildFeedbackPrompt(turns []domain.Turn) string {
	var conversation strings.Builder
	for _, t := range turns {
		conversation.WriteString(fmt.Sprintf("AI: %s\n", t.AIText))
		if t.UserText != nil {
			conversation.WriteString(fmt.Sprintf("Student: %s\n", *t.UserText))
		}
	}

	return fmt.Sprintf(
		`Analyze the following English conversation between an AI teacher and a Japanese beginner student.
Provide feedback in the following JSON format:
{
  "achievements": ["string array of things the student did well, in Japanese"],
  "natural_expressions": [{"original": "what student said", "natural": "more natural way to say it"}],
  "improvements": ["1-2 specific improvement points in Japanese"],
  "review_phrases": ["up to 3 key phrases the student should practice"],
  "current_level": {
    "level": <number 1-5>,
    "label": "<Japanese level label>",
    "description": "<one-sentence Japanese description of what the student can do>"
  },
  "next_level_advice": "<specific, actionable advice in Japanese on how to reach the next level>"
}

Rules:
- achievements: at least 1, written in Japanese, encouraging tone
- natural_expressions: compare student's actual words with natural alternatives (omit if student's English was already natural)
- improvements: max 2 items, in Japanese, gentle tone, no negative language
- review_phrases: max 3, English phrases for review
- current_level: assess the student's speaking level based on their responses:
    Level 1 (単語・超短文レベル): Can only say single words or yes/no answers
    Level 2 (基本文レベル): Can form basic sentences (I am..., I like...)
    Level 3 (理由追加レベル): Can add reasons or explanations (because..., so...)
    Level 4 (複数文レベル): Can speak 2-3 sentences and elaborate on topics
    Level 5 (自発展開レベル): Can lead conversation and change topics naturally
  Return the matching label and a short Japanese description of what the student can do at that level.
- next_level_advice: encouraging Japanese advice on reaching the next level; if already level 5, advise on maintaining and improving fluency. Use a warm, supportive tone — not a score, but guidance like "今ここにいるよ、次はこうしたらいいよ".
- Return ONLY valid JSON, no markdown, no explanation

Conversation:
%s`,
		conversation.String(),
	)
}
