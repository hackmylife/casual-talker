import { useAuthStore } from './auth-store'

// ---------------------------------------------------------------------------
// Mock setup
// ---------------------------------------------------------------------------

const mockFetch = vi.fn()
global.fetch = mockFetch

const mockStorage = new Map<string, string>()
vi.stubGlobal('localStorage', {
  getItem: (key: string) => mockStorage.get(key) ?? null,
  setItem: (key: string, value: string) => mockStorage.set(key, value),
  removeItem: (key: string) => mockStorage.delete(key),
})

// Suppress redirect side-effects from api-client's 401 handler.
Object.defineProperty(window, 'location', {
  value: { href: '' },
  writable: true,
})

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

const MOCK_USER = { id: 'u1', email: 'a@b.com', level: 2 }
const MOCK_TOKENS = { access_token: 'acc', refresh_token: 'ref' }

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('useAuthStore', () => {
  beforeEach(() => {
    mockFetch.mockReset()
    mockStorage.clear()
    // Reset Zustand store to its initial state before each test.
    useAuthStore.setState({
      isAuthenticated: false,
      isLoading: true,
      user: null,
    })
  })

  // -------------------------------------------------------------------------
  // Initial state
  // -------------------------------------------------------------------------

  it('has the correct initial state', () => {
    const { isAuthenticated, user, isLoading } = useAuthStore.getState()
    expect(isAuthenticated).toBe(false)
    expect(user).toBeNull()
    expect(isLoading).toBe(true)
  })

  // -------------------------------------------------------------------------
  // login
  // -------------------------------------------------------------------------

  describe('login', () => {
    it('sets isAuthenticated and user on success', async () => {
      mockFetch
        .mockResolvedValueOnce(jsonResponse(MOCK_TOKENS))  // POST /auth/login
        .mockResolvedValueOnce(jsonResponse(MOCK_USER))    // GET  /users/me

      await useAuthStore.getState().login('a@b.com', 'pw')

      const { isAuthenticated, user } = useAuthStore.getState()
      expect(isAuthenticated).toBe(true)
      expect(user).toEqual(MOCK_USER)
    })

    it('saves tokens to localStorage on success', async () => {
      mockFetch
        .mockResolvedValueOnce(jsonResponse(MOCK_TOKENS))
        .mockResolvedValueOnce(jsonResponse(MOCK_USER))

      await useAuthStore.getState().login('a@b.com', 'pw')

      expect(mockStorage.get('access_token')).toBe('acc')
      expect(mockStorage.get('refresh_token')).toBe('ref')
    })

    it('throws when the API returns an error', async () => {
      mockFetch.mockResolvedValueOnce(new Response(null, { status: 400 }))

      await expect(useAuthStore.getState().login('bad@b.com', 'wrong')).rejects.toThrow()
    })
  })

  // -------------------------------------------------------------------------
  // logout
  // -------------------------------------------------------------------------

  describe('logout', () => {
    it('clears user and isAuthenticated', async () => {
      // Start from a logged-in state
      useAuthStore.setState({ isAuthenticated: true, user: MOCK_USER, isLoading: false })
      mockStorage.set('access_token', 'acc')
      mockStorage.set('refresh_token', 'ref')

      // Logout POST may succeed or fail; either way the state should reset.
      mockFetch.mockResolvedValueOnce(jsonResponse({ ok: true }))

      await useAuthStore.getState().logout()

      const { isAuthenticated, user } = useAuthStore.getState()
      expect(isAuthenticated).toBe(false)
      expect(user).toBeNull()
    })

    it('removes tokens from localStorage', async () => {
      useAuthStore.setState({ isAuthenticated: true, user: MOCK_USER, isLoading: false })
      mockStorage.set('access_token', 'acc')
      mockStorage.set('refresh_token', 'ref')

      mockFetch.mockResolvedValueOnce(jsonResponse({ ok: true }))

      await useAuthStore.getState().logout()

      expect(mockStorage.get('access_token')).toBeUndefined()
      expect(mockStorage.get('refresh_token')).toBeUndefined()
    })

    it('still clears state even when the logout API call fails', async () => {
      useAuthStore.setState({ isAuthenticated: true, user: MOCK_USER, isLoading: false })
      mockFetch.mockResolvedValueOnce(new Response(null, { status: 500 }))

      // The logout implementation uses try/finally, so the state is cleared
      // even if the API call throws. The error propagates, so we swallow it here.
      await useAuthStore.getState().logout().catch(() => {})

      expect(useAuthStore.getState().isAuthenticated).toBe(false)
      expect(useAuthStore.getState().user).toBeNull()
    })
  })

  // -------------------------------------------------------------------------
  // checkAuth
  // -------------------------------------------------------------------------

  describe('checkAuth', () => {
    it('fetches /users/me and sets user when an access token is present', async () => {
      mockStorage.set('access_token', 'valid-token')
      mockFetch.mockResolvedValueOnce(jsonResponse(MOCK_USER))

      await useAuthStore.getState().checkAuth()

      const { isAuthenticated, user, isLoading } = useAuthStore.getState()
      expect(isAuthenticated).toBe(true)
      expect(user).toEqual(MOCK_USER)
      expect(isLoading).toBe(false)
    })

    it('sets isAuthenticated=false and isLoading=false when no token exists', async () => {
      // No token in storage

      await useAuthStore.getState().checkAuth()

      const { isAuthenticated, user, isLoading } = useAuthStore.getState()
      expect(isAuthenticated).toBe(false)
      expect(user).toBeNull()
      expect(isLoading).toBe(false)
      // fetch should not have been called
      expect(mockFetch).not.toHaveBeenCalled()
    })

    it('sets isAuthenticated=false when the /users/me call fails', async () => {
      mockStorage.set('access_token', 'expired-token')
      // Trigger a non-401 error so api-client throws directly
      mockFetch.mockResolvedValueOnce(new Response(null, { status: 500 }))

      await useAuthStore.getState().checkAuth()

      const { isAuthenticated, isLoading } = useAuthStore.getState()
      expect(isAuthenticated).toBe(false)
      expect(isLoading).toBe(false)
    })
  })
})
