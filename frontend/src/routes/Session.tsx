import { useEffect, useRef, useCallback, useState } from 'react'
import { useParams, useNavigate, useLocation } from 'react-router'
import { ArrowLeft, Send } from 'lucide-react'
import { api } from '@/lib/api-client'
import { useSessionStore } from '@/stores/session-store'
import { useChat, type Message } from '@/hooks/useChat'
import { useAudioRecorder } from '@/hooks/useAudioRecorder'
import { useTTS } from '@/hooks/useTTS'
import { ChatBubble } from '@/components/chat/ChatBubble'
import { TypingIndicator } from '@/components/chat/TypingIndicator'
import { VoiceInputButton } from '@/components/chat/VoiceInputButton'
import { RescuePanel } from '@/components/chat/RescuePanel'
import { LoadingScreen } from '@/components/common/LoadingSpinner'

interface SessionData {
  id: string
  theme_id: string
  theme_title?: string
  status?: string
  max_turns: number
}

interface TurnData {
  ai_text: string
  user_text?: string | null
}

interface LocationState {
  themeId?: string
  themeTitle?: string
  maxTurns?: number
  targetLanguage?: string
}

export default function Session() {
  const { id: sessionId } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const locationState = location.state as LocationState | null

  const {
    phase,
    themeTitle,
    turnNumber,
    maxTurns,
    setPhase,
    startSession,
    incrementTurn,
    reset,
  } = useSessionStore()

  const { messages, isStreaming, sendMessage, addUserMessage, setMessages } = useChat()
  const { isRecording, isProcessing, startRecording, stopRecording, error: recorderError } = useAudioRecorder()
  const { isPlaying, play: playTTS } = useTTS()

  const chatEndRef = useRef<HTMLDivElement>(null)
  const initializedRef = useRef(false)

  // Text input mode
  const [isTextMode, setIsTextMode] = useState(false)
  const [textInput, setTextInput] = useState('')

  // Auto-scroll
  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, isStreaming])

  // Complete the session and navigate to feedback
  const completeSession = useCallback(async () => {
    if (!sessionId) return
    setPhase('session_complete')
    try {
      const result = await api.put<{
        level_changed?: boolean
        previous_level?: number
        new_level?: number
      }>(`/api/v1/sessions/${sessionId}/complete`, {
        turn_count: useSessionStore.getState().turnNumber,
      })
      navigate(`/feedback/${sessionId}`, {
        state: {
          levelChanged: result.level_changed,
          previousLevel: result.previous_level,
          newLevel: result.new_level,
        },
      })
    } catch {
      navigate(`/feedback/${sessionId}`)
    }
  }, [sessionId, setPhase, navigate])

  // Send message → stream AI response → play TTS → advance phase
  const runAITurn = useCallback(
    async (userText: string): Promise<void> => {
      if (!sessionId) return

      // Step 1: Interpret the user's text to detect pronunciation errors.
      // This is non-blocking: any failure falls back to the original text.
      let interpretedText = userText
      if (userText.trim()) {
        try {
          const result = await api.post<{ interpreted: string; is_different: boolean }>(
            '/api/v1/chat/interpret',
            { session_id: sessionId, raw_text: userText },
          )
          if (result.is_different && result.interpreted) {
            interpretedText = result.interpreted
            // Update the last user message in state to show the interpretation
            setMessages((prev) => {
              const updated = [...prev]
              const lastUserIdx = updated.findLastIndex((m) => m.role === 'user')
              if (lastUserIdx >= 0) {
                updated[lastUserIdx] = {
                  ...updated[lastUserIdx],
                  interpretedText: result.interpreted,
                  isInterpreted: true,
                }
              }
              return updated
            })
          }
        } catch {
          // Interpret failure is non-blocking; continue with raw text
        }
      }

      // Step 2: Send to AI (passing the interpreted text so it can respond
      // to the intended meaning rather than the phonetically garbled text).
      setPhase('ai_thinking')
      let aiText = ''
      try {
        aiText = await sendMessage(
          sessionId,
          userText,
          interpretedText !== userText ? interpretedText : undefined,
        )
      } catch {
        setPhase('waiting_user')
        return
      }

      // Play the AI response via TTS (skip if empty)
      if (aiText.trim()) {
        setPhase('ai_speaking')
        try {
          await playTTS(aiText)
        } catch {
          // TTS failure is non-blocking
        }
      }

      incrementTurn()

      // Use getState() to get the latest turnNumber after incrementTurn
      const currentTurn = useSessionStore.getState().turnNumber
      if (currentTurn >= maxTurns) {
        completeSession()
      } else {
        setPhase('waiting_user')
      }
    },
    [sessionId, sendMessage, setMessages, playTTS, incrementTurn, maxTurns, setPhase, completeSession],
  )

  // Initialise the session
  useEffect(() => {
    if (!sessionId || initializedRef.current) return
    initializedRef.current = true

    const init = async () => {
      // Fetch session data
      let session: SessionData | null = null
      try {
        session = await api.get<SessionData>(`/api/v1/sessions/${sessionId}`)
      } catch {
        navigate('/')
        return
      }

      // If session is already completed, redirect to feedback
      if (session.status === 'completed') {
        navigate(`/feedback/${sessionId}`, { replace: true })
        return
      }

      const themeId = locationState?.themeId ?? session.theme_id
      const title = locationState?.themeTitle ?? session.theme_title ?? ''
      // Prefer max_turns from location state (set at session creation), then
      // fall back to the value returned by the API, then the store default.
      const maxTurnsValue = locationState?.maxTurns ?? session.max_turns

      // Fetch existing turns to restore conversation
      let existingTurns: TurnData[] = []
      try {
        existingTurns = await api.get<TurnData[]>(`/api/v1/sessions/${sessionId}/turns`)
      } catch {
        // If fetching turns fails, start fresh
      }

      // Restore messages from existing turns
      if (existingTurns.length > 0) {
        const restored: Message[] = []
        for (const turn of existingTurns) {
          if (turn.ai_text) {
            restored.push({ role: 'ai', text: turn.ai_text })
          }
          if (turn.user_text) {
            restored.push({ role: 'user', text: turn.user_text })
          }
        }
        setMessages(restored)
        startSession(sessionId, themeId, title, maxTurnsValue)
        // Set turnNumber to match existing turns
        const store = useSessionStore.getState()
        for (let i = 0; i < existingTurns.length; i++) {
          store.incrementTurn()
        }
        setPhase('waiting_user')
      } else {
        // New session: trigger the first AI message
        startSession(sessionId, themeId, title, maxTurnsValue)
        await runAITurn('')
      }
    }

    init()

    return () => {
      reset()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionId])

  // Voice input flow
  const handleVoicePress = async () => {
    if (phase === 'waiting_user' && !isRecording) {
      setPhase('recording')
      await startRecording()
      return
    }

    if (phase === 'recording' && isRecording) {
      setPhase('transcribing')
      const audioBlob = await stopRecording()

      if (!audioBlob) {
        setPhase('waiting_user')
        return
      }

      const formData = new FormData()
      formData.append('audio', audioBlob, 'recording.webm')
      // Pass the target language as a hint so Whisper skips auto-detection
      const targetLanguage = locationState?.targetLanguage ?? 'en'
      formData.append('language', targetLanguage)

      let userText = ''
      try {
        const sttResult = await api.postForm<{ text: string }>(
          '/api/v1/speech/stt',
          formData,
        )
        userText = sttResult.text?.trim() ?? ''
      } catch {
        setPhase('waiting_user')
        return
      }

      if (!userText) {
        setPhase('waiting_user')
        return
      }

      addUserMessage(userText)
      await runAITurn(userText)
    }
  }

  // Text submit
  const handleTextSubmit = async () => {
    const userText = textInput.trim()
    if (!userText || phase !== 'waiting_user') return

    setTextInput('')
    addUserMessage(userText)
    await runAITurn(userText)
  }

  const voiceButtonState = (() => {
    if (phase === 'recording') return 'recording' as const
    if (phase === 'transcribing' || isProcessing) return 'processing' as const
    return 'idle' as const
  })()

  const isVoiceDisabled =
    phase !== 'waiting_user' &&
    phase !== 'recording'

  // Loading screen until session is ready
  const isInitialising = phase === 'idle' || (phase === 'ai_thinking' && messages.length === 0)
  if (isInitialising) {
    return <LoadingScreen />
  }

  const displayTitle = themeTitle ?? locationState?.themeTitle ?? 'セッション'

  return (
    <div className="flex flex-col h-svh bg-neutral-50">
      {/* Header */}
      <header className="flex items-center justify-between px-4 py-3 border-b border-neutral-100 bg-white flex-shrink-0">
        <button
          onClick={() => navigate('/')}
          aria-label="セッションを終了して戻る"
          className="p-1 -ml-1 rounded-full text-neutral-600 hover:text-neutral-900 transition-colors"
        >
          <ArrowLeft size={22} />
        </button>

        <div className="flex flex-col items-center">
          <span className="text-sm font-semibold text-neutral-900 leading-tight">
            {displayTitle}
          </span>
          <span className="text-xs text-neutral-400">
            {turnNumber} / {maxTurns}
          </span>
        </div>

        <div className="w-8" />
      </header>

      {/* Chat area */}
      <div className="flex-1 overflow-y-auto px-4 py-4 flex flex-col gap-3">
        {messages.map((msg, i) => (
          <ChatBubble
            key={i}
            variant={msg.role}
            text={msg.text}
            interpretedText={msg.interpretedText}
            isInterpreted={msg.isInterpreted}
          />
        ))}

        {phase === 'ai_thinking' && messages[messages.length - 1]?.text === '' && (
          <TypingIndicator />
        )}

        <div ref={chatEndRef} />
      </div>

      {/* Rescue panel */}
      <div className="flex-shrink-0 border-t border-neutral-100 bg-white">
        <RescuePanel
          sessionId={sessionId ?? ''}
          turnNumber={turnNumber}
          disabled={isVoiceDisabled || isStreaming}
        />
      </div>

      {/* Voice input or text input */}
      <div className="flex-shrink-0 flex flex-col items-center py-5 bg-white">
        {isTextMode ? (
          <>
            <div className="flex items-center gap-2 w-full max-w-sm px-4">
              <input
                type="text"
                value={textInput}
                onChange={(e) => setTextInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !e.nativeEvent.isComposing) {
                    handleTextSubmit()
                  }
                }}
                placeholder={`${{ ko: '韓国語', it: 'イタリア語', pt: 'ポルトガル語', ja: '日本語', en: '英語' }[locationState?.targetLanguage ?? 'en'] ?? '英語'}で入力...`}
                disabled={phase !== 'waiting_user'}
                className="flex-1 border border-neutral-300 rounded-2xl px-4 h-12 text-sm text-neutral-900 placeholder:text-neutral-400 focus:outline-none focus:border-primary-600 disabled:opacity-50 bg-white"
              />
              <button
                onClick={handleTextSubmit}
                disabled={!textInput.trim() || phase !== 'waiting_user'}
                aria-label="送信"
                className="w-12 h-12 rounded-full bg-primary-600 text-white flex items-center justify-center flex-shrink-0 disabled:opacity-50"
              >
                <Send size={18} />
              </button>
            </div>
            {phase === 'ai_speaking' && isPlaying && (
              <p className="mt-2 text-xs text-neutral-400">AIが話しています...</p>
            )}
            <button
              onClick={() => setIsTextMode(false)}
              className="mt-3 text-sm text-neutral-600 underline"
            >
              音声で入力
            </button>
          </>
        ) : (
          <>
            {recorderError && (
              <p className="text-xs text-recording mb-2 px-4 text-center">{recorderError}</p>
            )}
            <VoiceInputButton
              state={voiceButtonState}
              disabled={isVoiceDisabled}
              onPress={handleVoicePress}
            />
            {phase === 'ai_speaking' && isPlaying && (
              <p className="mt-2 text-xs text-neutral-400">AIが話しています...</p>
            )}
            <button
              onClick={() => setIsTextMode(true)}
              className="mt-3 text-sm text-neutral-600 underline"
            >
              テキストで入力
            </button>
          </>
        )}
      </div>
    </div>
  )
}
