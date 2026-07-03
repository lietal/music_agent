import { useEffect, useState } from 'react'
import { MessageCircle } from 'lucide-react'

interface CommentData {
  comments: Array<{
    id?: string
    content: string
    nickname: string
    avatar?: string
    time?: string
    like_count?: number
  }>
}

export default function CommentsPanel({ songId, visible }: { songId: string | null; visible: boolean }) {
  const [comments, setComments] = useState<CommentData['comments']>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!songId || !visible) return
    setLoading(true)
    setError(null)
    fetch(`/api/player/comments/${encodeURIComponent(songId)}`)
      .then(r => r.json())
      .then((data: CommentData) => setComments(data.comments || []))
      .catch(e => setError(e.message))
      .finally(() => setLoading(false))
  }, [songId, visible])

  if (!visible) return null

  return (
    <div className="p-3 space-y-2 overflow-auto h-full">
      <h3 className="text-xs font-semibold text-gray-400 uppercase tracking-wide">Comments</h3>
      {loading && <p className="text-xs text-gray-500">Loading...</p>}
      {error && <p className="text-xs text-red-400">{error}</p>}
      {!loading && !error && comments.length === 0 && (
        <p className="text-xs text-gray-500 text-center py-4">No comments yet</p>
      )}
      {comments.map((c, i) => (
        <div key={c.id || i} className="bg-gray-800 rounded-lg p-2.5 text-xs">
          <div className="flex items-center gap-2 mb-1">
            <div className="w-5 h-5 rounded-full bg-indigo-600 flex items-center justify-center flex-shrink-0">
              {c.avatar ? (
                <img src={c.avatar} alt="" className="w-5 h-5 rounded-full" />
              ) : (
                <MessageCircle size={10} className="text-white" />
              )}
            </div>
            <span className="text-gray-300 font-medium">{c.nickname}</span>
            {c.time && <span className="text-gray-600 ml-auto">{c.time}</span>}
          </div>
          <p className="text-gray-400 leading-relaxed break-all">{c.content}</p>
        </div>
      ))}
    </div>
  )
}
