import { useState, useCallback } from 'react'

export function useWeChatLogin() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const startWeChatLogin = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch('/api/auth/wechat/qr')
      const data = await res.json()
      if (data.qrcode_url && data.type === 'wechat_oauth_url') {
        window.location.href = data.qrcode_url
      } else {
        setError(data.error || 'WeChat login not available')
      }
    } catch (e: any) {
      setError(e?.message || '网络错误')
    } finally {
      setLoading(false)
    }
  }, [])

  return { loading, error, startWeChatLogin }
}
