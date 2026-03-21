import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router'
import { api } from '@/lib/api-client'
import { LoadingSpinner } from '@/components/common/LoadingSpinner'

interface Session {
  id: string
  user_id: string
  theme_id: string
  difficulty: number
  status: 'active' | 'completed' | 'abandoned'
  started_at: string
  ended_at: string | null
  turn_count: number
}

interface Theme {
  id: string
  title: string
  description: string
}

// Format session duration in minutes, returning null if either timestamp is missing.
function formatDurationMinutes(startedAt: string, endedAt: string | null): string | null {
  if (!endedAt) return null
  const diffMs = new Date(endedAt).getTime() - new Date(startedAt).getTime()
  if (diffMs <= 0) return null
  const minutes = Math.round(diffMs / 60_000)
  return `${minutes}分`
}

// Map session status to a display label and Tailwind text class.
function statusDisplay(status: Session['status']): { label: string; className: string } {
  switch (status) {
    case 'completed':
      return { label: '完了', className: 'text-success' }
    case 'abandoned':
      return { label: '中断', className: 'text-neutral-600' }
    case 'active':
    default:
      return { label: '進行中', className: 'text-primary-600' }
  }
}

// Group sessions by their local date string (ja-JP format).
function groupByDate(sessions: Session[]): [string, Session[]][] {
  const map = new Map<string, Session[]>()

  for (const session of sessions) {
    const dateKey = new Date(session.started_at).toLocaleDateString('ja-JP', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    })
    const group = map.get(dateKey) ?? []
    group.push(session)
    map.set(dateKey, group)
  }

  return Array.from(map.entries())
}

export default function History() {
  const navigate = useNavigate()

  const [sessions, setSessions] = useState<Session[]>([])
  const [themeMap, setThemeMap] = useState<Map<string, Theme>>(new Map())
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false

    const load = async () => {
      try {
        // Fetch all sessions (up to 50 most recent)
        const fetchedSessions = await api.get<Session[]>(
          '/api/v1/sessions?limit=50&offset=0',
        )
        if (cancelled) return

        setSessions(fetchedSessions)

        // Collect unique theme IDs and fetch each theme in parallel
        const uniqueThemeIds = [...new Set(fetchedSessions.map((s) => s.theme_id))]
        const themeEntries = await Promise.all(
          uniqueThemeIds.map(async (themeId): Promise<[string, Theme] | null> => {
            try {
              const theme = await api.get<Theme>(`/api/v1/themes/${themeId}`)
              return [themeId, theme]
            } catch {
              return null
            }
          }),
        )

        if (cancelled) return

        const map = new Map<string, Theme>()
        for (const entry of themeEntries) {
          if (entry) map.set(entry[0], entry[1])
        }
        setThemeMap(map)
      } catch {
        if (!cancelled) {
          setError('履歴の取得に失敗しました')
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
  }, [])

  if (isLoading) {
    return (
      <div className="flex justify-center pt-16">
        <LoadingSpinner size="lg" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] px-4 gap-2">
        <p className="text-neutral-600">{error}</p>
      </div>
    )
  }

  const grouped = groupByDate(sessions)

  return (
    <div className="min-h-full bg-neutral-50 px-4 pt-6 pb-8">
      {/* Page title */}
      <h1 className="text-2xl font-semibold text-neutral-900 mb-6">学習きろく</h1>

      {sessions.length === 0 ? (
        /* Empty state */
        <div className="flex flex-col items-center justify-center pt-16 gap-3 text-center">
          <p className="text-5xl">📭</p>
          <p className="text-neutral-800 font-medium">まだセッションがありません</p>
          <p className="text-sm text-neutral-600">ホームから始めましょう</p>
          <button
            onClick={() => navigate('/')}
            className="mt-4 px-6 py-2 bg-primary-600 text-white rounded-2xl text-sm font-medium active:bg-primary-700 transition-colors"
          >
            ホームへ
          </button>
        </div>
      ) : (
        <div className="space-y-6">
          {grouped.map(([dateLabel, dateSessions]) => (
            <section key={dateLabel}>
              {/* Date group header */}
              <p className="text-sm font-medium text-neutral-600 mb-2">{dateLabel}</p>

              <div className="flex flex-col gap-3">
                {dateSessions.map((session) => {
                  const theme = themeMap.get(session.theme_id)
                  const duration = formatDurationMinutes(session.started_at, session.ended_at)
                  const { label: statusLabel, className: statusClass } = statusDisplay(
                    session.status,
                  )

                  return (
                    <button
                      key={session.id}
                      onClick={() => navigate(`/feedback/${session.id}`)}
                      className="w-full text-left bg-white rounded-2xl p-4 shadow-sm active:bg-neutral-100 transition-colors"
                    >
                      <div className="flex items-center justify-between gap-2">
                        {/* Theme title and turn count */}
                        <div className="min-w-0">
                          <p className="font-semibold text-neutral-900 truncate">
                            {theme?.title ?? 'テーマ不明'}
                          </p>
                          <p className="mt-0.5 text-sm text-neutral-600">
                            {session.turn_count}ターン完了
                          </p>
                        </div>

                        {/* Duration and status */}
                        <div className="flex-shrink-0 text-right">
                          {duration && (
                            <p className="text-sm text-neutral-600">{duration}</p>
                          )}
                          <p className={`text-sm font-medium ${statusClass}`}>{statusLabel}</p>
                        </div>
                      </div>
                    </button>
                  )
                })}
              </div>
            </section>
          ))}
        </div>
      )}
    </div>
  )
}
