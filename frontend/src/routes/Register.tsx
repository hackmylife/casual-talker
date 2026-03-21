import { useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { useAuthStore } from '@/stores/auth-store'
import { ApiError } from '@/lib/api-client'
import { LoadingSpinner } from '@/components/common/LoadingSpinner'

export default function Register() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const { register } = useAuthStore()
  const navigate = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    if (password.length < 8) {
      setError('パスワードは8文字以上で入力してください')
      return
    }

    setIsSubmitting(true)

    try {
      await register(email, password, displayName)
      navigate('/', { replace: true })
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 403) {
          setError('このメールアドレスは登録できません。招待されたメールアドレスをご確認ください')
        } else if (err.status === 409) {
          setError('このメールアドレスはすでに登録されています')
        } else if (err.status === 422) {
          setError('入力内容をご確認ください')
        } else {
          setError('登録に失敗しました。しばらくしてからお試しください')
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

        {/* Whitelist notice */}
        <div className="mb-5 px-4 py-2.5 rounded-xl bg-neutral-100 border border-neutral-300">
          <p className="text-xs text-neutral-600 text-center">
            招待されたメールアドレスのみ登録できます
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
          <div className="mb-4">
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
              autoComplete="new-password"
              placeholder="8文字以上で入力"
              className="w-full border border-neutral-300 rounded-xl px-4 py-3 bg-white text-neutral-900 placeholder:text-neutral-300 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent transition"
            />
            <p className="mt-1.5 text-xs text-neutral-600">
              最低8文字以上のパスワードを設定してください
            </p>
          </div>

          {/* Display name field (optional) */}
          <div className="mb-6">
            <label
              htmlFor="displayName"
              className="block text-sm font-medium text-neutral-800 mb-1.5"
            >
              表示名
              <span className="ml-1.5 text-xs font-normal text-neutral-600">
                （任意）
              </span>
            </label>
            <input
              id="displayName"
              type="text"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              autoComplete="nickname"
              placeholder="アプリ内での表示名"
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
                <span>登録中...</span>
              </>
            ) : (
              '新規登録'
            )}
          </button>
        </form>

        {/* Link to login */}
        <p className="mt-6 text-center text-sm text-neutral-600">
          すでにアカウントをお持ちの方は{' '}
          <Link
            to="/login"
            className="text-primary-600 font-medium hover:underline"
          >
            ログイン
          </Link>
        </p>
      </div>
    </div>
  )
}
