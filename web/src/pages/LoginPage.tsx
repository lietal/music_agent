import { useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { api, setToken } from '../api/client'

export default function LoginPage() {
  const [isRegister, setIsRegister] = useState(false)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [error, setError] = useState('')
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    try {
      const result = isRegister
        ? await api.auth.register(username, password, displayName || undefined)
        : await api.auth.login(username, password)

      setToken(result.token)
      const redirect = searchParams.get('redirect') || '/chat'
      navigate(redirect, { replace: true })
    } catch {
      setError(isRegister ? '注册失败，用户名可能已被占用' : '用户名或密码错误')
    }
  }

  return (
    <div className="flex items-center justify-center h-screen bg-gray-950">
      <div className="text-center w-full max-w-sm">
        <h1 className="text-3xl font-bold text-white mb-4">Music Agent</h1>
        <p className="text-gray-400 mb-8">你的音乐AI助手</p>

        <form onSubmit={handleSubmit} className="space-y-4">
          <input
            type="text"
            placeholder="用户名"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            className="w-full px-4 py-3 bg-gray-800 text-white rounded-lg border border-gray-700 focus:border-green-500 focus:outline-none"
            required
          />
          <input
            type="password"
            placeholder="密码"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full px-4 py-3 bg-gray-800 text-white rounded-lg border border-gray-700 focus:border-green-500 focus:outline-none"
            required
          />
          {isRegister && (
            <input
              type="text"
              placeholder="显示名称（可选）"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              className="w-full px-4 py-3 bg-gray-800 text-white rounded-lg border border-gray-700 focus:border-green-500 focus:outline-none"
            />
          )}

          {error && <p className="text-red-400 text-sm">{error}</p>}

          <button
            type="submit"
            className="w-full px-8 py-3 bg-green-600 text-white rounded-lg hover:bg-green-500 text-lg"
          >
            {isRegister ? '注册' : '登录'}
          </button>
        </form>

        <p className="text-gray-400 mt-4 text-sm">
          {isRegister ? (
            <>
              已有账号？{' '}
              <button
                onClick={() => setIsRegister(false)}
                className="text-green-400 hover:underline"
              >
                登录
              </button>
            </>
          ) : (
            <>
              没有账号？{' '}
              <button
                onClick={() => setIsRegister(true)}
                className="text-green-400 hover:underline"
              >
                注册
              </button>
            </>
          )}
        </p>
      </div>
    </div>
  )
}
