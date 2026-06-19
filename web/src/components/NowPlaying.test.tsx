import { render, screen, fireEvent } from '@testing-library/react'
import { vi } from 'vitest'
import NowPlaying from './NowPlaying'
import type { Song } from '../types'

const song: Song = {
  id: '1',
  title: 'Test Song',
  artist: 'Test Artist',
  coverUrl: 'https://example.com/cover.jpg',
  durationSeconds: 200,
}

const defaultProps = {
  song,
  isPlaying: true,
  currentTime: 60,
  duration: 200,
  volume: 0.8,
  onTogglePlay: vi.fn(),
  onPrev: vi.fn(),
  onNext: vi.fn(),
  onSeek: vi.fn(),
  onVolumeChange: vi.fn(),
}

describe('NowPlaying', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders song title and artist', () => {
    render(<NowPlaying {...defaultProps} />)
    expect(screen.getByText('Test Song')).toBeInTheDocument()
    expect(screen.getByText('Test Artist')).toBeInTheDocument()
  })

  it('renders cover image when coverUrl is provided', () => {
    render(<NowPlaying {...defaultProps} />)
    const img = screen.getByAltText('Test Song')
    expect(img).toBeInTheDocument()
    expect(img.getAttribute('src')).toBe('https://example.com/cover.jpg')
  })

  it('renders Music icon placeholder when no coverUrl', () => {
    render(<NowPlaying {...defaultProps} song={{ ...song, coverUrl: undefined }} />)
    expect(screen.queryByAltText('Test Song')).not.toBeInTheDocument()
  })

  it('renders playback control buttons', () => {
    render(<NowPlaying {...defaultProps} />)
    expect(screen.getByLabelText('Previous')).toBeInTheDocument()
    expect(screen.getByLabelText('Pause')).toBeInTheDocument()
    expect(screen.getByLabelText('Next')).toBeInTheDocument()
  })

  it('shows Play icon and label when not playing', () => {
    render(<NowPlaying {...defaultProps} isPlaying={false} />)
    expect(screen.getByLabelText('Play')).toBeInTheDocument()
  })

  it('calls onTogglePlay when play/pause button clicked', () => {
    render(<NowPlaying {...defaultProps} />)
    fireEvent.click(screen.getByLabelText('Pause'))
    expect(defaultProps.onTogglePlay).toHaveBeenCalledTimes(1)
  })

  it('calls onPrev when previous button clicked', () => {
    render(<NowPlaying {...defaultProps} />)
    fireEvent.click(screen.getByLabelText('Previous'))
    expect(defaultProps.onPrev).toHaveBeenCalledTimes(1)
  })

  it('calls onNext when next button clicked', () => {
    render(<NowPlaying {...defaultProps} />)
    fireEvent.click(screen.getByLabelText('Next'))
    expect(defaultProps.onNext).toHaveBeenCalledTimes(1)
  })

  it('renders progress bar with current time and duration', () => {
    render(<NowPlaying {...defaultProps} />)
    expect(screen.getByText('1:00')).toBeInTheDocument()
    expect(screen.getByText('3:20')).toBeInTheDocument()
  })

  it('calls onSeek when progress slider changes', () => {
    render(<NowPlaying {...defaultProps} />)
    const slider = screen.getByRole('slider', { name: 'Seek' })
    fireEvent.change(slider, { target: { value: '120' } })
    expect(defaultProps.onSeek).toHaveBeenCalledWith(120)
  })

  it('renders volume slider', () => {
    render(<NowPlaying {...defaultProps} />)
    const volSlider = screen.getByRole('slider', { name: 'Volume' })
    expect(volSlider).toBeInTheDocument()
  })

  it('calls onVolumeChange when volume slider changes', () => {
    render(<NowPlaying {...defaultProps} />)
    const volSlider = screen.getByRole('slider', { name: 'Volume' })
    fireEvent.change(volSlider, { target: { value: '0.5' } })
    expect(defaultProps.onVolumeChange).toHaveBeenCalledWith(0.5)
  })

  it('renders placeholder when song is null', () => {
    render(<NowPlaying {...defaultProps} song={null} />)
    expect(screen.getByText('No track selected')).toBeInTheDocument()
  })

  it('does not render controls when song is null', () => {
    render(<NowPlaying {...defaultProps} song={null} />)
    expect(screen.queryByLabelText('Play')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Previous')).not.toBeInTheDocument()
  })
})
