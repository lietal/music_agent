import { useState, useCallback, useEffect, useRef } from 'react'
import type { Song, PlayerState, PlaybackMode, LyricLine } from '../types'
import { getPlayUrl, getStreamUrl, getLyrics } from '../api/player'

const STORAGE_KEY = 'player_state'
const URL_CACHE: Record<string, { url: string; expiresAt: number }> = {}

const DEFAULT_STATE: PlayerState = {
  currentSong: null,
  queue: [],
  queueIndex: 0,
  isPlaying: false,
  currentTime: 0,
  duration: 0,
  volume: 0.7,
  playbackMode: 'sequential',
  urlSource: null,
  urlExpiresAt: null,
  lyrics: null,
  activeLyricIndex: 0,
  panelOpen: false,
}

function loadPersistedState(): Partial<PlayerState> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) return JSON.parse(raw)
  } catch {
    // ignore corrupted data
  }
  return {}
}

function persistState(state: PlayerState) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({
      queue: state.queue,
      queueIndex: state.queueIndex,
      playbackMode: state.playbackMode,
      volume: state.volume,
    }))
  } catch {
    // ignore quota errors
  }
}

async function ensurePlayableURL(song: Song): Promise<string> {
  const cache = URL_CACHE[song.id]
  if (cache && Date.now() < cache.expiresAt - 60000) {
    return cache.url
  }

  try {
    const resp = await getPlayUrl(song.id)
    if (resp.url && resp.source === 'cdn') {
      URL_CACHE[song.id] = { url: resp.url, expiresAt: Date.now() + resp.expires_in_seconds * 1000 }
      return resp.url
    }
  } catch {
    // fall through to proxy
  }

  return getStreamUrl(song.id)
}

function parseSyncedLyrics(synced: string): LyricLine[] {
  const lines = synced.split('\n')
  const result: LyricLine[] = []
  const re = /\[(\d{2}):(\d{2})\.(\d{2,3})\](.*)/
  for (const line of lines) {
    const m = line.match(re)
    if (m) {
      const mins = parseInt(m[1])
      const secs = parseInt(m[2])
      const ms = parseInt(m[3].padEnd(3, '0'))
      result.push({
        time: mins * 60 + secs + ms / 1000,
        text: m[4].trim(),
      })
    }
  }
  return result
}

export function usePlayerStore() {
  const [state, setState] = useState<PlayerState>(() => ({
    ...DEFAULT_STATE,
    ...loadPersistedState(),
  }))
  const audioRef = useRef<HTMLAudioElement | null>(null)

  useEffect(() => {
    if (typeof window === 'undefined') return
    if (!audioRef.current) {
      audioRef.current = new Audio()
      audioRef.current.volume = state.volume
    }
    const el = audioRef.current

    const onTime = () => setState(s => ({ ...s, currentTime: el.currentTime }))
    const onDur = () => setState(s => ({ ...s, duration: el.duration || 0 }))
    const onEnded = () => setState(s => ({ ...s, isPlaying: false }))
    const onPlay = () => setState(s => ({ ...s, isPlaying: true }))
    const onPause = () => setState(s => ({ ...s, isPlaying: false }))

    el.addEventListener('timeupdate', onTime)
    el.addEventListener('durationchange', onDur)
    el.addEventListener('ended', onEnded)
    el.addEventListener('play', onPlay)
    el.addEventListener('pause', onPause)

    return () => {
      el.removeEventListener('timeupdate', onTime)
      el.removeEventListener('durationchange', onDur)
      el.removeEventListener('ended', onEnded)
      el.removeEventListener('play', onPlay)
      el.removeEventListener('pause', onPause)
    }
  }, [])

  useEffect(() => {
    persistState(state)
  }, [state.queue, state.queueIndex, state.playbackMode, state.volume])

  const play = useCallback(async (song: Song) => {
    if (!audioRef.current) return

    const url = await ensurePlayableURL(song)
    const isProxy = url.includes('/api/player/stream/')

    audioRef.current.src = url
    audioRef.current.play()?.catch(() => {})

    setState(s => ({
      ...s,
      currentSong: song,
      queue: [song, ...s.queue.filter(q => q.id !== song.id)],
      queueIndex: 0,
      isPlaying: true,
      urlSource: isProxy ? 'proxy' : 'cdn',
      lyrics: null,
      activeLyricIndex: 0,
    }))

    getLyrics(song.id).then(l => {
      setState(s => ({ ...s, lyrics: parseSyncedLyrics(l.synced_text) }))
    }).catch(() => {})
  }, [])

  const togglePlay = useCallback(() => {
    if (!audioRef.current || !state.currentSong) return
    if (state.isPlaying) {
      audioRef.current.pause()
    } else {
      audioRef.current.play()?.catch(() => {})
    }
  }, [state.isPlaying, state.currentSong])

  const next = useCallback(() => {
    setState(s => {
      const nextIdx = s.queueIndex + 1
      if (nextIdx >= s.queue.length) return s
      const nextSong = s.queue[nextIdx]
      setTimeout(() => play(nextSong), 0)
      return { ...s, queueIndex: nextIdx }
    })
  }, [play])

  const prev = useCallback(() => {
    setState(s => {
      const prevIdx = Math.max(0, s.queueIndex - 1)
      const prevSong = s.queue[prevIdx]
      setTimeout(() => play(prevSong), 0)
      return { ...s, queueIndex: prevIdx }
    })
  }, [play])

  const seek = useCallback((seconds: number) => {
    if (audioRef.current) {
      audioRef.current.currentTime = seconds
    }
    setState(s => ({ ...s, currentTime: seconds }))
  }, [])

  const setVolume = useCallback((v: number) => {
    if (audioRef.current) audioRef.current.volume = v
    setState(s => ({ ...s, volume: v }))
  }, [])

  const addToQueue = useCallback((song: Song) => {
    setState(s => ({
      ...s,
      queue: [...s.queue.filter(q => q.id !== song.id), song],
    }))
  }, [])

  const removeFromQueue = useCallback((index: number) => {
    setState(s => {
      const newQueue = [...s.queue]
      newQueue.splice(index, 1)
      return { ...s, queue: newQueue }
    })
  }, [])

  const clearQueue = useCallback(() => {
    setState(s => ({ ...s, queue: [] }))
  }, [])

  const setPlaybackMode = useCallback((mode: PlaybackMode) => {
    setState(s => ({ ...s, playbackMode: mode }))
  }, [])

  const togglePanel = useCallback(() => {
    setState(s => ({ ...s, panelOpen: !s.panelOpen }))
  }, [])

  return {
    state,
    play,
    togglePlay,
    next,
    prev,
    seek,
    setVolume,
    addToQueue,
    removeFromQueue,
    clearQueue,
    setPlaybackMode,
    togglePanel,
  }
}
