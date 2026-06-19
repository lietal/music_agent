import { useState, useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { api, setToken } from '../api/client'
import { useQQMusicLogin } from '../hooks/useQQMusicLogin'

export default function LoginPage() {
  const [isRegister, setIsRegister] = useState(false)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [error, setError] = useState('')
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  const { loginStatus, qrcodeUrl, userName, errorMsg, isLoggedIn, startLogin, checkStatus, logout } = useQQMusicLogin()

  useEffect(() => {
    if (loginStatus !== 'pending_scan' && loginStatus !== 'scanned') return
    const interval = setInterval(() => {
      checkStatus()
    }, 2000)
    return () => clearInterval(interval)
  }, [loginStatus, checkStatus])

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

        <hr className="my-6 border-gray-700" />

        <div className="text-left">
          <h2 className="text-lg font-semibold text-white mb-3">QQ 音乐登录</h2>

          {loginStatus === 'idle' && (
            <button
              onClick={startLogin}
              className="w-full px-8 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-500 text-lg"
            >
              登录 QQ 音乐
            </button>
          )}

          {loginStatus === 'loading' && (
            <p className="text-gray-400 text-sm">加载中...</p>
          )}

          {(loginStatus === 'pending_scan' || loginStatus === 'scanned') && qrcodeUrl && (
            <div className="space-y-3">
              <img
                src={qrcodeUrl}
                alt="QQ 音乐登录二维码"
                className="mx-auto w-48 h-48 rounded-lg border border-gray-700 bg-white p-2"
              />
              <p className="text-gray-400 text-sm text-center">
                {loginStatus === 'pending_scan' ? '请用QQ音乐App扫描二维码' : '已扫码，确认中...'}
              </p>
            </div>
          )}

          {loginStatus === 'confirmed' && (
            <div className="space-y-3">
              <p className="text-green-400 text-sm flex items-center justify-center gap-1">
                <span>&#10003;</span> 已登录: {userName}
              </p>
              <button
                onClick={logout}
                className="w-full px-4 py-2 bg-gray-700 text-gray-300 rounded-lg hover:bg-gray-600 text-sm"
              >
                登出
              </button>
            </div>
          )}

          {loginStatus === 'expired' && (
            <div className="space-y-2">
              <p className="text-yellow-400 text-sm text-center">二维码已过期</p>
              <button
                onClick={startLogin}
                className="w-full px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-500 text-sm"
              >
                重新获取
              </button>
            </div>
          )}

          {loginStatus === 'error' && (
            <div className="space-y-2">
              <p className="text-red-400 text-sm text-center">{errorMsg || '获取二维码失败'}</p>
              <button
                onClick={startLogin}
                className="w-full px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-500 text-sm"
              >
                重试
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
