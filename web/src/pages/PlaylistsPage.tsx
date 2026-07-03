import { useState, useEffect, useCallback } from 'react'
import { useNavigate, useParams, Outlet } from 'react-router-dom'
import { Plus, Trash2, Music, Play, Pencil, Check, X } from 'lucide-react'
import { api, type PlaylistSummary, type PlaylistDetail, type PlaylistSong } from '../api/client'
import { usePlayerStore } from '../hooks/usePlayerStore'
import type { Song } from '../types'

function playlistSongToSong(s: PlaylistSong): Song {
  return {
    id: s.songId,
    title: s.title,
    artist: s.artist,
    coverUrl: s.coverUrl,
  }
}

export default function PlaylistsPage() {
  return <Outlet />
}

export function PlaylistList() {
  const navigate = useNavigate()
  const [playlists, setPlaylists] = useState<PlaylistSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [newName, setNewName] = useState('')
  const [creating, setCreating] = useState(false)
  const [renamingId, setRenamingId] = useState<string | null>(null)
  const [renameValue, setRenameValue] = useState('')

  const load = useCallback(async () => {
    try {
      const list = await api.playlists.list()
      setPlaylists(list)
    } catch { /* ignore */ }
    setLoading(false)
  }, [])

  useEffect(() => { load() }, [load])

  const handleCreate = useCallback(async () => {
    const name = newName.trim()
    if (!name) return
    setCreating(true)
    try {
      const created = await api.playlists.create(name)
      setNewName('')
      setPlaylists(prev => [created as PlaylistSummary, ...prev])
    } catch { /* ignore */ }
    setCreating(false)
  }, [newName])

  const handleDelete = useCallback(async (id: string, e: React.MouseEvent) => {
    e.stopPropagation()
    try {
      await api.playlists.delete(id)
      setPlaylists(prev => prev.filter(p => p.id !== id))
    } catch { /* ignore */ }
  }, [])

  const startRename = useCallback((id: string, name: string, e: React.MouseEvent) => {
    e.stopPropagation()
    setRenamingId(id)
    setRenameValue(name)
  }, [])

  const handleRename = useCallback(async (id: string) => {
    const name = renameValue.trim()
    if (!name) return
    try {
      await api.playlists.rename(id, name)
      setPlaylists(prev => prev.map(p => p.id === id ? { ...p, name } : p))
      setRenamingId(null)
    } catch { /* ignore */ }
  }, [renameValue])

  const cancelRename = useCallback(() => {
    setRenamingId(null)
    setRenameValue('')
  }, [])

  return (
    <div className="p-6 max-w-2xl mx-auto">
      <h2 className="text-xl font-bold text-gray-200 mb-4">我的歌单</h2>

      <div className="flex gap-2 mb-6">
        <input
          type="text"
          value={newName}
          onChange={e => setNewName(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter') handleCreate() }}
          placeholder="新建歌单名称..."
          className="flex-1 bg-gray-800 text-gray-100 rounded-lg px-3 py-2 outline-none focus:ring-2 focus:ring-indigo-500 placeholder-gray-500 text-sm"
        />
        <button
          onClick={handleCreate}
          disabled={!newName.trim() || creating}
          className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-500 disabled:opacity-50 transition-colors text-sm"
        >
          <Plus size={16} />
        </button>
      </div>

      {loading ? (
        <p className="text-gray-500 text-sm">加载中...</p>
      ) : playlists.length === 0 ? (
        <p className="text-gray-500 text-sm">还没有歌单，创建一个吧</p>
      ) : (
        <div className="space-y-2">
          {playlists.map(p => (
            <div
              key={p.id}
              onClick={() => { if (!renamingId) navigate(`/playlists/${p.id}`) }}
              className="flex items-center gap-3 bg-gray-800 rounded-lg p-3 cursor-pointer hover:bg-gray-700 transition-colors"
            >
              <Music size={18} className="text-gray-400 flex-shrink-0" />
              {renamingId === p.id ? (
                <>
                  <input
                    type="text"
                    value={renameValue}
                    onChange={e => setRenameValue(e.target.value)}
                    onKeyDown={e => { if (e.key === 'Enter') handleRename(p.id); if (e.key === 'Escape') cancelRename() }}
                    onClick={e => e.stopPropagation()}
                    className="flex-1 bg-gray-700 text-gray-100 rounded px-2 py-1 outline-none text-sm"
                    autoFocus
                  />
                  <button onClick={e => { e.stopPropagation(); handleRename(p.id) }} className="text-green-400 hover:text-green-300 p-1"><Check size={14} /></button>
                  <button onClick={e => { e.stopPropagation(); cancelRename() }} className="text-gray-400 hover:text-white p-1"><X size={14} /></button>
                </>
              ) : (
                <>
                  <span className="flex-1 text-sm text-gray-200 truncate">{p.name}</span>
                  <button
                    onClick={(e) => startRename(p.id, p.name, e)}
                    className="text-gray-500 hover:text-blue-400 p-1"
                    title="重命名"
                  >
                    <Pencil size={14} />
                  </button>
                  <button
                    onClick={(e) => handleDelete(p.id, e)}
                    className="text-gray-500 hover:text-red-400 p-1"
                    title="删除歌单"
                  >
                    <Trash2 size={14} />
                  </button>
                </>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export function PlaylistDetailView() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [playlist, setPlaylist] = useState<PlaylistDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const { play, state } = usePlayerStore()

  useEffect(() => {
    if (!id) return
    api.playlists.get(id).then(setPlaylist).catch((err) => { console.error('[PlaylistsPage] get failed:', err) }).finally(() => setLoading(false))
  }, [id])

  const playSong = useCallback((s: PlaylistSong) => {
    play(playlistSongToSong(s))
  }, [play])

  if (loading) return <div className="p-6"><p className="text-gray-500 text-sm">加载中...</p></div>
  if (!playlist) return <div className="p-6"><p className="text-gray-500 text-sm">歌单未找到</p></div>

  return (
    <div className="p-6 max-w-2xl mx-auto">
      <button onClick={() => navigate('/playlists')} className="text-gray-400 hover:text-gray-200 text-sm mb-4 block">
        ← 返回歌单列表
      </button>
      <h2 className="text-xl font-bold text-gray-200 mb-4">{playlist.name}</h2>

      {playlist.songs.length === 0 ? (
        <p className="text-gray-500 text-sm">歌单中还没有歌曲</p>
      ) : (
        <div className="space-y-1">
          {playlist.songs.map((s, i) => (
            <div
              key={s.songId + i}
              className={`flex items-center gap-3 p-2 rounded-lg cursor-pointer transition-colors
                ${state.currentSong?.id === s.songId ? 'bg-indigo-600/20 ring-1 ring-indigo-500' : 'hover:bg-gray-800'}`}
              onClick={() => playSong(s)}
            >
              <span className="text-xs text-gray-500 w-6 text-right">{i + 1}</span>
              <div className="min-w-0 flex-1">
                <p className="text-sm text-gray-200 truncate">{s.title}</p>
                <p className="text-xs text-gray-400 truncate">{s.artist}</p>
              </div>
              <Play size={14} className="text-gray-500 flex-shrink-0" />
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
