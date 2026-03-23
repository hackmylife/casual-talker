// Level display utilities shared across Home and Feedback views.

export const LEVEL_LABELS: Record<number, string> = {
  1: '単語・超短文',
  2: '基本文',
  3: '理由追加',
  4: '複数文',
  5: '自発展開',
}

// Returns the Tailwind badge classes for a given level (1-5).
export function levelBadgeClass(level: number): string {
  switch (level) {
    case 2: return 'bg-blue-100 text-blue-700'
    case 3: return 'bg-green-100 text-green-700'
    case 4: return 'bg-purple-100 text-purple-700'
    case 5: return 'bg-amber-100 text-amber-700'
    default: return 'bg-neutral-100 text-neutral-600'
  }
}
