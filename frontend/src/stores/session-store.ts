import { create } from 'zustand'

export type SessionPhase =
  | 'idle'
  | 'selecting_theme'
  | 'ai_speaking'
  | 'waiting_user'
  | 'recording'
  | 'transcribing'
  | 'ai_thinking'
  | 'session_complete'

interface SessionState {
  phase: SessionPhase
  sessionId: string | null
  themeId: string | null
  themeTitle: string | null
  turnNumber: number
  maxTurns: number
  setPhase: (phase: SessionPhase) => void
  startSession: (sessionId: string, themeId: string, themeTitle: string, maxTurns?: number) => void
  incrementTurn: () => void
  reset: () => void
}

const DEFAULT_MAX_TURNS = 6

export const useSessionStore = create<SessionState>((set) => ({
  phase: 'idle',
  sessionId: null,
  themeId: null,
  themeTitle: null,
  turnNumber: 0,
  maxTurns: DEFAULT_MAX_TURNS,

  setPhase: (phase) => set({ phase }),

  startSession: (sessionId, themeId, themeTitle, maxTurns = DEFAULT_MAX_TURNS) =>
    set({
      sessionId,
      themeId,
      themeTitle,
      phase: 'ai_speaking',
      turnNumber: 0,
      maxTurns,
    }),

  incrementTurn: () => set((state) => ({ turnNumber: state.turnNumber + 1 })),

  reset: () =>
    set({
      phase: 'idle',
      sessionId: null,
      themeId: null,
      themeTitle: null,
      turnNumber: 0,
      maxTurns: DEFAULT_MAX_TURNS,
    }),
}))
