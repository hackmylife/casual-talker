import { type ReactNode } from 'react'
import { Navigate } from 'react-router'
import { useAuthStore } from '@/stores/auth-store'
import { LoadingScreen } from '@/components/common/LoadingSpinner'

interface AuthGuardProps {
  children: ReactNode
}

export function AuthGuard({ children }: AuthGuardProps) {
  const { isAuthenticated, isLoading } = useAuthStore()

  if (isLoading) {
    return <LoadingScreen />
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}
