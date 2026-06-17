import { useEffect } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { setToken } from '../api/client'
import { useAuth } from '../hooks/useAuth'

export default function LoginPage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { login } = useAuth()

  useEffect(() => {
    const token = searchParams.get('token')
    if (token) {
      setToken(token)
      navigate('/chat', { replace: true })
    }
  }, [searchParams, navigate])

  return (
    <div className="flex items-center justify-center h-screen bg-gray-950">
      <div className="text-center">
        <h1 className="text-3xl font-bold text-white mb-4">Music Agent</h1>
        <p className="text-gray-400 mb-8">你的音乐AI助手</p>
        <button
          onClick={login}
          className="px-8 py-3 bg-green-600 text-white rounded-lg hover:bg-green-500 text-lg"
        >
          微信登录
        </button>
      </div>
    </div>
  )
}
