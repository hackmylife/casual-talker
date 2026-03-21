interface LoadingSpinnerProps {
  size?: 'sm' | 'md' | 'lg'
  color?: string
}

const sizeClasses = {
  sm: 'w-4 h-4 border-2',
  md: 'w-8 h-8 border-2',
  lg: 'w-12 h-12 border-3',
}

export function LoadingSpinner({ size = 'md', color = 'border-primary-600' }: LoadingSpinnerProps) {
  return (
    <div
      className={`${sizeClasses[size]} ${color} border-t-transparent rounded-full animate-spin`}
      role="status"
      aria-label="読み込み中"
    />
  )
}

export function LoadingScreen() {
  return (
    <div className="flex items-center justify-center min-h-svh bg-neutral-50">
      <LoadingSpinner size="lg" />
    </div>
  )
}
