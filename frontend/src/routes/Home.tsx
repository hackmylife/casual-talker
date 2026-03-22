import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router'
import { api } from '@/lib/api-client'
import { LoadingSpinner } from '@/components/common/LoadingSpinner'
import { OnboardingFlow } from '@/components/onboarding/OnboardingFlow'

interface Course {
  id: string
  title: string
  description: string
  target_language: string
}

interface Theme {
  id: string
  course_id: string
  title: string
  description: string
  target_phrases?: string[]
}

interface CreateSessionResponse {
  id: string
  max_turns: number
}

interface LanguageStats {
  sessions: number
  last_practiced: string | null
}

interface UserStats {
  total_sessions: number
  total_practice_minutes: number
  total_user_turns: number
  current_streak: number
  pronunciation_fixes: number
  languages: Record<string, LanguageStats>
}

// Supported target languages and their display metadata
const LANGUAGES: { code: string; label: string; flag: string }[] = [
  { code: 'en', label: '英語', flag: '🇬🇧' },
  { code: 'it', label: 'イタリア語', flag: '🇮🇹' },
  { code: 'ko', label: '韓国語', flag: '🇰🇷' },
  { code: 'pt', label: 'ポルトガル語', flag: '🇧🇷' },
]

const STORAGE_KEY_LANG = 'selected_language'

function getSavedLanguage(): string {
  const saved = localStorage.getItem(STORAGE_KEY_LANG)
  if (saved && LANGUAGES.some((l) => l.code === saved)) return saved
  return 'en'
}

// Format total minutes into a compact display string.
// Under 60 minutes: "◯分", 60 and over: "◯時間◯分"
function formatMinutes(minutes: number): string {
  if (minutes < 60) return `${minutes}分`
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  return m === 0 ? `${h}時間` : `${h}時間${m}分`
}

// Format an ISO timestamp into a relative date label ("今日", "昨日", "◯日前").
function formatLastPracticed(isoStr: string): string {
  const jst = new Date(isoStr)
  const now = new Date()

  // Compute calendar-day difference in JST
  const toJSTMidnight = (d: Date) => {
    const jstOffset = 9 * 60 // JST is UTC+9
    const local = new Date(d.getTime() + jstOffset * 60 * 1000)
    return new Date(local.toISOString().slice(0, 10) + 'T00:00:00Z')
  }
  const dayDiff = Math.round(
    (toJSTMidnight(now).getTime() - toJSTMidnight(jst).getTime()) /
      (24 * 60 * 60 * 1000),
  )

  if (dayDiff === 0) return '今日'
  if (dayDiff === 1) return '昨日'
  return `${dayDiff}日前`
}

// StatsCard renders the compact practice-statistics banner.
// It is hidden entirely when the user has zero completed sessions.
function StatsCard({ stats }: { stats: UserStats }) {
  if (stats.total_sessions === 0) return null

  return (
    <div className="bg-white rounded-2xl p-4 shadow-sm mb-6">
      <div className="flex flex-wrap gap-4">
        {stats.current_streak > 0 && (
          <StatItem
            emoji="🔥"
            value={`${stats.current_streak}日連続`}
          />
        )}
        <StatItem emoji="💬" value={`${stats.total_sessions}回`} />
        <StatItem emoji="⏱" value={formatMinutes(stats.total_practice_minutes)} />
        <StatItem emoji="🗣" value={`${stats.total_user_turns}ターン`} />
        {stats.pronunciation_fixes > 0 && (
          <StatItem emoji="✏️" value={`${stats.pronunciation_fixes}回修正`} />
        )}
      </div>
    </div>
  )
}

function StatItem({ emoji, value }: { emoji: string; value: string }) {
  return (
    <div className="flex items-baseline gap-1">
      <span className="text-base leading-none">{emoji}</span>
      <span className="text-lg font-bold text-neutral-900 leading-none">{value}</span>
    </div>
  )
}

// LangStats renders the per-language mini statistics below the language tabs.
// Only the currently selected language is shown.
function LangStats({
  stats,
  selectedLang,
}: {
  stats: UserStats | null
  selectedLang: string
}) {
  if (!stats || stats.total_sessions === 0) return null
  const ls = stats.languages[selectedLang]
  if (!ls || ls.sessions === 0) return null

  return (
    <p className="text-xs text-neutral-500 mb-4 -mt-4">
      {ls.sessions}セッション
      {ls.last_practiced ? ` • 最終: ${formatLastPracticed(ls.last_practiced)}` : ''}
    </p>
  )
}

