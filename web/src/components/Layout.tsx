import { Outlet, NavLink, useNavigate } from 'react-router-dom'
import { MessageCircle, History, Settings, LogOut } from 'lucide-react'
import { getToken, clearToken } from '../api/client'

export default function Layout() {
  const navigate = useNavigate()
  const token = getToken()

  if (!token) {
    navigate('/login')
    return null
  }

  const handleLogout = () => {
    clearToken()
    navigate('/login')
  }

  return (
    <div className="flex h-screen bg-gray-950 text-gray-100">
      <nav className="w-16 flex flex-col items-center py-4 border-r border-gray-800 gap-4">
        <NavLink to="/chat" className={({ isActive }) => `p-2 rounded-lg ${isActive ? 'bg-indigo-600 text-white' : 'text-gray-400 hover:text-white'}`}>
          <MessageCircle size={24} />
        </NavLink>
        <NavLink to="/history" className={({ isActive }) => `p-2 rounded-lg ${isActive ? 'bg-indigo-600 text-white' : 'text-gray-400 hover:text-white'}`}>
          <History size={24} />
        </NavLink>
        <NavLink to="/settings" className={({ isActive }) => `p-2 rounded-lg ${isActive ? 'bg-indigo-600 text-white' : 'text-gray-400 hover:text-white'}`}>
          <Settings size={24} />
        </NavLink>
        <div className="mt-auto">
          <button onClick={handleLogout} className="p-2 rounded-lg text-gray-400 hover:text-red-400">
            <LogOut size={24} />
          </button>
        </div>
      </nav>
      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  )
}
