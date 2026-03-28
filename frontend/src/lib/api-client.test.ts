import { api, ApiError } from './api-client'

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

// Capture location.href assignments so we can assert on redirects without
// actually navigating.
const locationAssignments: string[] = []
Object.defineProperty(window, 'location', {
  value: {
    get href() {
      return locationAssignments[locationAssignments.length - 1] ?? ''
    },
    set href(url: string) {
      locationAssignments.push(url)
    },
  },
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

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('api-client', () => {
  beforeEach(() => {
    mockFetch.mockReset()
    mockStorage.clear()
    locationAssignments.length = 0
  })

  // -------------------------------------------------------------------------
  // api.get
  // -------------------------------------------------------------------------

  describe('api.get', () => {
    it('returns parsed JSON on a successful response', async () => {
      mockFetch.mockResolvedValueOnce(jsonResponse({ id: '1', name: 'Alice' }))

      const result = await api.get<{ id: string; name: string }>('/api/v1/users/me')

      expect(result).toEqual({ id: '1', name: 'Alice' })
    })

    it('sends Content-Type: application/json by default', async () => {
      mockFetch.mockResolvedValueOnce(jsonResponse({}))

      await api.get('/api/v1/users/me')

      const [, init] = mockFetch.mock.calls[0] as [string, RequestInit]
      const headers = new Headers(init.headers)
      expect(headers.get('Content-Type')).toBe('application/json')
    })

    it('includes Authorization header when access_token is present', async () => {
      mockStorage.set('access_token', 'my-token')
      mockFetch.mockResolvedValueOnce(jsonResponse({}))

      await api.get('/api/v1/users/me')

      const [, init] = mockFetch.mock.calls[0] as [string, RequestInit]
      const headers = new Headers(init.headers)
      expect(headers.get('Authorization')).toBe('Bearer my-token')
    })
  })

  // -------------------------------------------------------------------------
  // api.post
  // -------------------------------------------------------------------------

  describe('api.post', () => {
    it('sends Content-Type: application/json', async () => {
      mockFetch.mockResolvedValueOnce(jsonResponse({ ok: true }))

      await api.post('/api/v1/auth/login', { email: 'a@b.com', password: 'pw' })

      const [, init] = mockFetch.mock.calls[0] as [string, RequestInit]
      const headers = new Headers(init.headers)
      expect(headers.get('Content-Type')).toBe('application/json')
    })

    it('serialises the body as JSON', async () => {
      mockFetch.mockResolvedValueOnce(jsonResponse({ ok: true }))

      await api.post('/api/v1/auth/login', { email: 'a@b.com' })

      const [, init] = mockFetch.mock.calls[0] as [string, RequestInit]
      expect(JSON.parse(init.body as string)).toEqual({ email: 'a@b.com' })
    })
  })

  // -------------------------------------------------------------------------
  // api.postForm
  // -------------------------------------------------------------------------

  describe('api.postForm', () => {
    it('does NOT set Content-Type so the browser can add multipart/form-data boundary', async () => {
      mockFetch.mockResolvedValueOnce(jsonResponse({ ok: true }))

      const form = new FormData()
      form.append('file', new Blob(['hello']), 'hello.txt')
      await api.postForm('/api/v1/upload', form)

      const [, init] = mockFetch.mock.calls[0] as [string, RequestInit]
      const headers = new Headers(init.headers)
      // The browser sets Content-Type automatically for FormData; the client
      // must NOT pre-set it so the boundary token is included.
      expect(headers.get('Content-Type')).toBeNull()
    })
  })

  // -------------------------------------------------------------------------
  // 401 handling – token refresh
  // -------------------------------------------------------------------------

  describe('401 response handling', () => {
    it('calls tryRefreshToken and retries the original request on 401', async () => {
      mockStorage.set('refresh_token', 'refresh-tok')

      // First call → 401 on the original request
      // Second call (inside tryRefreshToken) → refresh succeeds
      // Third call → retry of the original request succeeds
      mockFetch
        .mockResolvedValueOnce(new Response(null, { status: 401 }))
        .mockResolvedValueOnce(
          jsonResponse({ access_token: 'new-access', refresh_token: 'new-refresh' }),
        )
        .mockResolvedValueOnce(jsonResponse({ id: '42' }))

      const result = await api.get<{ id: string }>('/api/v1/users/me')

      expect(mockFetch).toHaveBeenCalledTimes(3)
      expect(result).toEqual({ id: '42' })
      // New tokens must be persisted
      expect(mockStorage.get('access_token')).toBe('new-access')
    })

    it('redirects to /login when the token refresh itself fails', async () => {
      mockStorage.set('refresh_token', 'stale-refresh')

      mockFetch
        .mockResolvedValueOnce(new Response(null, { status: 401 }))
        // Refresh request returns a non-OK response
        .mockResolvedValueOnce(new Response(null, { status: 400 }))

      await expect(api.get('/api/v1/users/me')).rejects.toBeInstanceOf(ApiError)
      expect(locationAssignments).toContain('/login')
    })

    it('redirects to /login when there is no refresh token', async () => {
      // No tokens at all in storage
      mockFetch.mockResolvedValueOnce(new Response(null, { status: 401 }))

      await expect(api.get('/api/v1/users/me')).rejects.toBeInstanceOf(ApiError)
      expect(locationAssignments).toContain('/login')
    })

    it('does NOT retry infinitely: a 401 on the retry throws without another refresh attempt', async () => {
      mockStorage.set('refresh_token', 'refresh-tok')

      mockFetch
        .mockResolvedValueOnce(new Response(null, { status: 401 }))
        // Refresh succeeds
        .mockResolvedValueOnce(
          jsonResponse({ access_token: 'new-access', refresh_token: 'new-refresh' }),
        )
        // But the retried original request also returns 401
        .mockResolvedValueOnce(new Response(null, { status: 401 }))

      // The second 401 (allowRetry=false path) should redirect and throw.
      await expect(api.get('/api/v1/users/me')).rejects.toBeInstanceOf(ApiError)
      // Only one refresh attempt; no third refresh
      expect(mockFetch).toHaveBeenCalledTimes(3)
    })
  })

  // -------------------------------------------------------------------------
  // Non-401 errors
  // -------------------------------------------------------------------------

  describe('non-401 error responses', () => {
    it('throws ApiError with the response status for 4xx/5xx errors', async () => {
      mockFetch.mockResolvedValueOnce(
        new Response(JSON.stringify({ message: 'Not found' }), { status: 404 }),
      )

      await expect(api.get('/api/v1/missing')).rejects.toMatchObject({
        status: 404,
      })
    })

    it('throws an ApiError instance (not a plain Error)', async () => {
      mockFetch.mockResolvedValueOnce(new Response(null, { status: 500 }))

      const err = await api.get('/api/v1/boom').catch((e) => e)
      expect(err).toBeInstanceOf(ApiError)
      expect(err.status).toBe(500)
    })
  })
})
