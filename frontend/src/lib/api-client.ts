// Base path for API requests. In production behind a sub-path reverse proxy
// (e.g. /talk/), Vite's import.meta.env.BASE_URL provides the prefix.
// In dev mode with the Vite proxy, BASE_URL is "/" so paths stay relative.
const API_BASE = (import.meta.env.BASE_URL ?? '/').replace(/\/$/, '')

class ApiError extends Error {
  status: number
  data: unknown

  constructor(status: number, data: unknown) {
    super(`API Error: ${status}`)
    this.status = status
    this.data = data
  }
}

async function apiFetch<T>(
  path: string,
  options: RequestInit = {},
  allowRetry = true,
): Promise<T> {
  const token = localStorage.getItem('access_token')

  // Build headers: start with defaults, merge caller headers on top.
  // For FormData requests the caller passes an empty headers object so that
  // Content-Type is NOT set here and the browser can auto-set multipart/form-data.
  const defaultHeaders: Record<string, string> = options.headers !== undefined
    ? {} // caller controls Content-Type (e.g. FormData)
    : { 'Content-Type': 'application/json' }

  if (token) {
    defaultHeaders['Authorization'] = `Bearer ${token}`
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      ...defaultHeaders,
      ...options.headers,
    },
  })

  if (res.status === 401 && allowRetry) {
    // Attempt a silent token refresh before giving up.
    // Pass allowRetry=false to prevent infinite loops.
    const refreshed = await tryRefreshToken()
    if (refreshed) {
      return apiFetch<T>(path, options, false)
    }
    // No valid session — redirect to login.
    window.location.href = `${API_BASE}/login`
    throw new ApiError(401, 'Unauthorized')
  }

  if (!res.ok) {
    throw new ApiError(res.status, await res.json().catch(() => null))
  }

  return res.json() as Promise<T>
}

// apiFetchBlob is like apiFetch but returns a Blob instead of parsed JSON.
// Used for binary endpoints such as TTS audio.
async function apiFetchBlob(path: string, options: RequestInit = {}): Promise<Blob> {
  const token = localStorage.getItem('access_token')

  const defaultHeaders: Record<string, string> = options.headers !== undefined
    ? {}
    : { 'Content-Type': 'application/json' }

  if (token) {
    defaultHeaders['Authorization'] = `Bearer ${token}`
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      ...defaultHeaders,
      ...options.headers,
    },
  })

  if (res.status === 401) {
    const refreshed = await tryRefreshToken()
    if (refreshed) {
      return apiFetchBlob(path, options)
    }
    window.location.href = `${API_BASE}/login`
    throw new ApiError(401, 'Unauthorized')
  }

  if (!res.ok) {
    throw new ApiError(res.status, await res.json().catch(() => null))
  }

  return res.blob()
}

async function tryRefreshToken(): Promise<boolean> {
  const refreshToken = localStorage.getItem('refresh_token')
  if (!refreshToken) return false

  try {
    const res = await fetch(`${API_BASE}/api/v1/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
    if (!res.ok) return false
    const data = await res.json() as { access_token: string; refresh_token?: string }
    localStorage.setItem('access_token', data.access_token)
    if (data.refresh_token) {
      localStorage.setItem('refresh_token', data.refresh_token)
    }
    return true
  } catch {
    return false
  }
}

export const api = {
  get: <T>(path: string) => apiFetch<T>(path),

  post: <T>(path: string, body?: unknown) =>
    apiFetch<T>(path, {
      method: 'POST',
      body: body !== undefined ? JSON.stringify(body) : undefined,
    }),

  put: <T>(path: string, body?: unknown) =>
    apiFetch<T>(path, {
      method: 'PUT',
      body: body !== undefined ? JSON.stringify(body) : undefined,
    }),

  // Pass an empty headers object so that apiFetch skips the default
  // Content-Type header and lets the browser set multipart/form-data automatically.
  postForm: <T>(path: string, formData: FormData) =>
    apiFetch<T>(path, {
      method: 'POST',
      headers: {},
      body: formData,
    }),

  // POST with JSON body, returning the response as a Blob (e.g. TTS audio).
  postBlob: (path: string, body?: unknown) =>
    apiFetchBlob(path, {
      method: 'POST',
      body: body !== undefined ? JSON.stringify(body) : undefined,
    }),
}

export { ApiError }
