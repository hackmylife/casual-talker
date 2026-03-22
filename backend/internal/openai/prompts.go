package openai

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/naoki-watanabe/casual-talker/backend/internal/domain"
)

// personas for AI conversation variety — the AI picks a different identity each session.
var personas = []struct {
	Name        string
	Personality string
}{
	{"Alex", "cheerful and enthusiastic, loves traveling"},
	{"Mia", "calm and thoughtful, enjoys reading and cooking"},
	{"Sam", "funny and casual, big sports fan"},
	{"Yuki", "gentle and patient, interested in art and music"},
	{"Leo", "energetic and curious, always asking follow-up questions"},
	{"Emma", "warm and supportive, loves sharing personal stories"},
}

// scenariosByTheme maps theme titles to conversation scenario variations.
// Each session randomly picks one scenario to keep conversations fresh.
var scenariosByTheme = map[string][]string{
	// English
	"Greetings":         {"meeting at a coffee shop for the first time", "running into someone at a park", "starting a conversation at a party", "greeting a new neighbor"},
	"Self Introduction": {"first day at a new class", "meeting at a language exchange event", "introducing yourself at a work meeting", "chatting on an online video call"},
	"Family":            {"showing family photos to a friend", "talking about a family trip", "discussing family traditions", "comparing families with a friend"},
	"Hobbies":           {"recommending hobbies to each other", "talking about a new hobby you started", "discussing weekend activities", "sharing a funny hobby story", "recomendando hobbies um ao outro", "falando sobre um novo hobby", "discutindo atividades de fim de semana", "compartilhando uma história engraçada"},
	"Food":              {"ordering at a restaurant together", "cooking a meal with a friend", "trying street food at a festival", "sharing recipes from your country"},
	"Weekend":           {"planning weekend activities together", "talking about last weekend's adventure", "comparing typical weekends", "suggesting fun weekend ideas"},
	"Shopping":          {"shopping at a local market", "buying a birthday gift for a friend", "comparing prices online vs in-store", "looking for souvenirs while traveling", "fare shopping al mercato locale", "comprare un regalo di compleanno", "confrontare prezzi online e in negozio", "cercare souvenir in viaggio"},
	"Weather":           {"deciding whether to go out based on weather", "talking about seasonal weather differences", "planning an outdoor event and checking weather", "comparing weather in different cities"},
	// Italian
	"Saluti":         {"incontro al bar per la prima volta", "incontro casuale al parco", "inizio conversazione a una festa", "saluto a un nuovo vicino"},
	"Presentazione":  {"primo giorno in una nuova classe", "incontro a un evento di scambio linguistico", "presentazione in una riunione", "chiacchierata in videochiamata"},
	"Famiglia":       {"mostrare foto di famiglia a un amico", "parlare di un viaggio in famiglia", "discutere tradizioni familiari", "confrontare famiglie con un amico"},
	"Hobby":          {"consigliare hobby a vicenda", "parlare di un nuovo hobby", "discutere attività del fine settimana", "condividere una storia divertente"},
	"Cibo":           {"ordinare al ristorante insieme", "cucinare con un amico", "provare cibo di strada a un festival", "condividere ricette del proprio paese"},
	"Fine settimana": {"pianificare attività del weekend", "raccontare l'avventura dello scorso weekend", "confrontare weekend tipici", "suggerire idee divertenti"},
	"Tempo":          {"decidere se uscire in base al meteo", "parlare delle differenze stagionali", "pianificare un evento all'aperto", "confrontare il tempo in diverse città"},
	// Korean
	"인사":     {"카페에서 처음 만나기", "공원에서 우연히 만나기", "파티에서 대화 시작하기", "새 이웃에게 인사하기"},
	"자기소개": {"새 수업 첫날", "언어 교환 이벤트에서 만나기", "회의에서 자기소개", "온라인 영상통화에서 대화"},
	"가족":     {"친구에게 가족 사진 보여주기", "가족 여행 이야기", "가족 전통 이야기", "친구와 가족 비교하기"},
	"취미":     {"서로 취미 추천하기", "새로 시작한 취미 이야기", "주말 활동 이야기", "재미있는 취미 이야기 나누기"},
	"음식":     {"식당에서 함께 주문하기", "친구와 요리하기", "축제에서 길거리 음식 먹기", "나라별 레시피 공유"},
	"주말":     {"주말 계획 세우기", "지난 주말 이야기", "평소 주말 비교하기", "재미있는 주말 아이디어 제안"},
	"쇼핑":     {"동네 시장에서 쇼핑", "친구 생일 선물 사기", "온라인과 오프라인 가격 비교", "여행 중 기념품 찾기"},
	"날씨":     {"날씨 보고 외출 결정하기", "계절별 날씨 차이 이야기", "야외 이벤트 계획하기", "다른 도시 날씨 비교"},
	// Portuguese
	"Saudações":      {"encontro num café pela primeira vez", "encontro casual no parque", "início de conversa numa festa", "cumprimentar um novo vizinho"},
	"Apresentação":   {"primeiro dia numa nova aula", "encontro num evento de intercâmbio", "apresentação numa reunião", "conversa por videochamada"},
	"Família":        {"mostrando fotos da família para um amigo", "falando sobre viagem em família", "discutindo tradições familiares", "comparando famílias com um amigo"},
	"Comida":         {"pedindo juntos num restaurante", "cozinhando com um amigo", "experimentando comida de rua num festival", "compartilhando receitas do seu país"},
	"Fim de semana":  {"planejando atividades de fim de semana", "contando sobre a aventura do último fim de semana", "comparando fins de semana típicos", "sugerindo ideias divertidas"},
	"Compras":        {"fazendo compras no mercado local", "comprando presente de aniversário", "comparando preços online e na loja", "procurando souvenirs viajando"},
	"Clima":          {"decidindo se sai com base no clima", "falando sobre diferenças sazonais", "planejando evento ao ar livre", "comparando clima em cidades diferentes"},
	// Japanese
	"あいさつ":  {"カフェで初めて会う", "公園で偶然会う", "パーティーで話しかける", "新しい隣人に挨拶する"},
	"自己紹介":  {"新しいクラスの初日", "言語交換イベントで会う", "会議で自己紹介する", "オンラインビデオ通話で話す"},
	"家族":     {"友達に家族の写真を見せる", "家族旅行の話をする", "家族の伝統について話す", "友達と家族を比べる"},
	"趣味":     {"お互いの趣味をおすすめする", "新しく始めた趣味について話す", "週末の活動を話す", "面白い趣味の話を共有する"},
	"食べ物":    {"レストランで一緒に注文する", "友達と料理する", "お祭りで屋台の食べ物を食べる", "自分の国のレシピを共有する"},
	"週末":     {"週末の計画を立てる", "先週末の冒険を話す", "普段の週末を比べる", "楽しい週末のアイデアを提案する"},
	"買い物":    {"地元の市場で買い物する", "友達の誕生日プレゼントを買う", "ネットと店舗の価格を比べる", "旅行中にお土産を探す"},
	"天気":     {"天気を見て外出を決める", "季節ごとの天気の違いを話す", "屋外イベントの計画を立てる", "違う都市の天気を比べる"},
}

