import { useState, useEffect, useCallback } from 'react'
import { X, Plus } from 'lucide-react'
import { api, type PlaylistSummary } from '../api/client'
import type { Song } from '../types'

interface PlaylistPickerModalProps {
  song: Song
  onClose: () => void
}

export default function PlaylistPickerModal({ song, onClose }: PlaylistPickerModalProps) {
  const [playlists, setPlaylists] = useState<PlaylistSummary[]>([])
  const [newName, setNewName] = useState('')
  const [savingId, setSavingId] = useState<string | null>(null)

  useEffect(() => {
    api.playlists.list().then(setPlaylists).catch((err) => { console.error('[PlaylistPicker] list failed:', err) })
  }, [])

  const saveTo = useCallback(async (playlistId: string) => {
    setSavingId(playlistId)
    try {
      await api.playlists.addSong(playlistId, {
        songId: song.id,
        title: song.title,
        artist: song.artist,
        coverUrl: song.coverUrl,
      })
    } catch { /* ignore */ }
    setSavingId(null)
    onClose()
  }, [song, onClose])

  const createAndSave = useCallback(async () => {
    const name = newName.trim()
    if (!name) return
    try {
      const created = await api.playlists.create(name, [{
        songId: song.id,
        title: song.title,
        artist: song.artist,
        coverUrl: song.coverUrl,
      }])
      if (created) onClose()
    } catch { /* ignore */ }
  }, [newName, song, onClose])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={onClose}>
      <div className="bg-gray-800 rounded-xl p-4 w-80 max-h-[70vh] overflow-auto" onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-medium text-gray-200">添加到歌单</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-white"><X size={16} /></button>
        </div>

        <div className="flex gap-2 mb-3">
          <input
            type="text"
            value={newName}
            onChange={e => setNewName(e.target.value)}
            onKeyDown={e => { if (e.key === 'Enter') createAndSave() }}
            placeholder="新建歌单..."
            className="flex-1 bg-gray-700 text-gray-100 rounded px-2 py-1 outline-none text-xs placeholder-gray-500"
          />
          <button
            onClick={createAndSave}
            disabled={!newName.trim()}
            className="px-3 py-1 bg-indigo-600 text-white rounded text-xs hover:bg-indigo-500 disabled:opacity-50"
          >
            <Plus size={12} />
          </button>
        </div>

        <div className="space-y-1">
          {playlists.map(p => (
            <button
              key={p.id}
              onClick={() => saveTo(p.id)}
              disabled={savingId === p.id}
              className="w-full text-left px-2 py-1.5 rounded text-sm text-gray-300 hover:bg-gray-700 disabled:opacity-50 transition-colors"
            >
              {savingId === p.id ? '保存中...' : p.name}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
