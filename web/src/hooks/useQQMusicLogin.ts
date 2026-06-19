import { useState, useCallback, useRef } from 'react'
import { getLoginQRCode, checkQRStatus } from '../api/player'

type LoginStatus = 'idle' | 'loading' | 'pending_scan' | 'scanned' | 'confirmed' | 'expired' | 'error'

export function useQQMusicLogin() {
  const [loginStatus, setLoginStatus] = useState<LoginStatus>('idle')
  const [qrcodeUrl, setQrcodeUrl] = useState<string | null>(null)
  const [userName, setUserName] = useState<string | null>(null)
  const qrKeyRef = useRef<string | null>(null)

  const startLogin = useCallback(async () => {
    setLoginStatus('loading')
    try {
      const qr = await getLoginQRCode()
      setQrcodeUrl(qr.qrcode_url)
      qrKeyRef.current = qr.key
      setLoginStatus('pending_scan')
    } catch {
      setLoginStatus('error')
    }
  }, [])

  const checkStatus = useCallback(async () => {
    if (!qrKeyRef.current) return
    try {
      const status = await checkQRStatus(qrKeyRef.current)
      if (status.status === 'confirmed') {
        setLoginStatus('confirmed')
        setUserName(status.user_name || null)
        qrKeyRef.current = null
      } else if (status.status === 'scanned') {
        setLoginStatus('scanned')
      } else if (status.status === 'expired') {
        setLoginStatus('expired')
        qrKeyRef.current = null
      }
    } catch {
      // polling errors are expected, keep current state
    }
  }, [])

  const logout = useCallback(() => {
    setLoginStatus('idle')
    setQrcodeUrl(null)
    setUserName(null)
    qrKeyRef.current = null
  }, [])

  return {
    loginStatus,
    qrcodeUrl,
    userName,
    isLoggedIn: loginStatus === 'confirmed',
    startLogin,
    checkStatus,
    logout,
  }
}
