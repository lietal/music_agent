import { useEffect, useRef } from 'react'
import type { LyricLine } from '../types'

interface LyricsPanelProps {
  lyrics: LyricLine[] | null
  activeIndex: number
  onLyricClick?: (time: number) => void
}

export default function LyricsPanel({ lyrics, activeIndex, onLyricClick }: LyricsPanelProps) {
  const activeRef = useRef<HTMLParagraphElement>(null)

  useEffect(() => {
    activeRef.current?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }, [activeIndex])

  if (!lyrics) {
    return (
      <div className="flex items-center justify-center h-full">
        <p className="text-gray-500">暂无歌词</p>
      </div>
    )
  }

  return (
    <div className="overflow-auto h-full px-4 py-6 space-y-3">
      {lyrics.map((line, idx) => {
        const isActive = idx === activeIndex
        return (
          <p
            key={idx}
            ref={isActive ? activeRef : undefined}
            onClick={() => onLyricClick?.(line.time)}
            className={`cursor-pointer transition-all duration-300 ${
              isActive ? 'text-green-400 text-lg' : 'text-gray-500 text-sm'
            }`}
          >
            {line.text}
          </p>
        )
      })}
    </div>
  )
}
