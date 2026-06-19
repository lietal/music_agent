import { X } from 'lucide-react'
import type { Song } from '../types'

export type { Song }

interface QueuePanelProps {
  queue: Song[]
  currentIndex: number
  onPlaySong: (song: Song) => void
  onRemove: (index: number) => void
}

export default function QueuePanel({ queue, currentIndex, onPlaySong, onRemove }: QueuePanelProps) {
  if (queue.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-gray-500 text-sm">
        队列为空
      </div>
    )
  }

  return (
    <div className="overflow-y-auto h-full">
      <ul className="space-y-1 p-2">
        {queue.map((song, i) => {
          const isCurrent = i === currentIndex
          return (
            <li
              key={`${song.id}-${i}`}
              className={`flex items-center gap-3 rounded-lg p-2 hover:bg-gray-700/50 transition-colors cursor-pointer ${isCurrent ? 'text-green-400' : 'text-gray-100'}`}
              onClick={() => {
                if (!isCurrent) onPlaySong(song)
              }}
            >
              <span className="w-6 text-center text-sm text-gray-500 flex-shrink-0">
                {i + 1}
              </span>
              <div className="min-w-0 flex-1">
                <p className="text-sm truncate">{song.title}</p>
                <p className="text-xs text-gray-400 truncate">{song.artist}</p>
              </div>
              <button
                aria-label={`Remove ${song.title}`}
                onClick={(e) => {
                  e.stopPropagation()
                  onRemove(i)
                }}
                className="flex-shrink-0 p-1 hover:text-red-400 transition-colors"
              >
                <X size={14} />
              </button>
            </li>
          )
        })}
      </ul>
    </div>
  )
}
