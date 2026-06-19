import type {
  SongURLResponse,
  LyricsResponse,
  QRCodeResponse,
  QRStatusResponse,
  LoginStatusResponse,
} from '../types'

const BASE = ''

export async function getPlayUrl(songId: string): Promise<SongURLResponse> {
  const res = await fetch(`${BASE}/api/player/url/${encodeURIComponent(songId)}`)
  if (!res.ok) throw new Error(`Failed to get play URL: ${res.status}`)
  return res.json()
}

export async function getLyrics(songId: string): Promise<LyricsResponse> {
  const res = await fetch(`${BASE}/api/player/lyrics/${encodeURIComponent(songId)}`)
  if (!res.ok) throw new Error(`Failed to get lyrics: ${res.status}`)
  return res.json()
}

export function getStreamUrl(songId: string): string {
  return `${BASE}/api/player/stream/${encodeURIComponent(songId)}`
}

export async function getLoginQRCode(): Promise<QRCodeResponse> {
  const res = await fetch(`${BASE}/api/qqmusic/login/qrcode`, { method: 'POST' })
  if (!res.ok) throw new Error(`Failed to get QR code: ${res.status}`)
  return res.json()
}

export async function checkQRStatus(key: string): Promise<QRStatusResponse> {
  const res = await fetch(`${BASE}/api/qqmusic/login/status/${encodeURIComponent(key)}`)
  if (!res.ok) throw new Error(`Failed to check QR status: ${res.status}`)
  return res.json()
}

export async function getLoginStatus(): Promise<LoginStatusResponse> {
  const res = await fetch(`${BASE}/api/qqmusic/login/status`)
  if (!res.ok) throw new Error(`Failed to get login status: ${res.status}`)
  return res.json()
}
