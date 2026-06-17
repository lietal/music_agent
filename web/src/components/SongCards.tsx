import { Music } from 'lucide-react'
import type { Song } from '../types'

export type { Song }

export default function SongCards({ songs }: { songs: Song[] }) {
  return (
    <div className="grid grid-cols-2 gap-2 mt-2">
      {songs.map((song: Song) => (
        <div
          key={song.id}
          className="flex items-center gap-3 bg-gray-700/50 rounded-lg p-2 hover:bg-gray-700 transition-colors"
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
          <div className="min-w-0">
            <p className="text-sm text-gray-100 truncate">{song.title}</p>
            <p className="text-xs text-gray-400 truncate">{song.artist}</p>
          </div>
        </div>
      ))}
    </div>
  )
}
