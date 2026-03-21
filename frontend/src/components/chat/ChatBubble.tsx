import { Volume2 } from 'lucide-react'
import { useTTS } from '@/hooks/useTTS'

interface ChatBubbleProps {
  variant: 'ai' | 'user'
  text: string
  /** Pronunciation-corrected interpretation of the user's text */
  interpretedText?: string
  /** Whether interpretedText differs from text and should be displayed */
  isInterpreted?: boolean
}

/**
 * Renders a single chat message bubble.
 * AI bubbles are left-aligned with a TTS play button.
 * User bubbles are right-aligned with a teal background.
 *
 * When a user bubble has isInterpreted=true, the interpreted text is shown
 * below the raw text in a subtle style so the user can see what the AI
 * understood without feeling embarrassed about their pronunciation.
 */
export function ChatBubble({ variant, text, interpretedText, isInterpreted }: ChatBubbleProps) {
  const { isPlaying, play, stop } = useTTS()

  const handlePlayToggle = () => {
    if (isPlaying) {
      stop()
    } else {
      play(text).catch(() => {
        // Ignore playback errors silently; they are non-critical UX
      })
    }
  }

  if (variant === 'ai') {
    return (
      <div className="flex items-end gap-2 self-start max-w-[80%]">
        <div className="bg-ai-bubble rounded-2xl px-4 py-3 text-neutral-900 text-sm leading-relaxed">
          {text}
        </div>
        {text && (
          <button
            onClick={handlePlayToggle}
            aria-label={isPlaying ? '再生を停止' : '音声を再生'}
            className="flex-shrink-0 p-1.5 rounded-full text-neutral-400 hover:text-primary-600 hover:bg-primary-50 transition-colors"
          >
            <Volume2 size={16} className={isPlaying ? 'text-primary-600' : ''} />
          </button>
        )}
      </div>
    )
  }

  return (
    <div className="self-end max-w-[80%]">
      <div className="bg-user-bubble text-white rounded-2xl px-4 py-3 text-sm leading-relaxed">
        <span>{text}</span>
        {isInterpreted && interpretedText && (
          <p className="mt-1 text-xs text-white/60 italic">{interpretedText}</p>
        )}
      </div>
    </div>
  )
}
