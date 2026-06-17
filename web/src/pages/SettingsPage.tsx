import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { User, Music, LogOut, Loader2 } from 'lucide-react'
import { api, clearToken } from '../api/client'

interface UserInfo {
  id: string
  displayName: string
  provider?: string
}

export default function SettingsPage() {
  const navigate = useNavigate()
  const [user, setUser] = useState<UserInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api.auth.me()
      .then(setUser)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [])

  const handleLogout = () => {
    clearToken()
    navigate('/login')
  }

  return (
    <div className="p-6 max-w-2xl mx-auto">
      <h2 className="text-xl font-bold mb-6">Settings</h2>

      {loading && (
        <div className="flex items-center justify-center py-12 text-gray-400">
          <Loader2 className="animate-spin mr-2" size={20} />
          Loading...
        </div>
      )}

      {error && (
        <div className="bg-red-900/30 border border-red-800 text-red-300 rounded-lg p-3 mb-4 text-sm">
          {error}
        </div>
      )}

      {user && (
        <>
          <section className="mb-8">
            <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-3">
              Account
            </h3>
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-4">
              <div className="flex items-center gap-3 mb-3">
                <div className="w-10 h-10 rounded-full bg-indigo-600 flex items-center justify-center">
                  <User size={20} className="text-white" />
                </div>
                <div>
                  <p className="text-gray-100 font-medium">{user.displayName}</p>
                  {user.provider && (
                    <p className="text-xs text-gray-500">
                      {user.provider}
                    </p>
                  )}
                </div>
              </div>
              <p className="text-xs text-gray-600">ID: {user.id}</p>
            </div>
          </section>

          <section className="mb-8">
            <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-3">
              Music Source
            </h3>
            <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 flex items-center gap-3">
              <Music size={20} className="text-gray-500" />
              <div>
                <p className="text-gray-100 text-sm">Coming soon</p>
                <p className="text-xs text-gray-600">Music source integration will be available here.</p>
              </div>
            </div>
          </section>

          <section>
            <button
              onClick={handleLogout}
              className="flex items-center gap-2 px-4 py-2 bg-red-600/20 border border-red-800 text-red-400 rounded-lg hover:bg-red-600/30 transition-colors text-sm"
            >
              <LogOut size={16} />
              Logout
            </button>
          </section>
        </>
      )}
    </div>
  )
}