export default function Home() {
  const navigate = useNavigate()

  // Show onboarding only on the first visit (controlled by localStorage flag)
  const [showOnboarding, setShowOnboarding] = useState<boolean>(
    () => localStorage.getItem('onboarding_completed') !== 'true',
  )

  const [selectedLang, setSelectedLang] = useState<string>(getSavedLanguage)
  const [courses, setCourses] = useState<Course[]>([])
  const [themesByCourse, setThemesByCourse] = useState<Record<string, Theme[]>>({})
  const [isLoadingCourses, setIsLoadingCourses] = useState(true)
  const [startingThemeId, setStartingThemeId] = useState<string | null>(null)
  const [userStats, setUserStats] = useState<UserStats | null>(null)

  // Persist language selection
  const handleLangSelect = (code: string) => {
    setSelectedLang(code)
    localStorage.setItem(STORAGE_KEY_LANG, code)
  }

  // Fetch all courses, then fetch themes for each course in parallel
  useEffect(() => {
    let cancelled = false

    const load = async () => {
      try {
        const fetchedCourses = await api.get<Course[]>('/api/v1/courses')
        if (cancelled) return
        setCourses(fetchedCourses)

        const entries = await Promise.all(
          fetchedCourses.map(async (course) => {
            const themes = await api.get<Theme[]>(`/api/v1/courses/${course.id}/themes`)
            return [course.id, themes] as [string, Theme[]]
          }),
        )
        if (cancelled) return
        setThemesByCourse(Object.fromEntries(entries))
      } finally {
        if (!cancelled) setIsLoadingCourses(false)
      }
    }

    load()
    return () => {
      cancelled = true
    }
  }, [])

  // Fetch user stats on mount — no caching, always latest data
  useEffect(() => {
    let cancelled = false

    const loadStats = async () => {
      try {
        const stats = await api.get<UserStats>('/api/v1/users/me/stats')
        if (!cancelled) setUserStats(stats)
      } catch {
        // Stats are non-critical; silently ignore errors
      }
    }

    loadStats()
    return () => {
      cancelled = true
    }
  }, [])

  // Courses that belong to the currently selected language
  const filteredCourses = courses.filter((c) => c.target_language === selectedLang)

  const handleThemeSelect = async (theme: Theme) => {
    if (startingThemeId) return // Prevent double-tap
    setStartingThemeId(theme.id)

    // Find the parent course to pass target_language to the session screen
    const parentCourse = courses.find((c) => c.id === theme.course_id)

    try {
      const session = await api.post<CreateSessionResponse>('/api/v1/sessions', {
        theme_id: theme.id,
        difficulty: 1,
      })
      navigate(`/session/${session.id}`, {
        state: {
          themeId: theme.id,
          themeTitle: theme.title,
          maxTurns: session.max_turns,
          targetLanguage: parentCourse?.target_language ?? 'en',
        },
      })
    } catch {
      setStartingThemeId(null)
    }
  }

  // Render the onboarding flow until the user completes it
  if (showOnboarding) {
    return <OnboardingFlow onComplete={() => setShowOnboarding(false)} />
  }

  return (
    <div className="min-h-full bg-neutral-50 px-4 pt-6 pb-8">
      {/* Greeting */}
      <div className="mb-6">
        <h2 className="text-2xl font-semibold text-neutral-900 leading-snug">
          今日も10分だけ話そう
        </h2>
        <p className="mt-1 text-sm text-neutral-600">話したいテーマを選んでスタート</p>
      </div>

      {/* Practice statistics card — hidden for first-time users */}
      {userStats !== null && <StatsCard stats={userStats} />}

      {/* Language selector tabs */}
      <div className="flex gap-2 mb-6 overflow-x-auto pb-1 -mx-1 px-1">
        {LANGUAGES.map((lang) => (
          <button
            key={lang.code}
            onClick={() => handleLangSelect(lang.code)}
            className={[
              'flex items-center gap-1.5 px-4 py-2 rounded-full text-sm font-medium whitespace-nowrap transition-colors flex-shrink-0',
              selectedLang === lang.code
                ? 'bg-primary-600 text-white'
                : 'bg-neutral-100 text-neutral-700 hover:bg-neutral-200',
            ].join(' ')}
          >
            <span>{lang.flag}</span>
            <span>{lang.label}</span>
          </button>
        ))}
      </div>

      {/* Per-language mini stats */}
      <LangStats stats={userStats} selectedLang={selectedLang} />

      {isLoadingCourses ? (
        <div className="flex justify-center pt-16">
          <LoadingSpinner size="lg" />
        </div>
      ) : (
        <div className="flex flex-col gap-6">
          {filteredCourses.map((course) => {
            const themes = themesByCourse[course.id] ?? []
            return (
              <section key={course.id}>
                <h3 className="text-xs font-medium text-neutral-400 uppercase tracking-wider mb-3">
                  {course.title}
                </h3>
                <div className="flex flex-col gap-3">
                  {themes.map((theme) => {
                    const isStarting = startingThemeId === theme.id
                    return (
                      <button
                        key={theme.id}
                        onClick={() => handleThemeSelect(theme)}
                        disabled={!!startingThemeId}
                        className="w-full text-left bg-neutral-100 rounded-2xl p-4 active:bg-neutral-200 transition-colors disabled:opacity-60"
                      >
                        <div className="flex items-start justify-between gap-2">
                          <div>
                            <p className="text-lg font-semibold text-neutral-900 leading-snug">
                              {theme.title}
                            </p>
                            <p className="mt-1 text-sm text-neutral-600 leading-relaxed">
                              {theme.description}
                            </p>
                          </div>
                          {isStarting && (
                            <div className="flex-shrink-0 mt-1">
                              <LoadingSpinner size="sm" />
                            </div>
                          )}
                        </div>
                      </button>
                    )
                  })}
                </div>
              </section>
            )
          })}
        </div>
      )}
    </div>
  )
}
