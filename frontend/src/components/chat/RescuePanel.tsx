import { useState } from 'react'
import { Lightbulb, BookOpen } from 'lucide-react'
import { api } from '@/lib/api-client'
import { HintModal } from './HintModal'

interface HintData {
  hint: string
  japanese: string
  sample_answer: string
}

interface RescuePanelProps {
  sessionId: string
  turnNumber: number
  disabled?: boolean
}

/**
 * Panel with "ヒント" and "回答例" buttons that fetch hint data from the API
 * and display it in a progressive bottom-sheet modal.
 */
export function RescuePanel({ sessionId, turnNumber, disabled = false }: RescuePanelProps) {
  const [hintData, setHintData] = useState<HintData | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [showModal, setShowModal] = useState(false)

  const fetchHint = async () => {
    if (isLoading || disabled) return

    setIsLoading(true)
    try {
      const data = await api.post<HintData>('/api/v1/chat/hint', {
        session_id: sessionId,
        turn_number: turnNumber,
      })
      setHintData(data)
      setShowModal(true)
    } catch {
      // Silently fail — hint is a non-critical feature
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <>
      <div className="flex gap-2 px-4 py-2">
        <button
          onClick={fetchHint}
          disabled={disabled || isLoading}
          className="flex items-center gap-1.5 bg-neutral-100 rounded-xl px-4 py-2 text-sm text-neutral-700 active:bg-neutral-200 transition-colors disabled:opacity-40"
        >
          <Lightbulb size={15} />
          ヒント
        </button>

        <button
          onClick={fetchHint}
          disabled={disabled || isLoading}
          className="flex items-center gap-1.5 bg-neutral-100 rounded-xl px-4 py-2 text-sm text-neutral-700 active:bg-neutral-200 transition-colors disabled:opacity-40"
        >
          <BookOpen size={15} />
          回答例
        </button>
      </div>

      {showModal && hintData && (
        <HintModal data={hintData} onClose={() => setShowModal(false)} />
      )}
    </>
  )
}
