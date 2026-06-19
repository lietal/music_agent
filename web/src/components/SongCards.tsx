import { Music, Play, Plus } from 'lucide-react'
import type { Song } from '../types'

export type { Song }

interface SongCardsProps {
  songs: Song[]
  onPlay?: (song: Song) => void
  onAddToQueue?: (song: Song) => void
  currentSongId?: string | null
}

export default function SongCards({ songs, onPlay, onAddToQueue, currentSongId }: SongCardsProps) {
  return (
    <div className="grid grid-cols-2 gap-2 mt-2">
      {songs.map((song: Song) => {
        const isCurrent = currentSongId === song.id
        return (
          <div
            key={song.id}
            className={`flex items-center gap-3 bg-gray-700/50 rounded-lg p-2 transition-colors cursor-pointer
              ${isCurrent ? 'ring-1 ring-green-500 bg-gray-700' : 'hover:bg-gray-700'}`}
            onClick={() => onPlay?.(song)}
          >
            {song.coverUrl ? (
              <img
                src={song.coverUrl}
                alt={song.title}
                className="w-10 h-10 rounded object-cover flex-shrink-0"
              />
            ) : (
              <div className="w-10 h-10 rounded bg-gray-600 flex items-center justify-center flex-shrink-0">
                <Music size={16} className="text-gray-400" />
              </div>
            )}
            <div className="min-w-0 flex-1">
              <p className="text-sm text-gray-100 truncate">{song.title}</p>
              <p className="text-xs text-gray-400 truncate">{song.artist}</p>
            </div>
            <button
              onClick={e => { e.stopPropagation(); onAddToQueue?.(song) }}
              className="flex-shrink-0 text-gray-500 hover:text-gray-200 p-1"
              title="Add to queue"
            >
              <Plus size={16} />
            </button>
            <button
              onClick={e => { e.stopPropagation(); onPlay?.(song) }}
              className={`flex-shrink-0 p-1 ${isCurrent ? 'text-green-400' : 'text-gray-500 hover:text-gray-200'}`}
              title="Play"
            >
              <Play size={16} />
            </button>
          </div>
        )
      })}
    </div>
  )
}
