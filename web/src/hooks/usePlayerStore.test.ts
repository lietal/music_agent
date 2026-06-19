import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { usePlayerStore } from './usePlayerStore'

vi.mock('../api/player', () => ({
  getPlayUrl: vi.fn(),
  getStreamUrl: vi.fn(),
  getLyrics: vi.fn(),
}))

import { getPlayUrl, getStreamUrl, getLyrics } from '../api/player'

const mockSong: import('../types').Song = { id: 'qqmusic:001', title: 'Test Song', artist: 'Test Artist' }
const mockSong2: import('../types').Song = { id: 'qqmusic:002', title: 'Song 2', artist: 'Artist 2' }

describe('usePlayerStore', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    vi.mocked(getStreamUrl).mockReturnValue('/api/player/stream/fallback')
    vi.mocked(getPlayUrl).mockResolvedValue({
      song_id: 'qqmusic:001',
      url: 'https://cdr.example.com/song.m4a',
      expires_in_seconds: 3600,
      source: 'cdn',
    })
    vi.mocked(getLyrics).mockResolvedValue({
      song_id: 'qqmusic:001',
      plain_text: 'test',
      synced_text: '[00:01.00]first line\n[00:02.00]second line',
    })
  })

  it('initial state has no current song', () => {
    const { result } = renderHook(() => usePlayerStore())
    expect(result.current.state.currentSong).toBeNull()
    expect(result.current.state.queue).toEqual([])
    expect(result.current.state.isPlaying).toBe(false)
  })

  it('play sets current song and queue', async () => {
    const { result } = renderHook(() => usePlayerStore())
    await act(() => result.current.play(mockSong))
    expect(result.current.state.currentSong?.id).toBe('qqmusic:001')
    expect(result.current.state.queue).toHaveLength(1)
    expect(result.current.state.queue[0].id).toBe('qqmusic:001')
    expect(result.current.state.urlSource).toBe('cdn')
  })

  it('addToQueue appends song to queue', async () => {
    const { result } = renderHook(() => usePlayerStore())
    await act(() => result.current.addToQueue(mockSong))
    await act(() => result.current.addToQueue(mockSong2))
    expect(result.current.state.queue).toHaveLength(2)
    expect(result.current.state.queue[1].id).toBe('qqmusic:002')
  })

  it('removeFromQueue removes song at index', async () => {
    const { result } = renderHook(() => usePlayerStore())
    await act(() => result.current.addToQueue(mockSong))
    await act(() => result.current.addToQueue(mockSong2))
    await act(() => result.current.removeFromQueue(0))
    expect(result.current.state.queue).toHaveLength(1)
    expect(result.current.state.queue[0].id).toBe('qqmusic:002')
  })

  it('clearQueue empties the queue', async () => {
    const { result } = renderHook(() => usePlayerStore())
    await act(() => result.current.addToQueue(mockSong))
    await act(() => result.current.addToQueue(mockSong2))
    await act(() => result.current.clearQueue())
    expect(result.current.state.queue).toHaveLength(0)
  })

  it('togglePanel opens and closes panel', () => {
    const { result } = renderHook(() => usePlayerStore())
    expect(result.current.state.panelOpen).toBe(false)
    act(() => result.current.togglePanel())
    expect(result.current.state.panelOpen).toBe(true)
    act(() => result.current.togglePanel())
    expect(result.current.state.panelOpen).toBe(false)
  })

  it('setPlaybackMode changes mode', () => {
    const { result } = renderHook(() => usePlayerStore())
    act(() => result.current.setPlaybackMode('repeat_all'))
    expect(result.current.state.playbackMode).toBe('repeat_all')
  })

  it('setVolume changes volume', () => {
    const { result } = renderHook(() => usePlayerStore())
    act(() => result.current.setVolume(0.5))
    expect(result.current.state.volume).toBe(0.5)
  })

  it('seek updates currentTime', () => {
    const { result } = renderHook(() => usePlayerStore())
    act(() => result.current.seek(30))
    expect(result.current.state.currentTime).toBe(30)
  })

  it('persists queue to localStorage on play', async () => {
    const { result } = renderHook(() => usePlayerStore())
    await act(() => result.current.play(mockSong))
    const stored = JSON.parse(localStorage.getItem('player_state') || '{}')
    expect(stored.queue).toHaveLength(1)
    expect(stored.queue[0].id).toBe('qqmusic:001')
  })

  it('falls back to proxy URL when CDN fails', async () => {
    vi.mocked(getPlayUrl).mockRejectedValueOnce(new Error('CDN down'))
    const { result } = renderHook(() => usePlayerStore())
    const failSong = { ...mockSong, id: 'qqmusic:fail' }
    await act(() => result.current.play(failSong))
    expect(result.current.state.urlSource).toBe('proxy')
  })
})
