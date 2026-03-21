import { ArrowDown } from 'lucide-react'

export interface ExpressionCardProps {
  original: string
  natural: string
}

// Card displaying a before/after comparison of user expression vs. natural expression.
export function ExpressionCard({ original, natural }: ExpressionCardProps) {
  return (
    <div className="bg-white rounded-2xl p-4">
      {/* Original (user) expression */}
      <p className="text-sm text-neutral-600">
        <span className="font-medium">あなた:</span> &ldquo;{original}&rdquo;
      </p>

      {/* Arrow separator */}
      <div className="flex justify-center my-2 text-neutral-300">
        <ArrowDown size={16} />
      </div>

      {/* Natural expression suggestion */}
      <p className="text-sm">
        <span className="font-medium text-neutral-600">自然:</span>{' '}
        <span className="text-primary-600 font-medium">&ldquo;{natural}&rdquo;</span>
      </p>
    </div>
  )
}
