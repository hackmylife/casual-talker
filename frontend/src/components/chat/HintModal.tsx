import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { X } from 'lucide-react'

interface HintData {
  hint: string
  japanese: string
  sample_answer: string
}

interface HintModalProps {
  data: HintData
  onClose: () => void
}

type HintStep = 1 | 2 | 3

const STEP_LABELS: Record<HintStep, string> = {
  1: '日本語ヒント',
  2: '英語キーワード',
  3: '完全な回答例',
}

/**
 * Bottom-sheet style modal that progressively reveals hint content in three steps:
 *  1. Japanese hint (japanese field)
 *  2. English keywords (hint field)
 *  3. Full sample answer (sample_answer field)
 */
export function HintModal({ data, onClose }: HintModalProps) {
  const [step, setStep] = useState<HintStep>(1)

  const handleNext = () => {
    if (step < 3) {
      setStep((prev) => (prev + 1) as HintStep)
    } else {
      onClose()
    }
  }

  return (
    <AnimatePresence>
      {/* Backdrop */}
      <motion.div
        key="backdrop"
        className="fixed inset-0 bg-black/40 z-40"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        onClick={onClose}
      />

      {/* Sheet */}
      <motion.div
        key="sheet"
        className="fixed bottom-0 left-0 right-0 z-50 bg-white rounded-t-3xl px-6 pt-6 pb-10 shadow-xl"
        initial={{ y: '100%' }}
        animate={{ y: 0 }}
        exit={{ y: '100%' }}
        transition={{ type: 'spring', damping: 30, stiffness: 300 }}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between mb-4">
          <span className="text-xs font-medium text-neutral-400 uppercase tracking-wider">
            {STEP_LABELS[step]}
          </span>
          <button
            onClick={onClose}
            aria-label="閉じる"
            className="p-1 rounded-full text-neutral-400 hover:text-neutral-600 transition-colors"
          >
            <X size={20} />
          </button>
        </div>

        {/* Step indicator dots */}
        <div className="flex gap-1.5 mb-5">
          {([1, 2, 3] as HintStep[]).map((s) => (
            <span
              key={s}
              className={[
                'h-1.5 rounded-full transition-all duration-200',
                s <= step ? 'bg-primary-600 w-5' : 'bg-neutral-200 w-1.5',
              ].join(' ')}
            />
          ))}
        </div>

        {/* Content */}
        <AnimatePresence mode="wait">
          <motion.div
            key={step}
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -8 }}
            transition={{ duration: 0.2 }}
            className="min-h-[5rem] mb-6"
          >
            {step === 1 && (
              <p className="text-neutral-800 text-base leading-relaxed">{data.japanese}</p>
            )}
            {step === 2 && (
              <p className="text-neutral-800 text-base leading-relaxed font-en">{data.hint}</p>
            )}
            {step === 3 && (
              <div className="bg-hint rounded-2xl px-4 py-3">
                <p className="text-neutral-800 text-base leading-relaxed font-en">
                  {data.sample_answer}
                </p>
              </div>
            )}
          </motion.div>
        </AnimatePresence>

        {/* Action button */}
        <button
          onClick={handleNext}
          className="w-full py-3 rounded-2xl bg-primary-600 text-white font-medium text-sm active:bg-primary-700 transition-colors"
        >
          {step < 3 ? '次のヒント' : '閉じる'}
        </button>
      </motion.div>
    </AnimatePresence>
  )
}