// pickRandom returns a random element from a slice.
func pickRandom[T any](items []T) T {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(items))))
	return items[n.Int64()]
}

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
	"ja": {
		partnerIntro:     "You are a friendly Japanese conversation partner for beginners learning Japanese.",
		levelInstruction: "Use simple Japanese appropriate for level %d (1-5). Speak in Japanese. Use polite form (です/ます).",
		levelGuidelines: `Level Guidelines:
- Level 1: Single words, very short phrases. Ask yes/no questions. Example: "すしが好きですか？" → expect "はい"
- Level 2: Basic sentences (です / ます form). Ask simple what/where questions. Example: "何が好きですか？" → expect "すしが好きです。"
- Level 3: Add reasons. Ask "どうして" questions. Example: "どうして好きですか？" → expect "おいしいからです。"
- Level 4: 2-3 sentences. Encourage elaboration. Example: "もっと教えてください。"
- Level 5: Free-form discussion with natural transitions. Mix casual and polite forms.`,
		interpretErrorExamples: `Common pronunciation errors when non-native speakers speak Japanese (STT transcription):
- Long vowel confusion: "おばさん" vs "おばあさん", "おじさん" vs "おじいさん"
- っ (double consonant) missed: "きて" → "きって", "かた" → "かった"
- Pitch accent errors causing different word recognition
- は particle read as "ha" instead of "wa"
- を particle read as "wo" instead of "o"
- ん confused with other sounds
- ず/づ and じ/ぢ confusion`,
		interpretOutputLang:       "Japanese",
		feedbackConversationLabel: "Japanese conversation",
		feedbackLevelLabels: [5]string{
			"単語・超短文レベル",
			"基本文レベル",
			"理由追加レベル",
			"複数文レベル",
			"自発展開レベル",
		},
		hintLevelDesc: "Japanese language learner",
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
// level, turn progress, target language, a random persona, and a random
// conversation scenario to ensure variety across sessions on the same theme.
// pastTopics lists summaries of previous sessions on the same theme so the AI
// can steer the conversation toward unexplored areas.
func BuildSystemPrompt(theme domain.Theme, level int, turnNumber, maxTurns int, targetLang string, pastTopics []string) string {
	cfg := langConfig(targetLang)
	targetPhrases := string(theme.TargetPhrases)

	// Pick a random persona and scenario for this session.
	persona := pickRandom(personas)
	scenario := ""
	if scenarios, ok := scenariosByTheme[theme.Title]; ok && len(scenarios) > 0 {
		scenario = pickRandom(scenarios)
	}

	var sb strings.Builder
	sb.WriteString(cfg.partnerIntro + "\n\n")

	// Persona
	sb.WriteString(fmt.Sprintf("Your name is %s. You are %s.\n", persona.Name, persona.Personality))
	sb.WriteString("Introduce yourself naturally when the conversation starts.\n\n")

	// Scenario
	if scenario != "" {
		sb.WriteString(fmt.Sprintf("Conversation scenario: %s\n", scenario))
		sb.WriteString("Use this scenario to guide the conversation naturally. Ask questions related to this situation.\n\n")
	}

	sb.WriteString("Rules:\n")
	sb.WriteString("- Always speak first in each turn\n")
	sb.WriteString(fmt.Sprintf("- %s\n", fmt.Sprintf(cfg.levelInstruction, level)))
	sb.WriteString("- Keep responses to 1-2 sentences\n")
	sb.WriteString(fmt.Sprintf("- Topic: %s\n", theme.Title))
	sb.WriteString(fmt.Sprintf("- Target phrases: %s\n", targetPhrases))
	sb.WriteString("- If the user struggles, simplify your language\n")
	sb.WriteString("- Never correct grammar directly during conversation\n")
	sb.WriteString("- Be encouraging and patient\n")
	sb.WriteString("- Ask varied and creative questions — do NOT repeat the same questions across sessions\n")
	sb.WriteString("- Share your own opinions and experiences to make the conversation feel natural\n\n")

	sb.WriteString(cfg.levelGuidelines + "\n\n")

	// Past session context to avoid repetition
	if len(pastTopics) > 0 {
		sb.WriteString("The student has practiced this theme before. Topics already covered:\n")
		for _, topic := range pastTopics {
			sb.WriteString(fmt.Sprintf("- %s\n", topic))
		}
		sb.WriteString("Please explore DIFFERENT aspects of the topic this time. Ask new questions and take the conversation in a fresh direction.\n\n")
	}

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
  "natural_expressions": [{"original": "COPY the student's EXACT words here (%s only, NOT Japanese)", "natural": "how a native speaker would say the same thing (%s only, NOT Japanese)"}],
  "improvements": [{"point": "improvement point in Japanese", "example": "a correct example sentence in %s (NOT Japanese)"}],
  "conversation_tips": [{"situation": "describe in Japanese when in the conversation this tip applies", "native_would_say": "what a native speaker would say in this situation (in %s, NOT Japanese)", "explanation": "brief Japanese explanation of why natives say this"}],
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
- natural_expressions: CRITICAL LANGUAGE RULE — Both "original" and "natural" MUST be in %s. NEVER write Japanese in these fields.
  "original" = copy the student's exact words from the conversation.
  "natural" = how a NATIVE SPEAKER would express the same idea in %s. Do NOT just fix grammar — show a completely natural, idiomatic way a native would phrase it. The goal is to teach natural expression patterns, not just correct mistakes.
  Only include when the native version would be meaningfully different. Return [] if the student already sounds natural.
- improvements: max 2 items. "point" is in Japanese. "example" MUST be in %s (NEVER Japanese). Be gentle, no negative language.
- conversation_tips: 2-3 tips showing what a native speaker would say in specific moments of THIS conversation. Look at the whole conversation flow and find moments where a native would respond differently — not just grammar, but conversation style, reactions, follow-up questions, humor, filler words, etc. "situation" is in Japanese, "native_would_say" MUST be in %s (NEVER Japanese), "explanation" is in Japanese.
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
		cfg.interpretOutputLang, cfg.interpretOutputLang, // JSON template: natural_expressions original, natural
		cfg.interpretOutputLang,                          // JSON template: improvements example
		cfg.interpretOutputLang,                          // JSON template: conversation_tips native_would_say
		cfg.interpretOutputLang, cfg.interpretOutputLang, // Rules: natural_expressions (2 mentions)
		cfg.interpretOutputLang,                          // Rules: improvements example
		cfg.interpretOutputLang,                          // Rules: conversation_tips native_would_say
		labels[0], labels[1], labels[2], labels[3], labels[4],
		conversation.String(),
	)
}
