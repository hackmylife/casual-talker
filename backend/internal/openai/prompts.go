package openai

import (
	"fmt"
	"strings"

	"github.com/naoki-watanabe/casual-talker/backend/internal/domain"
)

// languageConfig holds language-specific prompt fragments used across all
// prompt builders.
type languageConfig struct {
	// partnerIntro is the opening sentence of the system prompt describing the AI.
	partnerIntro string
	// levelInstruction is the per-level instruction inserted into the system prompt.
	levelInstruction string
	// levelGuidelines describes what each level looks like in the target language.
	levelGuidelines string
	// interpretErrorDesc describes the STT error patterns common for Japanese
	// speakers learning this language.
	interpretErrorExamples string
	// interpretOutputLang specifies the language the corrected output must be in.
	interpretOutputLang string
	// feedbackConversationLabel is the label used when listing conversation turns
	// in the feedback prompt.
	feedbackConversationLabel string
	// feedbackLevelLabels maps level numbers to Japanese label strings.
	feedbackLevelLabels [5]string
	// hintLevelDesc describes the user's level in the hint prompt.
	hintLevelDesc string
}

// languageConfigs holds configurations for each supported target language.
var languageConfigs = map[string]languageConfig{
	"en": {
		partnerIntro:     "You are a friendly English conversation partner for Japanese beginners.",
		levelInstruction: "Use simple English appropriate for level %d (1-5)",
		levelGuidelines: `Level Guidelines:
- Level 1: Single words, very short phrases. Ask yes/no questions. Example: "Do you like sushi?" → expect "Yes"
- Level 2: Basic sentences (I am / I like). Ask simple what/where questions. Example: "What do you like?" → expect "I like sushi."
- Level 3: Add reasons. Ask "why" questions. Example: "Why do you like it?" → expect "Because it is delicious."
- Level 4: 2-3 sentences. Encourage elaboration. Example: "Tell me more about your hobby."
- Level 5: Free-form discussion with natural transitions.`,
		interpretErrorExamples: `Common Japanese speaker pronunciation errors in STT transcription:
- L/R confusion: "rike" → "like", "runch" → "lunch", "lamen" → "ramen"
- TH sounds: "sink" → "think", "dis" → "this"
- V/B confusion: "berry" → "very"
- SI/SHI confusion: "shi" → "si"
- Added vowels: "desuku" → "desk"`,
		interpretOutputLang:       "English",
		feedbackConversationLabel: "English conversation",
		feedbackLevelLabels: [5]string{
			"単語・超短文レベル",
			"基本文レベル",
			"理由追加レベル",
			"複数文レベル",
			"自発展開レベル",
		},
		hintLevelDesc: "Japanese English learner",
	},
	"it": {
		partnerIntro:     "You are a friendly Italian conversation partner for Japanese beginners.",
		levelInstruction: "Use simple Italian appropriate for level %d (1-5). Speak in Italian.",
		levelGuidelines: `Level Guidelines:
- Level 1: Single words, very short phrases. Ask yes/no questions. Example: "Ti piace il sushi?" → expect "Sì"
- Level 2: Basic sentences (Io sono / Mi piace). Ask simple what/where questions. Example: "Cosa ti piace?" → expect "Mi piace la pizza."
- Level 3: Add reasons. Ask "perché" questions. Example: "Perché ti piace?" → expect "Perché è buona."
- Level 4: 2-3 sentences. Encourage elaboration.
- Level 5: Free-form discussion with natural transitions.`,
		interpretErrorExamples: `Common Japanese speaker pronunciation errors in Italian STT transcription:
- Rolled R: "r" sounds may be transcribed as "l" or softened (e.g., "loma" → "Roma")
- Double consonants missed: "ano" → "anno", "belo" → "bello"
- Final vowels dropped or mispronounced: "parl" → "parlo"
- GL sounds: "fiyo" → "figlio"
- GN sounds: "lasana" → "lasagna"`,
		interpretOutputLang:       "Italian",
		feedbackConversationLabel: "Italian conversation",
		feedbackLevelLabels: [5]string{
			"単語・超短文レベル",
			"基本文レベル",
			"理由追加レベル",
			"複数文レベル",
			"自発展開レベル",
		},
		hintLevelDesc: "Japanese Italian learner",
	},
	"ko": {
		partnerIntro:     "You are a friendly Korean conversation partner for Japanese beginners.",
		levelInstruction: "Use simple Korean appropriate for level %d (1-5). Speak in Korean using polite form (존댓말). Speak in Korean.",
		levelGuidelines: `Level Guidelines:
- Level 1: Single words, very short phrases. Ask yes/no questions. Example: "스시 좋아해요?" → expect "네"
- Level 2: Basic sentences (저는 / 좋아해요). Ask simple what/where questions. Example: "뭘 좋아해요?" → expect "저는 스시를 좋아해요."
- Level 3: Add reasons. Ask "왜" questions. Example: "왜 좋아해요?" → expect "맛있으니까요."
- Level 4: 2-3 sentences. Encourage elaboration.
- Level 5: Free-form discussion with natural transitions.`,
		interpretErrorExamples: `Common Japanese speaker pronunciation errors in Korean STT transcription:
- Final consonants (받침) may be dropped: "먹" transcribed as "머", "있" as "이"
- Aspirated consonants (격음) confused with plain: "카피" → "가피" or "파" → "바"
- Tense consonants (경음) softened: "빵" → "방"
- Vowel ㅓ confused with ㅗ or ㅡ: "어머니" → "오머니"
- Long vowels and short vowels treated equally`,
		interpretOutputLang:       "Korean",
		feedbackConversationLabel: "Korean conversation",
		feedbackLevelLabels: [5]string{
			"単語・超短文レベル",
			"基本文レベル",
			"理由追加レベル",
			"複数文レベル",
			"自発展開レベル",
		},
		hintLevelDesc: "Japanese Korean learner",
	},
	"pt": {
		partnerIntro:     "You are a friendly Brazilian Portuguese conversation partner for Japanese beginners.",
		levelInstruction: "Use simple Brazilian Portuguese appropriate for level %d (1-5). Speak in Portuguese.",
		levelGuidelines: `Level Guidelines:
- Level 1: Single words, very short phrases. Ask yes/no questions. Example: "Você gosta de sushi?" → expect "Sim"
- Level 2: Basic sentences (Eu sou / Eu gosto). Ask simple what/where questions. Example: "O que você gosta?" → expect "Eu gosto de pizza."
- Level 3: Add reasons. Ask "por que" questions. Example: "Por que você gosta?" → expect "Porque é delicioso."
- Level 4: 2-3 sentences. Encourage elaboration.
- Level 5: Free-form discussion with natural transitions.`,
		interpretErrorExamples: `Common Japanese speaker pronunciation errors in Portuguese STT transcription:
- Nasal vowels (ã, ẽ, õ) may be transcribed without nasalization: "pão" → "pao", "irmã" → "irma"
- R at the start of words (guttural) may be dropped or softened: "rua" → "ua"
- LH sound: "filho" → "filio" or "fiyo"
- NH sound: "banho" → "banio"
- Final -m nasalizing the vowel: "bem" → "be"`,
		interpretOutputLang:       "Portuguese",
		feedbackConversationLabel: "Portuguese conversation",
		feedbackLevelLabels: [5]string{
			"単語・超短文レベル",
			"基本文レベル",
			"理由追加レベル",
			"複数文レベル",
			"自発展開レベル",
		},
		hintLevelDesc: "Japanese Portuguese learner",
	},
}

