import { useEffect } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router'
import { AppShell } from '@/components/layout/AppShell'
import { AuthGuard } from '@/components/auth/AuthGuard'
import { LoadingScreen } from '@/components/common/LoadingSpinner'
import { useAuthStore } from '@/stores/auth-store'
import Home from '@/routes/Home'
import Login from '@/routes/Login'
import Register from '@/routes/Register'
import Session from '@/routes/Session'
import Feedback from '@/routes/Feedback'
import History from '@/routes/History'

function App() {
  const { checkAuth, isLoading } = useAuthStore()

  useEffect(() => {
    checkAuth()
  }, [checkAuth])

  if (isLoading) {
    return <LoadingScreen />
  }

  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route
          element={
            <AuthGuard>
              <AppShell />
            </AuthGuard>
          }
        >
          <Route path="/" element={<Home />} />
          <Route path="/session/:id" element={<Session />} />
          <Route path="/feedback/:id" element={<Feedback />} />
          <Route path="/history" element={<History />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
