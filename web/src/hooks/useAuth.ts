import { useCallback, useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { getToken, setToken, clearToken } from '../api/client'

export function useAuth() {
  const navigate = useNavigate()
  const location = useLocation()

  const login = useCallback(() => {
    window.location.href = '/api/auth/wechat'
  }, [])

  const logout = useCallback(() => {
    clearToken()
    navigate('/login')
  }, [navigate])

  const isAuthenticated = useCallback((): boolean => {
    const token = getToken()
    if (!token) return false

    try {
      const parts = token.split('.')
      if (parts.length !== 3) return false

      const base64Url = parts[1]
      const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/')
      const jsonPayload = decodeURIComponent(
        atob(base64)
          .split('')
          .map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
          .join(''),
      )
      const payload = JSON.parse(jsonPayload)

      if (payload.exp && payload.exp * 1000 < Date.now()) {
        clearToken()
        return false
      }
      return true
    } catch {
      return false
    }
  }, [])

  useEffect(() => {
    if (location.pathname !== '/login' && !isAuthenticated()) {
      navigate('/login')
    }
  }, [location.pathname, navigate, isAuthenticated])

  return { login, logout, getToken, setToken, isAuthenticated }
}
