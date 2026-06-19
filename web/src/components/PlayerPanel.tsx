import { useState } from 'react'
import type { Song, LyricLine } from '../types'
import NowPlaying from './NowPlaying'
import LyricsPanel from './LyricsPanel'
import QueuePanel from './QueuePanel'

type Tab = 'nowplaying' | 'lyrics' | 'queue'

interface PlayerPanelProps {
  currentSong: Song | null
  isPlaying: boolean
  currentTime: number
  duration: number
  volume: number
  lyrics: LyricLine[] | null
  activeLyricIndex: number
  queue: Song[]
  queueIndex: number
  onTogglePlay: () => void
  onPrev: () => void
  onNext: () => void
  onSeek: (s: number) => void
  onVolumeChange: (v: number) => void
  onPlaySong: (song: Song) => void
  onRemoveFromQueue: (index: number) => void
  onClose: () => void
}

const TABS: { key: Tab; label: string }[] = [
  { key: 'nowplaying', label: '正在播放' },
  { key: 'lyrics', label: '歌词' },
  { key: 'queue', label: '队列' },
]

export default function PlayerPanel(props: PlayerPanelProps) {
  const [tab, setTab] = useState<Tab>('nowplaying')

  return (
    <div className="fixed bottom-14 left-0 right-0 bg-gray-900 border-t border-gray-700 z-40" style={{ height: '50vh' }}>
      <div className="flex border-b border-gray-700">
        {TABS.map(t => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`flex-1 py-2 text-sm font-medium transition-colors
              ${tab === t.key ? 'text-green-400 border-b-2 border-green-400' : 'text-gray-500 hover:text-gray-300'}`}
          >
            {t.label}
          </button>
        ))}
        <button onClick={props.onClose} className="px-3 text-gray-500 hover:text-gray-300 text-lg">
          ✕
        </button>
      </div>

      <div className="h-full overflow-hidden">
        {tab === 'nowplaying' && (
          <NowPlaying
            song={props.currentSong}
            isPlaying={props.isPlaying}
            currentTime={props.currentTime}
            duration={props.duration}
            volume={props.volume}
            onTogglePlay={props.onTogglePlay}
            onPrev={props.onPrev}
            onNext={props.onNext}
            onSeek={props.onSeek}
            onVolumeChange={props.onVolumeChange}
          />
        )}
        {tab === 'lyrics' && (
          <LyricsPanel
            lyrics={props.lyrics}
            activeIndex={props.activeLyricIndex}
            onLyricClick={props.onSeek}
          />
        )}
        {tab === 'queue' && (
          <QueuePanel
            queue={props.queue}
            currentIndex={props.queueIndex}
            onPlaySong={props.onPlaySong}
            onRemove={props.onRemoveFromQueue}
          />
        )}
      </div>
    </div>
  )
}
