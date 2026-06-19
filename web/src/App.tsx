import { Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import LoginPage from './pages/LoginPage'
import ChatPage from './pages/ChatPage'
import HistoryPage from './pages/HistoryPage'
import SettingsPage from './pages/SettingsPage'

function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route element={<Layout />}>
        <Route path="/chat" element={<ChatPage />} />
        <Route path="/history" element={<HistoryPage />} />
        <Route path="/settings" element={<SettingsPage />} />
      </Route>
      <Route path="*" element={<LoginPage />} />
    </Routes>
  )
}

export default App
