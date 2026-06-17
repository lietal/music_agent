import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { MessageSquare, Trash2, Loader2 } from 'lucide-react'
import { api, type Conversation } from '../api/client'

export default function HistoryPage() {
  const navigate = useNavigate()
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    api.conversations.list()
      .then((data) => {
        if (!cancelled) setConversations(data)
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to load')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => { cancelled = true }
  }, [])

  const handleDelete = async (id: string) => {
    try {
      await api.conversations.delete(id)
      setConversations((prev) => prev.filter((c) => c.id !== id))
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to delete')
    }
  }

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return ''
    return new Date(dateStr).toLocaleDateString('zh-CN', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  return (
    <div className="p-6 max-w-2xl mx-auto">
      <h2 className="text-xl font-bold mb-6">History</h2>

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

      {!loading && !error && conversations.length === 0 && (
        <p className="text-gray-500 text-center py-12">No conversations yet.</p>
      )}

      {!loading && conversations.length > 0 && (
        <div className="space-y-2">
          {conversations.map((conv) => (
            <div
              key={conv.id}
              className="flex items-center gap-3 p-3 rounded-lg bg-gray-900 border border-gray-800 hover:border-gray-700 transition-colors group"
            >
              <button
                onClick={() => navigate(`/chat?conversationId=${conv.id}`)}
                className="flex-1 flex items-center gap-3 text-left min-w-0"
              >
                <MessageSquare size={18} className="text-gray-500 shrink-0" />
                <div className="min-w-0">
                  <p className="text-sm text-gray-200 truncate">
                    {conv.title || 'Untitled'}
                  </p>
                  {conv.createdAt && (
                    <p className="text-xs text-gray-500 mt-0.5">
                      {formatDate(conv.createdAt)}
                    </p>
                  )}
                </div>
              </button>
              <button
                onClick={() => handleDelete(conv.id)}
                className="p-1.5 rounded text-gray-600 hover:text-red-400 hover:bg-red-950/50 opacity-0 group-hover:opacity-100 transition-all"
                title="Delete"
              >
                <Trash2 size={16} />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
