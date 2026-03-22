import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router'
import { Check, Lightbulb, Bookmark, Volume2 } from 'lucide-react'
import { api } from '@/lib/api-client'
import { LoadingSpinner } from '@/components/common/LoadingSpinner'
import { ExpressionCard } from '@/components/feedback/ExpressionCard'

// Shape returned by GET /api/v1/sessions/:id/feedback
interface NaturalExpression {
  original: string
  natural: string
}

interface CurrentLevel {
  level: number
  label: string
  description: string
}

interface FeedbackData {
  id: string
  session_id: string
  achievements: string[]
  natural_expressions: NaturalExpression[] | string[]
  improvements: (string | { point: string; example: string })[]
  review_phrases: string[]
  current_level?: CurrentLevel
  next_level_advice?: string
}

// Shape returned by GET /api/v1/sessions/:id/turns
interface TurnData {
  id: string
  turn_number: number
  ai_text: string
  user_text?: string | null
  interpreted_text?: string | null
}

// A pair of user's raw text and the corrected interpretation
interface PronunciationItem {
  original: string
  corrected: string
}

// Parse natural_expressions which may be:
//   - an array of {original, natural} objects
//   - an array of plain strings (fallback)
function parseNaturalExpressions(
  raw: NaturalExpression[] | string[],
): NaturalExpression[] {
  if (!Array.isArray(raw) || raw.length === 0) return []

  const first = raw[0]
  if (typeof first === 'object' && first !== null && 'original' in first) {
    return raw as NaturalExpression[]
  }

  // Each element is a JSON string or a plain string — try to parse as JSON
  return (raw as string[]).flatMap((item) => {
    try {
      const parsed = JSON.parse(item) as unknown
      if (
        parsed !== null &&
        typeof parsed === 'object' &&
        'original' in (parsed as object) &&
        'natural' in (parsed as object)
      ) {
        return [parsed as NaturalExpression]
      }
    } catch {
      // Not valid JSON — treat the whole string as a plain expression with no pair
    }
    return []
  })
}

// Level badge colors per level number (1-5)
function levelColor(level: number): string {
  switch (level) {
    case 1: return 'bg-neutral-100 text-neutral-700'
    case 2: return 'bg-blue-50 text-blue-700'
    case 3: return 'bg-green-50 text-green-700'
    case 4: return 'bg-purple-50 text-purple-700'
    case 5: return 'bg-amber-50 text-amber-700'
    default: return 'bg-neutral-100 text-neutral-700'
  }
}