// langConfig returns the configuration for the given target language code,
// falling back to English if the code is not recognised.
func langConfig(targetLang string) languageConfig {
	if cfg, ok := languageConfigs[targetLang]; ok {
		return cfg
	}
	return languageConfigs["en"]
}

// BuildSystemPrompt constructs the system prompt sent to the AI at the start
// of every chat completion request. It injects the theme context, difficulty
// level, turn progress, and the target language so the model adapts its
// language accordingly.
func BuildSystemPrompt(theme domain.Theme, level int, turnNumber, maxTurns int, targetLang string) string {
	cfg := langConfig(targetLang)
	targetPhrases := string(theme.TargetPhrases)

	var sb strings.Builder
	sb.WriteString(cfg.partnerIntro + "\n\n")
	sb.WriteString("Rules:\n")
	sb.WriteString("- Always speak first in each turn\n")
	sb.WriteString(fmt.Sprintf("- %s\n", fmt.Sprintf(cfg.levelInstruction, level)))
	sb.WriteString("- Keep responses to 1-2 sentences\n")
	sb.WriteString(fmt.Sprintf("- Topic: %s\n", theme.Title))
	sb.WriteString(fmt.Sprintf("- Target phrases: %s\n", targetPhrases))
	sb.WriteString("- If the user struggles, simplify your language\n")
	sb.WriteString("- Never correct grammar directly during conversation\n")
	sb.WriteString("- Be encouraging and patient\n\n")

	sb.WriteString(cfg.levelGuidelines + "\n\n")

	sb.WriteString(fmt.Sprintf("Turn: %d of %d\n", turnNumber, maxTurns))
	sb.WriteString("When turn reaches maxTurns, wrap up the conversation naturally.")

	return sb.String()
}

