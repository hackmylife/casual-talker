import { Mic, Loader2 } from 'lucide-react'
import { motion } from 'framer-motion'

type VoiceInputState = 'idle' | 'recording' | 'processing'

interface VoiceInputButtonProps {
  state: VoiceInputState
  disabled?: boolean
  onPress: () => void
}

/**
 * Large circular voice input button with three visual states:
 * - idle: microphone icon, primary teal
 * - recording: pulsing red with "聞いています..."
 * - processing: spinner with "聞き取り中..."
 */
export function VoiceInputButton({ state, disabled = false, onPress }: VoiceInputButtonProps) {
  const isDisabled = disabled || state === 'processing'

  return (
    <div className="flex flex-col items-center gap-2">
      <motion.button
        onClick={onPress}
        disabled={isDisabled}
        aria-label={
          state === 'recording'
            ? '録音を停止する'
            : state === 'processing'
              ? '音声を処理中'
              : '話す'
        }
        className={[
          'w-16 h-16 rounded-full flex items-center justify-center transition-colors shadow-md',
          state === 'recording'
            ? 'bg-recording text-white'
            : 'bg-primary-600 text-white',
          isDisabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer',
        ].join(' ')}
        // Pulse animation during recording
        animate={
          state === 'recording'
            ? { scale: [1, 1.12, 1] }
            : { scale: 1 }
        }
        transition={
          state === 'recording'
            ? { duration: 1, repeat: Infinity, ease: 'easeInOut' }
            : { duration: 0.15 }
        }
      >
        {state === 'processing' ? (
          <Loader2 size={26} className="animate-spin" />
        ) : (
          <Mic size={26} />
        )}
      </motion.button>

      <span className="text-xs text-neutral-600 min-h-[1em]">
        {state === 'recording'
          ? '聞いています...'
          : state === 'processing'
            ? '聞き取り中...'
            : disabled
              ? ''
              : '話す'}
      </span>
    </div>
  )
}