export default function Feedback() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [feedback, setFeedback] = useState<FeedbackData | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Pronunciation practice state
  const [turns, setTurns] = useState<TurnData[]>([])
  const [pronunciationItems, setPronunciationItems] = useState<PronunciationItem[]>([])
  const [playingIndex, setPlayingIndex] = useState<number | null>(null)
  const [showLog, setShowLog] = useState(false)

  useEffect(() => {
    if (!id) return

    let cancelled = false

    const load = async () => {
      try {
        // Fetch feedback and turns in parallel
        const [data, turns] = await Promise.all([
          api.get<FeedbackData>(`/api/v1/sessions/${id}/feedback`),
          api.get<TurnData[]>(`/api/v1/sessions/${id}/turns`).catch(() => [] as TurnData[]),
        ])

        if (!cancelled) {
          setFeedback(data)
          setTurns(turns)

          // Extract turns where the interpreted text differs from raw user text
          const items: PronunciationItem[] = turns
            .filter(
              (t) =>
                t.user_text &&
                t.interpreted_text &&
                t.interpreted_text !== t.user_text,
            )
            .map((t) => ({
              original: t.user_text!,
              corrected: t.interpreted_text!,
            }))
          setPronunciationItems(items)
        }
      } catch {
        if (!cancelled) {
          setError('フィードバックの取得に失敗しました')
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false)
        }
      }
    }

    load()
    return () => {
      cancelled = true
    }
  }, [id])

  // Play TTS for a corrected phrase
  const handlePlayTTS = async (text: string, index: number) => {
    if (playingIndex !== null) return
    setPlayingIndex(index)
    try {
      const blob = await api.postBlob('/api/v1/speech/tts', { text })
      const url = URL.createObjectURL(blob)
      const audio = new Audio(url)
      audio.onended = () => {
        setPlayingIndex(null)
        URL.revokeObjectURL(url)
      }
      audio.onerror = () => {
        setPlayingIndex(null)
        URL.revokeObjectURL(url)
      }
      await audio.play()
    } catch {
      setPlayingIndex(null)
    }
  }

  // Feedback may still be generating on the server — show spinner
  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] gap-3">
        <LoadingSpinner size="lg" />
        <p className="text-sm text-neutral-600">フィードバックを生成中...</p>
      </div>
    )
  }

  if (error || !feedback) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] gap-4 px-4">
        <p className="text-neutral-600">{error ?? 'フィードバックが見つかりませんでした'}</p>
        <button
          onClick={() => navigate('/')}
          className="px-6 py-2 bg-primary-600 text-white rounded-2xl text-sm font-medium"
        >
          ホームに戻る
        </button>
      </div>
    )
  }

  const achievements = feedback.achievements ?? []
  const naturalExpressions = parseNaturalExpressions(feedback.natural_expressions ?? [])
  const rawNaturalStrings =
    Array.isArray(feedback.natural_expressions) &&
    feedback.natural_expressions.length > 0 &&
    typeof feedback.natural_expressions[0] === 'string' &&
    naturalExpressions.length === 0
      ? (feedback.natural_expressions as string[])
      : []
  const improvements = feedback.improvements ?? []
  const reviewPhrases = feedback.review_phrases ?? []
  const currentLevel = feedback.current_level
  const nextLevelAdvice = feedback.next_level_advice

  return (
    <div className="min-h-full bg-neutral-50 px-4 pt-6 pb-8">
      {/* Header */}
      <div className="text-center mb-8">
        <p className="text-4xl mb-2">🎉</p>
        <h1 className="text-2xl font-bold text-neutral-900">セッション完了！</h1>
        <p className="mt-1 text-neutral-600">お疲れさまでした</p>
      </div>

      <div className="space-y-6">
        {/* Current level section */}
        {currentLevel && currentLevel.level > 0 && (
          <section className="bg-white rounded-2xl p-4 shadow-sm">
            <h2 className="text-lg font-semibold text-neutral-900 mb-3">
              今のあなたのレベル
            </h2>
            <div className="flex items-center gap-3 mb-2">
              <span
                className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-semibold ${levelColor(currentLevel.level)}`}
              >
                Lv{currentLevel.level}　{currentLevel.label}
              </span>
            </div>
            <p className="text-sm text-neutral-600">{currentLevel.description}</p>
            {nextLevelAdvice && (
              <div className="mt-3 bg-primary-50 rounded-xl p-3">
                <p className="text-xs font-semibold text-primary-700 mb-1">次のステップ</p>
                <p className="text-sm text-primary-800">{nextLevelAdvice}</p>
              </div>
            )}
          </section>
        )}

        {/* Achievements section */}
        {achievements.length > 0 && (
          <section className="bg-white rounded-2xl p-4 shadow-sm">
            <h2 className="text-lg font-semibold text-neutral-900 mb-3 flex items-center gap-2">
              <span className="text-success">✅</span> できたこと
            </h2>
            <ul className="space-y-2">
              {achievements.map((item, index) => (
                <li key={index} className="flex items-start gap-2 text-sm text-neutral-800">
                  <Check size={16} className="text-success mt-0.5 flex-shrink-0" />
                  <span>{item}</span>
                </li>
              ))}
            </ul>
          </section>
        )}

        {/* Natural expressions section */}
        {(naturalExpressions.length > 0 || rawNaturalStrings.length > 0) && (
          <section>
            <h2 className="text-lg font-semibold text-neutral-900 mb-3">
              💡 こう言うともっと自然
            </h2>
            <div className="space-y-3">
              {naturalExpressions.length > 0
                ? naturalExpressions.map((expr, index) => (
                    <ExpressionCard
                      key={index}
                      original={expr.original}
                      natural={expr.natural}
                    />
                  ))
                : rawNaturalStrings.map((str, index) => (
                    <div key={index} className="bg-white rounded-2xl p-4 shadow-sm">
                      <p className="text-sm text-neutral-800">{str}</p>
                    </div>
                  ))}
            </div>
          </section>
        )}

        {/* Pronunciation practice section */}
        {pronunciationItems.length > 0 && (
          <section className="bg-white rounded-2xl p-4 shadow-sm">
            <h2 className="text-lg font-semibold text-neutral-900 mb-3">
              🎤 発音を練習
            </h2>
            <div className="space-y-3">
              {pronunciationItems.map((item, index) => (
                <div
                  key={index}
                  className="border border-neutral-100 rounded-xl p-3 flex items-center justify-between gap-3"
                >
                  <div className="flex flex-col gap-1 flex-1 min-w-0">
                    {/* Original (incorrect) text */}
                    <span className="text-sm text-red-500 line-through leading-snug">
                      {item.original}
                    </span>
                    {/* Corrected text */}
                    <span className="text-sm text-green-700 font-medium leading-snug">
                      {item.corrected}
                    </span>
                  </div>
                  {/* TTS play button */}
                  <button
                    onClick={() => handlePlayTTS(item.corrected, index)}
                    disabled={playingIndex !== null}
                    aria-label={`"${item.corrected}"を再生`}
                    className="flex-shrink-0 w-10 h-10 rounded-full bg-primary-50 text-primary-600 flex items-center justify-center active:bg-primary-100 transition-colors disabled:opacity-40"
                  >
                    {playingIndex === index ? (
                      <LoadingSpinner size="sm" />
                    ) : (
                      <Volume2 size={18} />
                    )}
                  </button>
                </div>
              ))}
            </div>
          </section>
        )}

        {/* Improvements section */}
        {improvements.length > 0 && (
          <section className="bg-white rounded-2xl p-4 shadow-sm">
            <h2 className="text-lg font-semibold text-neutral-900 mb-3">📝 改善ポイント</h2>
            <ul className="space-y-2">
              {improvements.map((item, index) => {
                const point = typeof item === 'string' ? item : item.point
                const example = typeof item === 'object' && item !== null ? item.example : null
                return (
                  <li key={index} className="flex items-start gap-2 text-sm text-neutral-800">
                    <Lightbulb size={16} className="text-secondary-500 mt-0.5 flex-shrink-0" />
                    <div>
                      <span>{point}</span>
                      {example && (
                        <p className="mt-1 text-xs text-primary-700 bg-primary-50 rounded-lg px-2 py-1 italic">
                          例: {example}
                        </p>
                      )}
                    </div>
                  </li>
                )
              })}
            </ul>
          </section>
        )}

        {/* Review phrases section */}
        {reviewPhrases.length > 0 && (
          <section className="bg-white rounded-2xl p-4 shadow-sm">
            <h2 className="text-lg font-semibold text-neutral-900 mb-3">🔖 復習フレーズ</h2>
            <ul className="space-y-2">
              {reviewPhrases.map((phrase, index) => (
                <li key={index} className="flex items-start gap-2 text-sm text-neutral-800">
                  <Bookmark size={16} className="text-primary-600 mt-0.5 flex-shrink-0" />
                  <span>{phrase}</span>
                </li>
              ))}
            </ul>
          </section>
        )}

        {/* Conversation log section */}
        {turns.length > 0 && (
          <section className="bg-white rounded-2xl p-4 shadow-sm">
            <button
              onClick={() => setShowLog(!showLog)}
              className="w-full flex items-center justify-between text-lg font-semibold text-neutral-900"
            >
              <span>💬 会話ログ</span>
              <span className="text-sm font-normal text-neutral-400">
                {showLog ? '閉じる' : '開く'}
              </span>
            </button>
            {showLog && (
              <div className="mt-3 space-y-2">
                {turns.map((turn) => (
                  <div key={turn.id}>
                    {/* AI message */}
                    <div className="flex gap-2 mb-1">
                      <span className="text-xs font-medium text-neutral-400 mt-0.5 flex-shrink-0 w-6">AI</span>
                      <p className="text-sm text-neutral-800 bg-neutral-50 rounded-xl px-3 py-2 flex-1">
                        {turn.ai_text}
                      </p>
                    </div>
                    {/* User message */}
                    {turn.user_text && (
                      <div className="flex gap-2 justify-end">
                        <div className="text-sm bg-primary-500 text-white rounded-xl px-3 py-2 max-w-[80%]">
                          <p>{turn.user_text}</p>
                          {turn.interpreted_text && turn.interpreted_text !== turn.user_text && (
                            <p className="text-xs text-white/60 italic mt-0.5">
                              {turn.interpreted_text}
                            </p>
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </section>
        )}

        {/* CTA button */}
        <button
          onClick={() => navigate('/')}
          className="w-full h-14 bg-primary-600 text-white rounded-2xl font-semibold text-base active:bg-primary-700 transition-colors"
        >
          ホームに戻る
        </button>
      </div>
    </div>
  )
}
