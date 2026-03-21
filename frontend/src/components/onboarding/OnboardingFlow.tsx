import { useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { Mic, Lightbulb, ArrowRight } from 'lucide-react'

interface OnboardingFlowProps {
  onComplete: () => void
}

type StepId = 0 | 1 | 2

const TOTAL_STEPS = 3

/**
 * Slide variants for AnimatePresence step transitions.
 * Entering slides in from the right; exiting slides out to the left.
 */
const slideVariants = {
  enter: { x: '100%', opacity: 0 },
  center: { x: 0, opacity: 1 },
  exit: { x: '-100%', opacity: 0 },
}

const slideTransition = { duration: 0.3, ease: 'easeInOut' as const }

export function OnboardingFlow({ onComplete }: OnboardingFlowProps) {
  const [step, setStep] = useState<StepId>(0)
  const [micError, setMicError] = useState<string | null>(null)
  const [isRequestingMic, setIsRequestingMic] = useState(false)

  const goToStep = (next: StepId) => {
    setStep(next)
  }

  /**
   * Request microphone permission.
   * Immediately closes the stream after obtaining it — we only need the permission grant.
   * On denial, shows an error message and still allows the user to proceed.
   */
  const handleMicRequest = async () => {
    setIsRequestingMic(true)
    setMicError(null)

    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      // Close the stream immediately — we only needed the permission grant
      stream.getTracks().forEach((track) => track.stop())
      goToStep(2)
    } catch {
      setMicError(
        'マイクが許可されませんでした。設定から許可してください。テキスト入力でも会話できます。',
      )
    } finally {
      setIsRequestingMic(false)
    }
  }

  /** Mark onboarding as completed and invoke the parent callback. */
  const handleComplete = () => {
    localStorage.setItem('onboarding_completed', 'true')
    onComplete()
  }

  return (
    <div className="fixed inset-0 bg-neutral-50 flex flex-col items-center justify-center overflow-hidden">
      {/* Step content with slide animation */}
      <div className="relative w-full flex-1 flex items-center justify-center overflow-hidden px-6">
        <AnimatePresence mode="wait" initial={false}>
          {step === 0 && (
            <motion.div
              key="step-0"
              variants={slideVariants}
              initial="enter"
              animate="center"
              exit="exit"
              transition={slideTransition}
              className="absolute w-full px-6 flex flex-col items-center gap-6 text-center"
            >
              {/* Step 1: Welcome */}
              <div className="w-16 h-16 rounded-full bg-primary-100 flex items-center justify-center">
                <ArrowRight size={28} className="text-primary-600" />
              </div>
              <h1 className="text-2xl font-bold text-neutral-900">AIと英語で話してみよう</h1>
              <p className="text-neutral-600 text-center leading-relaxed">
                やさしい英語で会話する
                <br />
                10分だけの英会話です
              </p>
              <button
                onClick={() => goToStep(1)}
                className="bg-primary-600 text-white rounded-2xl h-14 w-full max-w-xs font-medium text-base"
              >
                次へ
              </button>
            </motion.div>
          )}

          {step === 1 && (
            <motion.div
              key="step-1"
              variants={slideVariants}
              initial="enter"
              animate="center"
              exit="exit"
              transition={slideTransition}
              className="absolute w-full px-6 flex flex-col items-center gap-6 text-center"
            >
              {/* Step 2: Microphone */}
              <div className="w-16 h-16 rounded-full bg-primary-100 flex items-center justify-center">
                <Mic size={28} className="text-primary-600" />
              </div>
              <h1 className="text-2xl font-bold text-neutral-900">マイクを使います</h1>
              <p className="text-neutral-600 text-center leading-relaxed">
                ボタンを押して
                <br />
                英語で話してみましょう
              </p>
              {micError && (
                <p className="text-sm text-recording text-center leading-relaxed px-2">
                  {micError}
                </p>
              )}
              <button
                onClick={handleMicRequest}
                disabled={isRequestingMic}
                className="bg-primary-600 text-white rounded-2xl h-14 w-full max-w-xs font-medium text-base disabled:opacity-60"
              >
                {isRequestingMic ? '許可を確認中...' : 'マイクを許可する'}
              </button>
              {micError && (
                <button
                  onClick={() => goToStep(2)}
                  className="text-sm text-neutral-600 underline"
                >
                  このまま次へ進む
                </button>
              )}
            </motion.div>
          )}

          {step === 2 && (
            <motion.div
              key="step-2"
              variants={slideVariants}
              initial="enter"
              animate="center"
              exit="exit"
              transition={slideTransition}
              className="absolute w-full px-6 flex flex-col items-center gap-6 text-center"
            >
              {/* Step 3: Hint */}
              <div className="w-16 h-16 rounded-full bg-primary-100 flex items-center justify-center">
                <Lightbulb size={28} className="text-primary-600" />
              </div>
              <h1 className="text-2xl font-bold text-neutral-900">
                困ったときは
                <br />
                ヒントボタンを押そう
              </h1>
              <p className="text-neutral-600 text-center leading-relaxed">
                答え方が分からなくても
                <br />
                大丈夫！
              </p>
              <button
                onClick={handleComplete}
                className="bg-primary-600 text-white rounded-2xl h-14 w-full max-w-xs font-medium text-base"
              >
                はじめる
              </button>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {/* Dot indicator showing current step */}
      <div className="flex items-center gap-2 pb-12" aria-label="ステップのインジケーター">
        {Array.from({ length: TOTAL_STEPS }).map((_, i) => (
          <span
            key={i}
            className={[
              'rounded-full transition-all duration-300',
              i === step
                ? 'w-6 h-2 bg-primary-600'
                : 'w-2 h-2 bg-neutral-300',
            ].join(' ')}
          />
        ))}
      </div>
    </div>
  )
}
