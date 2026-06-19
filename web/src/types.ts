export interface Song {
  id: string
  title: string
  artist: string
  album?: string
  coverUrl?: string
  durationSeconds?: number
}

export interface ChatMessage {
  id: string
  role: 'user' | 'agent'
  content: string
  songs?: Song[]
  timestamp: number
}

export interface TraceStep {
  id: string
  type: 'plan' | 'tool_start' | 'tool_done'
  name: string
  status: 'running' | 'done' | 'error'
  details?: string
}

// ── Playback types ──

export type PlaybackMode = 'sequential' | 'repeat_one' | 'repeat_all' | 'shuffle'

export interface LyricLine {
  time: number
  text: string
}

export interface PlayerState {
  currentSong: Song | null
  queue: Song[]
  queueIndex: number
  isPlaying: boolean
  currentTime: number
  duration: number
  volume: number
  playbackMode: PlaybackMode
  urlSource: 'cdn' | 'proxy' | null
  urlExpiresAt: number | null
  lyrics: LyricLine[] | null
  activeLyricIndex: number
  panelOpen: boolean
}

// ── API response types ──

export interface SongURLResponse {
  song_id: string
  url: string
  expires_in_seconds: number
  source: 'cdn' | 'unavailable'
}

export interface LyricsResponse {
  song_id: string
  plain_text: string
  synced_text: string
}

export interface QRCodeResponse {
  qrcode_url: string
  key: string
}

export interface QRStatusResponse {
  status: 'pending' | 'scanned' | 'confirmed' | 'expired'
  music_id?: string
  music_key?: string
  openid?: string
  user_name?: string
  avatar_url?: string
  token?: string
  user?: {
    user_id: string
    provider: string
    display_name: string
  }
}

export interface LoginStatusResponse {
  logged_in: boolean
  user_name?: string
}
