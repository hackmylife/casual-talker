import { useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { useAuthStore } from '@/stores/auth-store'
import { ApiError } from '@/lib/api-client'
import { LoadingSpinner } from '@/components/common/LoadingSpinner'

export default function Login() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const { login } = useAuthStore()
  const navigate = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setIsSubmitting(true)

    try {
      await login(email, password)
      navigate('/', { replace: true })
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 401) {
          setError('メールアドレスまたはパスワードが正しくありません')
        } else {
          setError('ログインに失敗しました。しばらくしてからお試しください')
        }
      } else {
        setError('ネットワークエラーが発生しました')
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="min-h-svh bg-neutral-50 flex items-center justify-center px-4">
      <div className="w-full max-w-sm">
        {/* App name */}
        <div className="text-center mb-10">
          <h1 className="text-2xl font-bold text-neutral-900 tracking-tight">
            casual talker
          </h1>
          <p className="mt-2 text-sm text-neutral-600">
            英会話を、もっと気軽に。
          </p>
        </div>

        <form onSubmit={handleSubmit} noValidate>
          {/* Error message */}
          {error && (
            <div className="mb-4 px-4 py-3 rounded-xl bg-red-50 border border-red-200">
              <p className="text-sm text-red-600">{error}</p>
            </div>
          )}

          {/* Email field */}
          <div className="mb-4">
            <label
              htmlFor="email"
              className="block text-sm font-medium text-neutral-800 mb-1.5"
            >
              メールアドレス
            </label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              autoComplete="email"
              placeholder="example@email.com"
              className="w-full border border-neutral-300 rounded-xl px-4 py-3 bg-white text-neutral-900 placeholder:text-neutral-300 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent transition"
            />
          </div>

          {/* Password field */}
          <div className="mb-6">
            <label
              htmlFor="password"
              className="block text-sm font-medium text-neutral-800 mb-1.5"
            >
              パスワード
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              autoComplete="current-password"
              placeholder="パスワードを入力"
              className="w-full border border-neutral-300 rounded-xl px-4 py-3 bg-white text-neutral-900 placeholder:text-neutral-300 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent transition"
            />
          </div>

          {/* Submit button */}
          <button
            type="submit"
            disabled={isSubmitting}
            className="w-full bg-primary-600 text-white rounded-2xl h-14 font-medium text-base flex items-center justify-center gap-2 hover:bg-primary-700 active:bg-primary-700 transition disabled:opacity-60 disabled:cursor-not-allowed"
          >
            {isSubmitting ? (
              <>
                <LoadingSpinner size="sm" color="border-white" />
                <span>ログイン中...</span>
              </>
            ) : (
              'ログイン'
            )}
          </button>
        </form>

        {/* Link to register */}
        <p className="mt-6 text-center text-sm text-neutral-600">
          アカウントをお持ちでない方は{' '}
          <Link
            to="/register"
            className="text-primary-600 font-medium hover:underline"
          >
            新規登録
          </Link>
        </p>
      </div>
    </div>
  )
}
