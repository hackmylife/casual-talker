import { useSessionStore } from './session-store'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const DEFAULT_MAX_TURNS = 6

function resetStore() {
  useSessionStore.setState({
    phase: 'idle',
    sessionId: null,
    themeId: null,
    themeTitle: null,
    turnNumber: 0,
    maxTurns: DEFAULT_MAX_TURNS,
  })
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('useSessionStore', () => {
  beforeEach(() => {
    resetStore()
  })

  // -------------------------------------------------------------------------
  // Initial state
  // -------------------------------------------------------------------------

  it('has the correct initial state', () => {
    const state = useSessionStore.getState()
    expect(state.phase).toBe('idle')
    expect(state.turnNumber).toBe(0)
    expect(state.maxTurns).toBe(DEFAULT_MAX_TURNS)
    expect(state.sessionId).toBeNull()
    expect(state.themeId).toBeNull()
    expect(state.themeTitle).toBeNull()
  })

  // -------------------------------------------------------------------------
  // startSession
  // -------------------------------------------------------------------------

  describe('startSession', () => {
    it('sets phase to ai_speaking and records session/theme IDs', () => {
      useSessionStore.getState().startSession('sess-1', 'theme-a', 'Sports')

      const { phase, sessionId, themeId, themeTitle } = useSessionStore.getState()
      expect(phase).toBe('ai_speaking')
      expect(sessionId).toBe('sess-1')
      expect(themeId).toBe('theme-a')
      expect(themeTitle).toBe('Sports')
    })

    it('resets turnNumber to 0 when a new session starts', () => {
      // Simulate an in-progress session
      useSessionStore.setState({ turnNumber: 3 })

      useSessionStore.getState().startSession('sess-2', 'theme-b', 'Music')

      expect(useSessionStore.getState().turnNumber).toBe(0)
    })

    it('accepts a custom maxTurns value', () => {
      useSessionStore.getState().startSession('sess-3', 'theme-c', 'Food', 10)

      expect(useSessionStore.getState().maxTurns).toBe(10)
    })

    it('uses DEFAULT_MAX_TURNS when maxTurns is not provided', () => {
      useSessionStore.getState().startSession('sess-4', 'theme-d', 'Travel')

      expect(useSessionStore.getState().maxTurns).toBe(DEFAULT_MAX_TURNS)
    })
  })

  // -------------------------------------------------------------------------
  // incrementTurn
  // -------------------------------------------------------------------------

  describe('incrementTurn', () => {
    it('increments turnNumber by 1', () => {
      useSessionStore.getState().incrementTurn()

      expect(useSessionStore.getState().turnNumber).toBe(1)
    })

    it('increments turnNumber correctly after multiple calls', () => {
      useSessionStore.getState().incrementTurn()
      useSessionStore.getState().incrementTurn()
      useSessionStore.getState().incrementTurn()

      expect(useSessionStore.getState().turnNumber).toBe(3)
    })
  })

  // -------------------------------------------------------------------------
  // reset
  // -------------------------------------------------------------------------

  describe('reset', () => {
    it('restores all fields to their initial values', () => {
      useSessionStore.getState().startSession('sess-5', 'theme-e', 'Anime', 8)
      useSessionStore.getState().incrementTurn()
      useSessionStore.getState().setPhase('recording')

      useSessionStore.getState().reset()

      const state = useSessionStore.getState()
      expect(state.phase).toBe('idle')
      expect(state.sessionId).toBeNull()
      expect(state.themeId).toBeNull()
      expect(state.themeTitle).toBeNull()
      expect(state.turnNumber).toBe(0)
      expect(state.maxTurns).toBe(DEFAULT_MAX_TURNS)
    })
  })

  // -------------------------------------------------------------------------
  // setPhase
  // -------------------------------------------------------------------------

  describe('setPhase', () => {
    it('updates the phase field', () => {
      useSessionStore.getState().setPhase('recording')
      expect(useSessionStore.getState().phase).toBe('recording')
    })

    it('can transition through multiple phases', () => {
      const phases = [
        'selecting_theme',
        'ai_speaking',
        'waiting_user',
        'recording',
        'transcribing',
        'ai_thinking',
        'session_complete',
        'idle',
      ] as const

      for (const phase of phases) {
        useSessionStore.getState().setPhase(phase)
        expect(useSessionStore.getState().phase).toBe(phase)
      }
    })
  })
})
