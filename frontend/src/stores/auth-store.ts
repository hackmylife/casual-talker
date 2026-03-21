import { create } from 'zustand'
import { api } from '@/lib/api-client'

interface User {
  id: string
  email: string
  displayName?: string
  level: number
}

interface AuthState {
  isAuthenticated: boolean
  isLoading: boolean
  user: User | null
  login: (email: string, password: string) => Promise<void>
  register: (email: string, password: string, displayName: string) => Promise<void>
  logout: () => Promise<void>
  checkAuth: () => Promise<void>
}

interface AuthTokenResponse {
  access_token: string
  refresh_token: string
}

function saveTokens(tokens: AuthTokenResponse): void {
  localStorage.setItem('access_token', tokens.access_token)
  localStorage.setItem('refresh_token', tokens.refresh_token)
}

function clearTokens(): void {
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
}

export const useAuthStore = create<AuthState>((set) => ({
  isAuthenticated: false,
  isLoading: true,
  user: null,

  login: async (email, password) => {
    const tokens = await api.post<AuthTokenResponse>('/api/v1/auth/login', {
      email,
      password,
    })
    saveTokens(tokens)

    const user = await api.get<User>('/api/v1/users/me')
    set({ isAuthenticated: true, user })
  },

  register: async (email, password, displayName) => {
    const tokens = await api.post<AuthTokenResponse>('/api/v1/auth/register', {
      email,
      password,
      display_name: displayName,
    })
    saveTokens(tokens)

    const user = await api.get<User>('/api/v1/users/me')
    set({ isAuthenticated: true, user })
  },

  logout: async () => {
    try {
      await api.post('/api/v1/auth/logout')
    } finally {
      clearTokens()
      set({ isAuthenticated: false, user: null })
    }
  },

  checkAuth: async () => {
    const token = localStorage.getItem('access_token')
    if (!token) {
      set({ isAuthenticated: false, user: null, isLoading: false })
      return
    }

    try {
      const user = await api.get<User>('/api/v1/users/me')
      set({ isAuthenticated: true, user, isLoading: false })
    } catch {
      clearTokens()
      set({ isAuthenticated: false, user: null, isLoading: false })
    }
  },
}))
