import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router'
import { api } from '@/lib/api-client'
import { LoadingSpinner } from '@/components/common/LoadingSpinner'
import { OnboardingFlow } from '@/components/onboarding/OnboardingFlow'

interface Course {
  id: string
  title: string
  description: string
}

interface Theme {
  id: string
  title: string
  description: string
  target_phrases?: string[]
}

interface CreateSessionResponse {
  id: string
  max_turns: number
}

export default function Home() {
  const navigate = useNavigate()

  // Show onboarding only on the first visit (controlled by localStorage flag)
  const [showOnboarding, setShowOnboarding] = useState<boolean>(
    () => localStorage.getItem('onboarding_completed') !== 'true',
  )

  const [courses, setCourses] = useState<Course[]>([])
  const [themesByCourse, setThemesByCourse] = useState<Record<string, Theme[]>>({})
  const [isLoadingCourses, setIsLoadingCourses] = useState(true)
  const [startingThemeId, setStartingThemeId] = useState<string | null>(null)

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

  const handleThemeSelect = async (theme: Theme) => {
    if (startingThemeId) return // Prevent double-tap
    setStartingThemeId(theme.id)

    try {
      const session = await api.post<CreateSessionResponse>('/api/v1/sessions', {
        theme_id: theme.id,
        difficulty: 1,
      })
      navigate(`/session/${session.id}`, {
        state: { themeId: theme.id, themeTitle: theme.title, maxTurns: session.max_turns },
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

      {isLoadingCourses ? (
        <div className="flex justify-center pt-16">
          <LoadingSpinner size="lg" />
        </div>
      ) : (
        <div className="flex flex-col gap-6">
          {courses.map((course) => {
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