// BuildHintPrompt produces a prompt that asks the AI to generate a structured
// hint for the user based on the AI's most recent message. The hint fields are
// in the target language, while the japanese field is always a Japanese
// translation of the AI message.
func BuildHintPrompt(aiMessage string, level int, targetLang string) string {
	cfg := langConfig(targetLang)
	return fmt.Sprintf(
		`The user is a %s at level %d (1-5) and needs help responding to the following AI message:

"%s"

Please provide a JSON object with exactly these three fields:
{
  "hint": "A simple hint or clue to guide the user (in the target language, keep it short)",
  "japanese": "Japanese translation of the AI message above",
  "sample_answer": "A natural, level-appropriate sample response the user could say (in the target language)"
}

Respond with valid JSON only. No markdown, no explanation.`,
		cfg.hintLevelDesc, level, aiMessage,
	)
}

// BuildInterpretPrompt constructs a prompt that asks the model to detect and
// correct pronunciation-related transcription errors in a raw STT string
// produced by a Japanese speaker learning the target language.
func BuildInterpretPrompt(rawText string, targetLang string) string {
	cfg := langConfig(targetLang)
	return fmt.Sprintf(
		`You are a %s pronunciation error corrector for Japanese speakers.

CRITICAL RULES:
- The output MUST be in %s. NEVER translate to Japanese or any other language.
- ONLY fix pronunciation-related transcription errors. Do NOT change the meaning or rephrase.
- Do NOT fix grammar. Only fix words that are clearly wrong due to pronunciation issues.
- If the text is already correct %s (even with grammar mistakes), return it as-is with is_different=false.

%s

Return ONLY a JSON object: {"interpreted": "corrected %s text", "is_different": true/false}
No explanation, no markdown, no translation.

Raw transcription: "%s"`,
		cfg.interpretOutputLang,
		cfg.interpretOutputLang,
		cfg.interpretOutputLang,
		cfg.interpretErrorExamples,
		cfg.interpretOutputLang,
		rawText,
	)
}

// BuildFeedbackPrompt generates a prompt that asks the AI to analyse the
// completed session turns and produce structured feedback in Japanese.
// The feedback text fields are always in Japanese (since the UI is Japanese),
// while example phrases in natural_expressions are in the target language.
func BuildFeedbackPrompt(turns []domain.Turn, targetLang string) string {
	cfg := langConfig(targetLang)

	var conversation strings.Builder
	for _, t := range turns {
		conversation.WriteString(fmt.Sprintf("AI: %s\n", t.AIText))
		if t.UserText != nil {
			conversation.WriteString(fmt.Sprintf("Student: %s\n", *t.UserText))
		}
	}

	labels := cfg.feedbackLevelLabels

	return fmt.Sprintf(
		`Analyze the following %s conversation between an AI teacher and a Japanese beginner student.
Provide feedback in the following JSON format:
{
  "achievements": ["string array of things the student did well, in Japanese"],
  "natural_expressions": [{"original": "what student actually said in %s", "natural": "more natural way to say it in %s"}],
  "improvements": [{"point": "improvement point in Japanese", "example": "concrete example sentence in %s showing the correct usage"}],
  "review_phrases": ["up to 3 key phrases the student should practice (in the target language)"],
  "current_level": {
    "level": <number 1-5>,
    "label": "<Japanese level label>",
    "description": "<one-sentence Japanese description of what the student can do>"
  },
  "next_level_advice": "<specific, actionable advice in Japanese on how to reach the next level>"
}

Rules:
- achievements: at least 1, written in Japanese, encouraging tone
- natural_expressions: compare student's actual words with natural alternatives. CRITICAL: Both "original" and "natural" fields MUST be written in the TARGET LANGUAGE (%s), NEVER in Japanese. Only include entries where the WORDING or GRAMMAR is meaningfully different. Do NOT include entries that differ only in punctuation, spacing, capitalization, or exclamation marks. If the student's expression is already natural, return an empty array [].
- improvements: max 2 items. Each item has "point" (Japanese explanation) and "example" (a concrete example sentence in %s showing the correct usage). Be gentle, no negative language.
- review_phrases: max 3, target-language phrases for review
- current_level: assess the student's speaking level based on their responses:
    Level 1 (%s): Can only say single words or yes/no answers
    Level 2 (%s): Can form basic sentences
    Level 3 (%s): Can add reasons or explanations
    Level 4 (%s): Can speak 2-3 sentences and elaborate on topics
    Level 5 (%s): Can lead conversation and change topics naturally
  Return the matching label and a short Japanese description of what the student can do at that level.
- next_level_advice: encouraging Japanese advice on reaching the next level; if already level 5, advise on maintaining and improving fluency. Use a warm, supportive tone.
- Return ONLY valid JSON, no markdown, no explanation

Conversation:
%s`,
		cfg.feedbackConversationLabel,
		cfg.interpretOutputLang, cfg.interpretOutputLang, // natural_expressions: original, natural
		cfg.interpretOutputLang, // improvements: example
		cfg.interpretOutputLang, cfg.interpretOutputLang, // rules: natural_expressions, improvements
		labels[0], labels[1], labels[2], labels[3], labels[4],
		conversation.String(),
	)
}
