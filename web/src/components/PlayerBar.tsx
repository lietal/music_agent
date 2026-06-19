import { Play, Pause, SkipForward } from 'lucide-react'
import type { Song } from '../types'

interface PlayerBarProps {
  currentSong: Song | null
  isPlaying: boolean
  currentTime: number
  duration: number
  onTogglePlay: () => void
  onTogglePanel: () => void
  onNext: () => void
}

function formatTime(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${m}:${s.toString().padStart(2, '0')}`
}

export default function PlayerBar({
  currentSong,
  isPlaying,
  currentTime,
  duration,
  onTogglePlay,
  onTogglePanel,
  onNext,
}: PlayerBarProps) {
  if (!currentSong) return null

  const progress = duration > 0 ? (currentTime / duration) * 100 : 0

  return (
    <div
      data-testid="player-bar"
      className="fixed bottom-0 left-0 right-0 bg-gray-900 border-t border-gray-700 p-2 z-50"
      onClick={onTogglePanel}
    >
      <div className="flex items-center gap-2">
        <div className="min-w-0 flex-1">
          <p className="text-sm text-gray-100 truncate">{currentSong.title}</p>
          <p className="text-xs text-gray-400 truncate">{currentSong.artist}</p>
        </div>
        <button onClick={(e) => { e.stopPropagation(); onTogglePlay() }} aria-label={isPlaying ? 'Pause' : 'Play'}>
          {isPlaying ? <Pause data-testid="pause-icon" size={20} /> : <Play data-testid="play-icon" size={20} />}
        </button>
        <button onClick={(e) => { e.stopPropagation(); onNext() }} aria-label="Next">
          <SkipForward size={20} />
        </button>
      </div>
      <div className="flex items-center gap-2 mt-1">
        <span className="text-xs text-gray-400">{formatTime(currentTime)}</span>
        <div className="flex-1 h-1 bg-gray-600 rounded">
          <div className="h-1 bg-green-500 rounded" style={{ width: `${progress}%` }} />
        </div>
        <span className="text-xs text-gray-400">{formatTime(duration)}</span>
      </div>
    </div>
  )
}
