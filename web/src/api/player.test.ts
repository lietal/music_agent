import { describe, it, expect, vi, beforeEach } from 'vitest'
import { getPlayUrl, getLyrics, getStreamUrl, getLoginQRCode, getLoginStatus } from './player'

describe('player API', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('getPlayUrl returns song URL with expiry', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({
        song_id: 'qqmusic:001abc',
        url: 'https://isure.stream.qqmusic.qq.com/C400001abc.m4a',
        expires_in_seconds: 3600,
        source: 'cdn',
      }),
    } as Response)

    const result = await getPlayUrl('qqmusic:001abc')
    expect(result.url).toBe('https://isure.stream.qqmusic.qq.com/C400001abc.m4a')
    expect(result.expires_in_seconds).toBe(3600)
    expect(result.source).toBe('cdn')
  })

  it('getLyrics returns lyrics data', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({
        song_id: 'qqmusic:001abc',
        plain_text: 'test lyrics',
        synced_text: '[00:00.00]test lyrics',
      }),
    } as Response)

    const result = await getLyrics('qqmusic:001abc')
    expect(result.plain_text).toBe('test lyrics')
    expect(result.synced_text).toBe('[00:00.00]test lyrics')
  })

  it('getStreamUrl returns proxy stream URL', () => {
    const url = getStreamUrl('qqmusic:001abc')
    expect(url).toContain('/api/player/stream/qqmusic%3A001abc')
  })

  it('getLoginQRCode returns QR code data', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ qrcode_url: 'https://qr.example.com/abc', key: 'key_123' }),
    } as Response)

    const result = await getLoginQRCode()
    expect(result.qrcode_url).toBe('https://qr.example.com/abc')
    expect(result.key).toBe('key_123')
  })

  it('getLoginStatus returns logged out state', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ logged_in: false }),
    } as Response)

    const result = await getLoginStatus()
    expect(result.logged_in).toBe(false)
  })

  it('getPlayUrl throws on non-ok response', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce({
      ok: false,
      status: 500,
    } as Response)

    await expect(getPlayUrl('bad-id')).rejects.toThrow('Failed to get play URL')
  })
})
