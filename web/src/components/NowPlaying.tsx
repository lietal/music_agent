import { Music, SkipBack, Play, Pause, SkipForward } from 'lucide-react'
import type { Song } from '../types'

interface NowPlayingProps {
  song: Song | null
  isPlaying: boolean
  currentTime: number
  duration: number
  volume: number
  onTogglePlay: () => void
  onPrev: () => void
  onNext: () => void
  onSeek: (s: number) => void
  onVolumeChange: (v: number) => void
}

function formatTime(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${m}:${s.toString().padStart(2, '0')}`
}

export default function NowPlaying({
  song,
  isPlaying,
  currentTime,
  duration,
  volume,
  onTogglePlay,
  onPrev,
  onNext,
  onSeek,
  onVolumeChange,
}: NowPlayingProps) {
  if (!song) {
    return (
      <div className="flex items-center justify-center h-full">
        <p className="text-gray-500 text-sm">No track selected</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col items-center justify-center gap-6 p-6 h-full">
      {song.coverUrl ? (
        <img
          src={song.coverUrl}
          alt={song.title}
          className="w-48 h-48 rounded object-cover"
        />
      ) : (
        <div className="w-48 h-48 rounded bg-gray-700 flex items-center justify-center">
          <Music size={64} className="text-gray-500" />
        </div>
      )}

      <div className="text-center">
        <p className="text-lg text-gray-100 truncate max-w-xs">{song.title}</p>
        <p className="text-sm text-gray-400 truncate max-w-xs">{song.artist}</p>
      </div>

      <div className="flex items-center gap-6">
        <button onClick={onPrev} aria-label="Previous" className="p-2 text-gray-400 hover:text-white">
          <SkipBack size={28} />
        </button>
        <button onClick={onTogglePlay} aria-label={isPlaying ? 'Pause' : 'Play'} className="p-3 bg-indigo-600 rounded-full hover:bg-indigo-500 text-white">
          {isPlaying ? <Pause size={32} /> : <Play size={32} />}
        </button>
        <button onClick={onNext} aria-label="Next" className="p-2 text-gray-400 hover:text-white">
          <SkipForward size={28} />
        </button>
      </div>

      <div className="w-full max-w-sm flex items-center gap-3">
        <span className="text-xs text-gray-400 w-10 text-right">{formatTime(currentTime)}</span>
        <input
          type="range"
          aria-label="Seek"
          min={0}
          max={duration || 1}
          value={currentTime}
          onChange={(e) => onSeek(Number(e.target.value))}
          className="flex-1 h-1 accent-indigo-500"
        />
        <span className="text-xs text-gray-400 w-10">{formatTime(duration)}</span>
      </div>

      <div className="w-full max-w-sm hidden sm:flex items-center gap-3">
        <input
          type="range"
          aria-label="Volume"
          min={0}
          max={1}
          step={0.01}
          value={volume}
          onChange={(e) => onVolumeChange(Number(e.target.value))}
          className="flex-1 h-1 accent-indigo-500"
        />
      </div>
    </div>
  )
}
