import { useEffect } from 'react'
import { useQQMusicLogin } from '../hooks/useQQMusicLogin'

export default function LoginPage() {
  const { loginStatus, qrcodeUrl, userName, errorMsg, startLogin, checkStatus, logout } = useQQMusicLogin()

  useEffect(() => {
    if (loginStatus !== 'pending_scan' && loginStatus !== 'scanned') return
    const interval = setInterval(() => {
      checkStatus()
    }, 2000)
    return () => clearInterval(interval)
  }, [loginStatus, checkStatus])

  return (
    <div className="flex items-center justify-center h-screen bg-gray-950">
      <div className="text-center w-full max-w-sm">
        <h1 className="text-3xl font-bold text-white mb-4">Music Agent</h1>
        <p className="text-gray-400 mb-8">用 QQ 音乐扫码登录</p>

        {loginStatus === 'idle' && (
          <button
            onClick={startLogin}
            className="w-full px-8 py-3 bg-green-600 text-white rounded-lg hover:bg-green-500 text-lg"
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
              className="w-full px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-500 text-sm"
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
              className="w-full px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-500 text-sm"
            >
              重试
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
