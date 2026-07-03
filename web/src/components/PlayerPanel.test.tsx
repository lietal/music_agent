import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import PlayerPanel from './PlayerPanel'

const defaultProps = {
  currentSong: null,
  isPlaying: false,
  currentTime: 0,
  duration: 0,
  volume: 0.7,
  lyrics: null,
  activeLyricIndex: 0,
  queue: [],
  queueIndex: 0,
  onTogglePlay: vi.fn(),
  onPrev: vi.fn(),
  onNext: vi.fn(),
  onSeek: vi.fn(),
  onVolumeChange: vi.fn(),
  onPlaySong: vi.fn(),
  onRemoveFromQueue: vi.fn(),
  onClose: vi.fn(),
}

describe('PlayerPanel', () => {
  it('renders tab buttons', () => {
    render(<PlayerPanel {...defaultProps} />)
    expect(screen.getByText('正在播放')).toBeDefined()
    expect(screen.getByText('歌词')).toBeDefined()
    expect(screen.getByText('队列')).toBeDefined()
  })

  it('switches to lyrics tab', () => {
    render(<PlayerPanel {...defaultProps} />)
    fireEvent.click(screen.getByText('歌词'))
    expect(screen.getByText('暂无歌词')).toBeDefined()
  })

  it('switches to queue tab', () => {
    render(<PlayerPanel {...defaultProps} />)
    fireEvent.click(screen.getByText('队列'))
    expect(screen.getByText('队列为空')).toBeDefined()
  })

  it('calls onClose when close button is clicked', () => {
    render(<PlayerPanel {...defaultProps} />)
    fireEvent.click(screen.getByText('✕'))
    expect(defaultProps.onClose).toHaveBeenCalled()
  })
})
