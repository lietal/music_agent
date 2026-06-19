import { render, screen, fireEvent } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'
import PlayerBar from './PlayerBar'
import type { Song } from '../types'

const mockSong: Song = {
  id: 'song-1',
  title: 'Song Title',
  artist: 'Artist Name',
  durationSeconds: 200,
}

function renderPlayerBar(currentSong: Song | null = mockSong, isPlaying = false) {
  const props = {
    currentSong,
    isPlaying,
    currentTime: 60,
    duration: 200,
    onTogglePlay: vi.fn(),
    onTogglePanel: vi.fn(),
    onNext: vi.fn(),
  }
  const utils = render(<PlayerBar {...props} />)
  return { ...props, ...utils }
}

describe('PlayerBar', () => {
  it('renders nothing when currentSong is null', () => {
    const { container } = renderPlayerBar(null)
    expect(container.firstChild).toBeNull()
  })

  it('renders song title and artist when provided', () => {
    renderPlayerBar()
    expect(screen.getByText('Song Title')).toBeInTheDocument()
    expect(screen.getByText('Artist Name')).toBeInTheDocument()
  })

  it('renders progress time in M:SS format', () => {
    renderPlayerBar()
    expect(screen.getByText('1:00')).toBeInTheDocument()
    expect(screen.getByText('3:20')).toBeInTheDocument()
  })

  it('shows play icon when not playing', () => {
    renderPlayerBar(mockSong, false)
    expect(document.querySelector('[data-testid="play-icon"]')).toBeInTheDocument()
    expect(document.querySelector('[data-testid="pause-icon"]')).not.toBeInTheDocument()
  })

  it('shows pause icon when playing', () => {
    renderPlayerBar(mockSong, true)
    expect(document.querySelector('[data-testid="pause-icon"]')).toBeInTheDocument()
    expect(document.querySelector('[data-testid="play-icon"]')).not.toBeInTheDocument()
  })

  it('calls onTogglePlay when play/pause button is clicked', () => {
    const { onTogglePlay } = renderPlayerBar()
    const btn = screen.getByRole('button', { name: /play/i })
    fireEvent.click(btn)
    expect(onTogglePlay).toHaveBeenCalledTimes(1)
  })

  it('calls onTogglePanel when bar is clicked', () => {
    const { onTogglePanel } = renderPlayerBar()
    const bar = screen.getByTestId('player-bar')
    fireEvent.click(bar)
    expect(onTogglePanel).toHaveBeenCalledTimes(1)
  })

  it('calls onNext when skip forward button is clicked', () => {
    const { onNext } = renderPlayerBar()
    const btn = screen.getByRole('button', { name: /next/i })
    fireEvent.click(btn)
    expect(onNext).toHaveBeenCalledTimes(1)
  })
})
